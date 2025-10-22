import {
  CreateDashboardEvent,
  CreateDashboardResponse,
  DashboardWidget,
  ManipulateWidgetEvent,
  ManipulateWidgetArgumentsEvent,
  AddWidgetEvent,
  AddWidgetResponse,
  GenerateUIEvent,
  SetDashboardMetadataEvent,
  ManipulateWidgetResponse,
} from '../types/dashboard';
import { MCPGenerateUIOutput, ComponentDataHandBuildComponent } from '../types/ngui';

export type ToolCallEvent =
  | CreateDashboardEvent
  | ManipulateWidgetEvent
  | ManipulateWidgetArgumentsEvent
  | AddWidgetEvent
  | GenerateUIEvent
  | any;

export function isCreateDashboardEvent(event: any): event is CreateDashboardEvent {
  return (
    event &&
    event.event === 'tool_result' &&
    event.data &&
    event.data.token &&
    event.data.token.response &&
    typeof event.data.token.response === 'object' &&
    event.data.token.response.operation === 'create_dashboard'
  );
}

export function parseCreateDashboardEvent(
  event: CreateDashboardEvent,
): CreateDashboardResponse | null {
  if (!isCreateDashboardEvent(event)) {
    return null;
  }

  const response = event.data.token.response;
  console.log('Parsing dashboard event:', { response });

  // Validate the response has required properties
  if (!response || typeof response !== 'object') {
    console.warn('Invalid dashboard response: not an object', response);
    return null;
  }

  if (!response.layout || !response.layout.name) {
    console.warn('Invalid dashboard response: missing layout.name', response);
    return null;
  }

  // Widgets array is now optional (empty dashboards don't have widgets)
  if (response.widgets && !Array.isArray(response.widgets)) {
    console.warn('Invalid dashboard response: widgets is not an array', response);
    return null;
  }

  // Ensure widgets array exists even if empty
  if (!response.widgets) {
    response.widgets = [];
  }

  return response;
}

export function isAddWidgetEvent(event: any): event is AddWidgetEvent {
  return (
    event &&
    event.event === 'tool_result' &&
    event.data &&
    event.data.token &&
    event.data.token.response &&
    typeof event.data.token.response === 'object' &&
    event.data.token.response.operation === 'add_widget'
  );
}

export function parseAddWidgetEvent(event: AddWidgetEvent): AddWidgetResponse | null {
  if (!isAddWidgetEvent(event)) {
    return null;
  }

  const response = event.data.token.response;
  console.log('Parsing add widget event:', response);

  // Validate the response has required properties
  if (!response || typeof response !== 'object') {
    console.warn('Invalid add widget response: not an object', response);
    return null;
  }

  if (!Array.isArray(response.widgets)) {
    console.warn('Invalid add widget response: widgets is not an array', response);
    return null;
  }

  // Add Perses component type for chart widgets and extract title from message if needed
  response.widgets.forEach((widget) => {
    if (widget.componentType) {
      // Extract title from the response message if not already present
      let title = widget.props?.title;
      if (!title && response.message) {
        // Try to extract title from message like "Added TimeSeriesChart widget 'Chart showing CPU usage in namespaces over the last hour'"
        const titleMatch = response.message.match(/'([^']+)'/);
        if (titleMatch) {
          title = titleMatch[1];
        }
      }

      widget.props = {
        ...widget.props,
        ...(title && { title }),
      };
    }
  });

  return response;
}

export function isManipulateWidgetEvent(event: any): event is ManipulateWidgetEvent {
  return (
    event &&
    event.event === 'tool_result' &&
    event.data &&
    event.data.token &&
    event.data.token.response &&
    typeof event.data.token.response === 'object' &&
    event.data.token.response.operation === 'manipulate_widget'
  );
}

export function parseManipulateWidgetEvent(
  event: ManipulateWidgetEvent,
): ManipulateWidgetResponse | null {
  if (!isManipulateWidgetEvent(event)) {
    return null;
  }

  const response = event.data.token.response;

  // Validate the response has required properties
  if (!response || typeof response !== 'object') {
    console.warn('Invalid manipulate widget response: not an object', response);
    return null;
  }

  if (!Array.isArray(response.widgets)) {
    console.warn('Invalid manipulate widget response: widgets is not an array', response);
    return null;
  }

  return response;
}

export function isManipulateWidgetArgumentsEvent(
  event: any,
): event is ManipulateWidgetArgumentsEvent {
  return (
    event &&
    event.event === 'tool_call' &&
    event.data &&
    event.data.token &&
    typeof event.data.token === 'object' &&
    event.data.token.tool_name === 'manipulate_widget' &&
    event.data.token.arguments &&
    event.data.token.arguments.widget_id &&
    event.data.token.arguments.operation
  );
}

export function parseManipulateWidgetArgumentsEvent(event: ManipulateWidgetArgumentsEvent): {
  widgetId: string;
  position: { x: number; y: number; w?: number; h?: number };
  operation: string;
} | null {
  if (!isManipulateWidgetArgumentsEvent(event)) {
    return null;
  }

  try {
    const { arguments: args } = event.data.token;
    console.log('Parsing manipulate widget arguments event:', args);

    return {
      widgetId: args.widget_id,
      operation: args.operation,
      position: {
        x: parseInt(args.x, 10),
        y: parseInt(args.y, 10),
        ...(args.w && { w: parseInt(args.w, 10) }),
        ...(args.h && { h: parseInt(args.h, 10) }),
      },
    };
  } catch (error) {
    console.error('Failed to parse manipulate widget arguments event:', error);
    return null;
  }
}

export function isGenerateUIEvent(event: any): event is GenerateUIEvent {
  return (
    event &&
    event.event === 'tool_result' &&
    event.data &&
    event.data.token &&
    typeof event.data.token === 'object' &&
    event.data.token.tool_name === 'generate_ui' &&
    event.data.token.status === 'success' &&
    (event.data.token.artifact || event.data.token.response)
  );
}

export function parseGenerateUIEvent(
  event: GenerateUIEvent,
  dashboardWidgets: DashboardWidget[] | null,
): AddWidgetResponse | null {
  if (!isGenerateUIEvent(event)) {
    return null;
  }

  let ngui_response: MCPGenerateUIOutput;

  // Handle new event format (data.artifact is already parsed object)
  if ('tool_name' in event.data && event.data.tool_name === 'generate_ui') {
    // TypeScript narrowing: we know this is the new format
    const newFormatEvent = event.data as any; // Cast to handle union type
    if (newFormatEvent.status === 'error') {
      console.error('Error in NGUI TOOL (new format)', newFormatEvent);
      return null;
    }
    ngui_response = newFormatEvent.artifact; // Already parsed by client
  }
  // Handle old event format (data.token.artifact) - keeping for backward compatibility
  else if ('token' in event.data && event.data.token) {
    // TypeScript narrowing: we know this is the old format
    const oldFormatEvent = event.data as any; // Cast to handle union type
    if (oldFormatEvent.token.status == 'error') {
      console.error('Error in NGUI TOOL (old format)', oldFormatEvent.token);
      return null;
    }
    ngui_response = JSON.parse(oldFormatEvent.token.artifact);
  } else {
    console.error('No valid artifact found in generate UI event');
    return null;
  }
  const result = {
    widgets: ngui_response.blocks
      .filter((ngui_block) => {
        if (
          dashboardWidgets &&
          dashboardWidgets.find((w) => w.props && w.props.ngui_id === ngui_block.id)
        ) {
          console.log('NGUI component already present in dashboard');
          return false;
        }
        return true;
      })
      .map((ngui_block) => {
        console.log('NGUI BLOCK:', ngui_block);
        const component = JSON.parse(ngui_block.rendering.content);
        console.log('NGUI component:', component);
        if (component.component === 'text' || component.component === 'log') {
          // Custom component
          const hbc: ComponentDataHandBuildComponent = component;
          return {
            componentType: component.component,
            id: ngui_block.id,
            props: {
              title: '',
              ngui_id: hbc.id,
              content: hbc.data,
            },
            position: {
              x: 0,
              y: 0,
              w: 5,
              h: 10,
            },
            breakpoint: 'lg',
          } as DashboardWidget;
        }
        return {
          componentType: 'ngui',
          id: ngui_block.id,
          props: {
            title: component.title,
            ngui_id: ngui_block.id,
            ngui_content: ngui_block.rendering.content,
          },
          position: {
            x: 0,
            y: 0,
            w: 6,
            h: 10,
          },
          breakpoint: 'lg',
        } as DashboardWidget;
      }),
  };

  return result as unknown as AddWidgetResponse;
}

export function extractWidgetsFromDashboard(
  dashboardResponse: CreateDashboardResponse,
): DashboardWidget[] {
  return dashboardResponse.widgets || [];
}

export function isSetDashboardMetadata(event: any): event is SetDashboardMetadataEvent {
  const t = event?.data?.token;
  return !!(event && t && t.tool_name === 'set_dashboard_metadata' && (t.arguments || t.response));
}

export function parseSetDashboardMetadata(
  event: SetDashboardMetadataEvent,
): { layoutId?: string; name?: string; description?: string } | null {
  if (!isSetDashboardMetadata(event)) return null;
  try {
    const token = event.data.token;
    // Prefer optimistic arguments
    const args = token.arguments;
    if (args && (args.name != null || args.description != null)) {
      return {
        layoutId: args.layout_id || args.dashboard_id,
        name: args.name,
        description: args.description,
      };
    }

    // Fallback to response
    const responsePayload = token.response;
    const response =
      typeof responsePayload === 'string' ? JSON.parse(responsePayload) : responsePayload;
    const layout = (response as any)?.layout;
    if (!layout) return null;
    return {
      layoutId: layout.layoutId,
      name: layout.name,
      description: layout.description,
    };
  } catch (error) {
    console.error('Failed to parse set_dashboard_metadata event:', error);
    return null;
  }
}
