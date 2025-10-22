// OLS and LSC APIs have differences. We need a temporary "hybrid" client to handle ideally both until we are switched fully to LSC.

import {
  ClientInitLimitation,
  HandleChunkCallback,
  IAIClient,
  IConversation,
  IConversationMessage,
  IInitErrorResponse,
  IMessageResponse,
  ISimpleStreamingHandler,
  IStreamChunk,
} from '@redhat-cloud-services/ai-client-common';
import {
  isAssistantAnswerEvent,
  isEndEvent,
  isErrorEvent,
  isStartEvent,
  isTokenEvent,
  isToolCallEvent,
  LightspeedClientError,
  LightSpeedCoreAdditionalProperties,
  LightspeedSendMessageOptions,
  LightspeedValidationError,
  LLMRequest,
  LLMResponse,
  StreamingEvent,
} from '@redhat-cloud-services/lightspeed-client';

const TEMP_CONVERSATION_ID = '__temp_lightspeed_conversation__';

class DefaultStreamingHandler implements ISimpleStreamingHandler<string | StreamingEvent> {
  // LSC does not provide message IDs in stream, so we generate one and keep it constant across stream instance
  private messageId = crypto.randomUUID();
  private conversationId = '';
  private additionalAttributes: LightSpeedCoreAdditionalProperties = {
    toolCalls: [],
  };
  private messageBuffer = '';
  private streamPromise: Promise<IMessageResponse<LightSpeedCoreAdditionalProperties>>;

  constructor(
    private response: Response,
    private initialConversationId: string,
    private mediaType: 'text/plain' | 'application/json',
    private handleChunk: HandleChunkCallback<LightSpeedCoreAdditionalProperties>,
  ) {
    // Start processing immediately and store the promise
    this.streamPromise = this.processStream();
  }

  /**
   * Process the entire stream internally
   */
  private async processStream(): Promise<IMessageResponse<LightSpeedCoreAdditionalProperties>> {
    if (!this.response.body) {
      throw new Error('Response body is not available for streaming');
    }

    const reader = this.response.body.getReader();
    const decoder = new TextDecoder();
    this.conversationId = this.initialConversationId;

    try {
      if (this.mediaType === 'application/json') {
        // Process JSON Server-Sent Events
        let textBuffer = '';

        // eslint-disable-next-line no-constant-condition
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          textBuffer += decoder.decode(value, { stream: true });
          const lines = textBuffer.split('\n');
          textBuffer = lines.pop() || '';

          for (const line of lines) {
            if (line.trim() && line.startsWith('data: ')) {
              try {
                const eventData = JSON.parse(line.slice(6));
                this.messageBuffer = this.processChunk(
                  eventData,
                  this.messageBuffer,
                  this.handleChunk,
                );
              } catch (error) {
                console.warn('Failed to parse JSON event:', line, error);
              }
            }
          }
        }
      } else {
        // Process text/plain streaming
        // eslint-disable-next-line no-constant-condition
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          const textChunk = decoder.decode(value, { stream: true });
          this.messageBuffer = this.processChunk(textChunk, this.messageBuffer, this.handleChunk);
        }
      }

      return {
        messageId: this.messageId,
        answer: this.messageBuffer,
        conversationId: this.conversationId,
        additionalAttributes: this.additionalAttributes,
      };
    } catch (error) {
      this.onError?.(error as Error);
      throw error;
    } finally {
      reader.releaseLock();
    }
  }

  /**
   * Get the final result (call this after stream completes)
   */
  async getResult(): Promise<IMessageResponse<LightSpeedCoreAdditionalProperties>> {
    return this.streamPromise;
  }

  /**
   * Process a chunk and return updated message buffer
   * Supports both text/plain and JSON SSE formats
   */
  processChunk(
    chunk: string | StreamingEvent,
    currentBuffer: string,
    handleChunk: HandleChunkCallback<LightSpeedCoreAdditionalProperties>,
  ): string {
    let updatedBuffer = currentBuffer;
    let hasUpdate = false;

    if (typeof chunk === 'string') {
      // Text/plain mode - simple accumulation
      updatedBuffer = currentBuffer + chunk;
      hasUpdate = true;
    } else {
      // JSON mode - parse streaming events
      const result = this.processJsonEvent(chunk, currentBuffer);
      updatedBuffer = result.buffer;
      hasUpdate = result.hasUpdate;

      // Update conversation metadata
      if (result.conversationId) {
        this.conversationId = result.conversationId;
      }
      if (result.additionalAttributes) {
        this.additionalAttributes = {
          ...this.additionalAttributes,
          ...result.additionalAttributes,
          toolCalls: [
            ...(this.additionalAttributes.toolCalls || []),
            ...(result.additionalAttributes.toolCalls || []),
          ],
        };
      }
    }

    // Call the callback with current complete message if there's an update
    if (hasUpdate) {
      const streamChunk: IStreamChunk<LightSpeedCoreAdditionalProperties> = {
        messageId: this.messageId,
        answer: updatedBuffer,
        conversationId: this.conversationId,
        additionalAttributes: this.additionalAttributes,
      };
      handleChunk(streamChunk);
    }

    return updatedBuffer;
  }

  /**
   * Process a JSON streaming event and extract relevant data
   */
  private processJsonEvent(
    event: StreamingEvent,
    currentBuffer: string,
  ): {
    buffer: string;
    hasUpdate: boolean;
    conversationId?: string;
    additionalAttributes: LightSpeedCoreAdditionalProperties;
  } {
    let buffer = currentBuffer;
    let hasUpdate = false;
    let conversationId: string | undefined;
    let additionalAttributes: LightSpeedCoreAdditionalProperties = {};

    // Focus on answer-building events
    if (isTokenEvent(event)) {
      // Accumulate tokens
      buffer += event.data.token;
      hasUpdate = true;
    } else if (isAssistantAnswerEvent(event)) {
      // Complete answer overrides token accumulation
      buffer = event.answer;
      conversationId = event.conversation_id;
      hasUpdate = true;
    } else if (isStartEvent(event)) {
      // Capture conversation ID
      conversationId = event.data.conversation_id;
    } else if (isEndEvent(event)) {
      // Capture final metadata
      additionalAttributes = {
        ...additionalAttributes,
        referencedDocuments: event.data.referenced_documents,
        truncated: event.data.truncated,
        inputTokens: event.data.input_tokens,
        outputTokens: event.data.output_tokens,
        availableQuotas: event.available_quotas as Record<string, number>,
      };
    } else if (isToolCallEvent(event)) {
      // Only process tool calls with role 'tool_execution' - other roles add extra tokens and are not usable
      if (event.data.role === 'tool_execution') {
        hasUpdate = true;
        if (!additionalAttributes.toolCalls) {
          additionalAttributes.toolCalls = [];
        }
        additionalAttributes.toolCalls.push(event);
      }
    } else if (event.event === 'tool_result') {
      // This is a specific tool result event to OLS, it is not currently in LSC as a valid tool event
      hasUpdate = true;
      console.log('Processing tool_result event:', event);
      if (!additionalAttributes.toolCalls) {
        additionalAttributes.toolCalls = [];
      }
      const event_data = (event as any).data;
      let call = {};
      if ((event as any)?.data?.tool_name === 'generate_ui' && (event as any)?.data?.artifact) {
        console.log('Parsing NGUI tool_result event with artifact', {
          d: JSON.parse((event as any).data?.artifact),
        });
        call = {
          event: 'tool_result',
          data: {
            token: {
              tool_name: event_data.tool_name,
              response: JSON.parse((event as any).data?.artifact),
              artifact: event_data.artifact,
              status: event_data.status,
            },
          },
        };
      } else {
        call = {
          event: 'tool_result',
          data: {
            token: {
              tool_name: event_data.tool_name,
              response:
                // send the object directly if it's not a string
                typeof event_data.content === 'string'
                  ? JSON.parse(event_data.content)
                  : event_data.content,
              artifact: event_data.artifact,
              status: event_data.status,
            },
          },
        };
      }
      console.log('Parsed tool call:', call);
      additionalAttributes.toolCalls.push(call);
    } else if (isErrorEvent(event)) {
      // Handle error events
      const error = new Error(event.data.response);
      this.onError?.(error);
      throw error;
    }
    // Ignore other events (tool_call, tool_result, user_question) for now

    return { buffer, hasUpdate, conversationId, additionalAttributes };
  }

  /**
   * Called when an error occurs during streaming
   */
  onError?(error: Error): void {
    console.error('Lightspeed streaming error:', error);
  }
}

export class OLSClient implements IAIClient {
  baseUrl: string;
  fetchFunction: (input: RequestInfo, init?: RequestInit) => Promise<Response>;

  constructor(options: {
    baseUrl: string;
    fetchFunction: (input: RequestInfo, init?: RequestInit) => Promise<Response>;
  }) {
    this.baseUrl = options.baseUrl.startsWith('http')
      ? options.baseUrl
      : `${window.location.origin}${options.baseUrl}`;
    this.fetchFunction = options.fetchFunction;
  }

  async init(): Promise<{
    conversations: IConversation[];
    limitation?: ClientInitLimitation;
    error?: IInitErrorResponse;
  }> {
    // We do not need conversation history, we have the dashboard history
    return {
      conversations: [],
    };
  }

  private buildUrl(path: string, userId?: string): string {
    const url = new URL(this.baseUrl);
    url.pathname = url.pathname.concat(path.replace(/^\//, '')); // ensure single slash
    if (userId) {
      url.searchParams.set('user_id', userId);
    }
    console.log('Built URL:', url.toString());
    return url.toString();
  }

  createNewConversation(): Promise<IConversation> {
    // OLS does not seem to support conversations yet
    return Promise.resolve({
      id: TEMP_CONVERSATION_ID,
      title: 'OLS Conversation',
      createdAt: new Date(),
      locked: false,
    });
  }

  private async handleErrorResponse(response: Response): Promise<never> {
    const status = response.status;
    const statusText = response.statusText;

    try {
      const errorBody = await response.json();

      // Handle validation errors (422) - exact OpenAPI spec format
      if (status === 422 && errorBody.detail && Array.isArray(errorBody.detail)) {
        throw new LightspeedValidationError(errorBody.detail);
      }

      // Handle other error formats based on OpenAPI spec
      let message: string;
      if (typeof errorBody.detail === 'string') {
        // UnauthorizedResponse, ForbiddenResponse format
        message = errorBody.detail;
      } else if (errorBody.detail?.response) {
        // ErrorResponse format
        message = errorBody.detail.response;
      } else if (errorBody.detail?.cause) {
        // ErrorResponse format
        message = errorBody.detail.cause;
      } else if (errorBody.message) {
        // Generic message field
        message = errorBody.message;
      } else {
        message = statusText || 'Unknown error';
      }

      throw new LightspeedClientError(status, statusText, message, response);
    } catch (parseError) {
      // If parseError is our own thrown error, re-throw it
      if (
        parseError instanceof LightspeedClientError ||
        parseError instanceof LightspeedValidationError
      ) {
        throw parseError;
      }

      // If we can't parse the error response, throw a generic error
      throw new LightspeedClientError(
        status,
        statusText,
        `HTTP ${status}: ${statusText}`,
        response,
      );
    }
  }

  private async makeRequest<T>(urlOrPath: string, options: RequestInit): Promise<T> {
    const url = urlOrPath.startsWith('http') ? urlOrPath : `${this.baseUrl}${urlOrPath}`;

    try {
      const response = await this.fetchFunction(url, options);

      if (!response.ok) {
        await this.handleErrorResponse(response);
      }

      // For Response objects (like streaming or metrics), return as-is
      if (urlOrPath.includes('/streaming_query') || urlOrPath.includes('/metrics')) {
        return response as T;
      }

      // For JSON responses, parse and return
      return (await response.json()) as T;
    } catch (error) {
      if (error instanceof LightspeedClientError || error instanceof LightspeedValidationError) {
        throw error;
      }
      throw new LightspeedClientError(
        0,
        'Network Error',
        `Failed to make request to ${url}: ${error}`,
      );
    }
  }

  private generateMessageId(): string {
    return crypto.randomUUID();
  }

  async sendMessage(
    conversationId: string,
    message: string,
    options?: LightspeedSendMessageOptions & {
      userId?: string;
    },
  ): Promise<IMessageResponse<LightSpeedCoreAdditionalProperties>> {
    // Determine media type from options, defaulting to application/json
    const mediaType = options?.mediaType || 'application/json';

    const request: LLMRequest = {
      query: message,
      // Omit conversation_id if it's the temporary ID - let API auto-generate
      conversation_id: conversationId === TEMP_CONVERSATION_ID ? null : conversationId,
      media_type: mediaType,
      attachments: [],
    };

    if (options?.stream) {
      // Streaming request - use self-contained handler approach
      const url = this.buildUrl('/v1/streaming_query', options?.userId);
      const response = await this.makeRequest<Response>(url, {
        method: 'POST',
        body: JSON.stringify(request),
        headers: {
          'Content-Type': 'application/json',
          Accept: mediaType === 'application/json' ? 'application/json' : 'text/plain',
          ...options?.headers,
        },
        signal: options?.signal,
      });

      // Create self-contained streaming handler
      // Always provide handleChunk callback (state manager should provide this)
      const handleChunk = options?.handleChunk || (() => undefined); // fallback for safety
      const handler = new DefaultStreamingHandler(response, conversationId, mediaType, handleChunk);

      return await handler.getResult();
    } else {
      // Non-streaming request
      const url = this.buildUrl('/v1/query', options?.userId);
      const response = await this.makeRequest<LLMResponse>(url, {
        method: 'POST',
        body: JSON.stringify(request),
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
          ...options?.headers,
        },
        signal: options?.signal,
      });

      // Convert LLMResponse to IMessageResponse format for common interface compatibility
      const messageResponse: IMessageResponse<LightSpeedCoreAdditionalProperties> = {
        messageId: this.generateMessageId(),
        answer: response.response,
        date: new Date(),
        conversationId: response.conversation_id,
        additionalAttributes: {
          referencedDocuments: response.referenced_documents,
          truncated: response.truncated,
          inputTokens: response.input_tokens,
          outputTokens: response.output_tokens,
          availableQuotas: response.available_quotas,
          toolCalls: response.tool_calls,
          toolResults: response.tool_results,
        },
      };

      return messageResponse;
    }
  }

  getConversationHistory(): Promise<Omit<IConversationMessage<Record<string, unknown>>, 'role'>[]> {
    // OLS does not seem to support conversations yet
    return Promise.resolve([]);
  }

  healthCheck(): Promise<unknown> {
    return Promise.resolve({});
  }
}
