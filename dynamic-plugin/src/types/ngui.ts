// https://github.com/RedHat-UX/next-gen-ui-agent/tree/main/spec/mcp

/**
 * MCP Output for Generate UI
 */
export interface MCPGenerateUIOutput {
  blocks: UIBlock[];
  summary: string;
  [k: string]: unknown;
}
/**
 * UI Block model with all details
 */
export interface UIBlock {
  id: string;
  rendering?: Rendition | null;
  [k: string]: unknown;
}
/**
 * Rendition of the component - output of the UI rendering step.
 */
export interface Rendition {
  id: string;
  component_system: string;
  mime_type: string;
  content: string;
  [k: string]: unknown;
}

// https://github.com/RedHat-UX/next-gen-ui-agent/blob/main/spec/component/hand-build-component.schema.json
/**
 * Component Data for HandBuildComponent rendered by hand-build code registered in the renderer for given `component_type`.
 */
export interface ComponentDataHandBuildComponent {
  /**
   * type of the component to be used in renderer to select hand-build rendering implementation
   */
  component: string;
  /**
   * id of the backend data this component is for
   */
  id: string;
  /**
   * JSON backend data to be rendered by the hand-build rendering implementation
   */
  data: {
    [k: string]: unknown;
  };
  [k: string]: unknown;
}
