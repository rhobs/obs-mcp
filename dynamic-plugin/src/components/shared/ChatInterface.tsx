import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSendMessage, useMessages } from '@redhat-cloud-services/ai-react-state';
import {
  Chatbot,
  ChatbotDisplayMode,
  ChatbotContent,
  ChatbotWelcomePrompt,
  ChatbotFooter,
  MessageBox,
  Message,
  MessageBar,
} from '@patternfly/chatbot';

interface ChatInterfaceProps {
  welcomeTitle?: string;
  welcomeDescription?: string;
  placeholder?: string;
}

export function ChatInterface({
  welcomeTitle,
  welcomeDescription,
  placeholder,
}: ChatInterfaceProps) {
  const { t } = useTranslation('plugin__genie-plugin');
  const [isLoading, setIsLoading] = useState(false);
  const bottomRef = React.createRef<HTMLDivElement>();

  const sendMessage = useSendMessage();
  const messages = useMessages();
  useEffect(() => {
    setTimeout(() => {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    });
  }, [messages]);

  // Convert Red Hat Cloud Services messages to PatternFly format
  const formatMessages = () => {
    return messages.map((msg) => {
      const message = msg as any; // Type assertion for Red Hat Cloud Services message format
      const isBot = message.role === 'bot' || message.role === 'assistant';
      let content = message.answer || message.query || message.message || message.content || '';
      content = content.split('=====The following is the user query that was asked:').pop();
      return (
        <Message
          key={msg.id}
          isLoading={!content}
          name={isBot ? 'Genie' : 'You'}
          role={isBot ? 'bot' : 'user'}
          avatar={
            isBot
              ? 'https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/png/openshift.png'
              : 'https://w7.pngwing.com/pngs/831/88/png-transparent-user-profile-computer-icons-user-interface-mystique-miscellaneous-user-interface-design-smile-thumbnail.png'
          }
          timestamp={new Date(
            message.timestamp || message.createdAt || Date.now(),
          ).toLocaleTimeString()}
          content={content}
        />
      );
    });
  };

  const handlePatternFlySend = async (message: string) => {
    if (!message.trim() || isLoading) return;

    setIsLoading(true);
    try {
      const prompt = `
When adding widgets that are charts, ensure or pass query property to the widget.
Chart widgets do load the data at runtime and do not need any hardcoded data. They do not know how to use them.
Avoid using queries that have invalid characters in them. For example:
      - invalid query: sum by(namespace) (rate(container_cpu_usage_seconds_total{container!\"POD\",container!\"\"}[5m]))
      - valid query: sum by(namespace) (rate(container_cpu_usage_seconds_total{container!="POD", container!=""}[5m]))
The invalid query is using a character ! which is not allowed in PromQL without escaping it with a backslash \. Correct operator for "not equal" is !=.
The quote character " can never follow exclamation !. No queries like that are valid.

You are not allowed to use != in a query. Find alternatives like !~

We want to avoid runtime errors due to invalid queries.

Here is more context about the system:
- The system is OpenShift, a Kubernetes-based container orchestration platform.
- The widgets are part of a dashboard that visualizes various metrics and data related to OpenShift clusters.
- The queries are written in PromQL, the query language for Prometheus, which is commonly used for monitoring in Kubernetes environments.

We also have a NG UI capabilities that can visualize data that is not related to charts. For example, we can show lists, tables, text blocks, and other UI elements.

=====The following is the user query that was asked:
${message}
`;
      await sendMessage(prompt, { stream: true, requestOptions: {} });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Chatbot displayMode={ChatbotDisplayMode.embedded}>
      <ChatbotContent>
        <ChatbotWelcomePrompt
          title={welcomeTitle || t("Hello! I'm Genie")}
          description={welcomeDescription || t('Your AI assistant for OpenShift. Ask me anything!')}
        />
        <MessageBox>
          {formatMessages()}
          <div ref={bottomRef}></div>
        </MessageBox>
      </ChatbotContent>
      <ChatbotFooter>
        <MessageBar
          onSendMessage={handlePatternFlySend}
          placeholder={placeholder || t('Ask me anything about OpenShift...')}
          hasMicrophoneButton={false}
          isSendButtonDisabled={isLoading}
          alwayShowSendButton={true}
        />
      </ChatbotFooter>
    </Chatbot>
  );
}
