import { ActiveDashboardResponse, ListDashboardsResponse } from 'src/types/dashboard';

export class DashboardMCPClient {
  private baseURL: string;
  private requestId = 0;
  private sessionId: string | null = null;

  constructor(baseURL = `${window.location.origin}/api/proxy/plugin/genie-plugin/dashboard-mcp/`) {
    this.baseURL = baseURL;
  }

  async initialize(): Promise<void> {
    if (this.sessionId) return;

    const response = await fetch(this.baseURL, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        jsonrpc: '2.0',
        id: this.requestId++,
        method: 'initialize',
        params: {
          protocolVersion: '2024-11-05',
          capabilities: {},
          clientInfo: {
            name: 'dashboard-frontend-client',
            version: '1.0.0',
          },
        },
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to initialize MCP client: ${response.statusText}`);
    }

    const result = await response.json();
    if (result.error) {
      throw new Error(`MCP initialization error: ${result.error.message}`);
    }

    console.log({ h: Array.from(response.headers.keys()), result });
    // Extract session ID from response header
    this.sessionId = response.headers.get('Mcp-Session-Id');
    if (!this.sessionId) {
      throw new Error('Server did not return a session ID');
    }
  }

  private async callTool<T>(name: string, args: Record<string, any>): Promise<T> {
    console.log('Calling tool:', name, args);
    if (!this.sessionId) {
      console.log('Session ID not found, initializing...');
      try {
        await this.initialize();
      } catch (error) {
        console.error('Failed to initialize MCP client:', error);
      }
    }
    const response = await fetch(this.baseURL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Mcp-Session-Id': this.sessionId!,
      },
      body: JSON.stringify({
        jsonrpc: '2.0',
        id: this.requestId++,
        method: 'tools/call',
        params: {
          name: name,
          arguments: args,
        },
      }),
    });
    console.log({ response });

    if (!response.ok) {
      throw new Error(`Failed to call tool ${name}: ${response.statusText}`);
    }

    const result = await response.json();

    if (result.error) {
      throw new Error(`MCP Error: ${result.error.message}`);
    }

    const content = result?.result?.content?.[0] || result?.result;

    // Check if this is an error response
    if (content?.isError) {
      throw new Error(`MCP Tool Error: ${content?.text || 'Unknown error'}`);
    }

    const text = content?.text;
    if (text) {
      return JSON.parse(text) as T;
    }
    return content;
  }

  async getActiveDashboard(): Promise<ActiveDashboardResponse> {
    return await this.callTool('get_active_dashboard', {});
  }

  async getDashboard(dashboardId: string): Promise<ActiveDashboardResponse> {
    return await this.callTool('get_dashboard', { layout_id: dashboardId });
  }

  async getDashboardById(dashboardId: string): Promise<ActiveDashboardResponse> {
    return await this.callTool('get_dashboard', { layout_id: dashboardId });
  }

  async listDashboards(limit = 50, offset = 0): Promise<ListDashboardsResponse> {
    return await this.callTool('list_dashboards', { limit: String(limit), offset: String(offset) });
  }

  async updateWidgetPositions(layout: any[]): Promise<void> {
    // Update each widget's position individually using manipulate_widget
    // We need to do both move and resize in separate operations
    const promises = layout.flatMap((item) => [
      // Move operation
      this.callTool('manipulate_widget', {
        widget_id: item.i,
        operation: 'move',
        x: String(item.x),
        y: String(item.y),
      }),
      // Resize operation
      this.callTool('manipulate_widget', {
        widget_id: item.i,
        operation: 'resize',
        w: String(item.w),
        h: String(item.h),
      }),
    ]);

    await Promise.all(promises);
  }
}
