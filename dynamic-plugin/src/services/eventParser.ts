import {
  CreateDashboardEvent,
  CreateDashboardResponse,
  DashboardWidget,
  ManipulateWidgetEvent,
  ManipulateWidgetArgumentsEvent,
  AddWidgetEvent,
  AddWidgetResponse,
  GenerateUIEvent,
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
    event.event === 'tool_call' &&
    event.data &&
    event.data.token &&
    typeof event.data.token === 'object' &&
    event.data.token.tool_name === 'create_dashboard' &&
    event.data.token.response
  );
}

export function parseCreateDashboardEvent(
  event: CreateDashboardEvent,
): CreateDashboardResponse | null {
  if (!isCreateDashboardEvent(event)) {
    return null;
  }

  try {
    let response = event.data.token.response;
    console.log({ response });

    // Handle case where response is a JSON string
    if (typeof response === 'string') {
      response = JSON.parse(response);
    }

    // Validate the parsed response has required properties
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

    return response as CreateDashboardResponse;
  } catch (error) {
    console.error('Failed to parse dashboard response:', error);
    return null;
  }
}

export function isAddWidgetEvent(event: any): event is AddWidgetEvent {
  return (
    event &&
    event.event === 'tool_call' &&
    event.data &&
    event.data.token &&
    typeof event.data.token === 'object' &&
    event.data.token.tool_name === 'add_widget' &&
    event.data.token.response
  );
}

export function parseAddWidgetEvent(event: AddWidgetEvent): AddWidgetResponse | null {
  if (!isAddWidgetEvent(event)) {
    return null;
  }

  try {
    let response = (event.data.token as any).response;
    console.log('Parsing add widget event:', response);

    // Handle case where response is a JSON string
    if (typeof response === 'string') {
      response = JSON.parse(response);
    }

    // Validate the parsed response has required properties
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
      if (widget.componentType === 'chart') {
        // Extract title from the response message if not already present
        let title = widget.props.title;
        if (!title && response.message) {
          // Try to extract title from message like "Added chart widget 'Chart showing cluster CPU usage over the past 15 minutes'"
          const titleMatch = response.message.match(/'([^']+)'/);
          if (titleMatch) {
            title = titleMatch[1];
          }
        }

        widget.props = {
          ...widget.props,
          persesComponent: 'PersesTimeSeries',
          ...(title && { title }),
        };
      }
    });

    return response as AddWidgetResponse;
  } catch (error) {
    console.error('Failed to parse add widget event:', error);
    return null;
  }
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
  if (event.data.token.status == 'error') {
    console.error('Error in NGUI TOOL', event.data.token);
    return null;
  }

  const ngui_response: MCPGenerateUIOutput = JSON.parse(event.data.token.artifact);
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
