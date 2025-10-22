## Prerequisites

Before starting, ensure you have the following:

- A working Lightspeed Core-based server with the capability to integrate the MCP server located in the `obs-mcp` directory of this project.
- Access to a model capable of tool calling. This project was tested with `gpt-4o-mini`.
- An environment where both Node.js (version 20 or higher) and Golang are available. Using `nvm` (Node Version Manager) and `gvm` (Go Version Manager) is recommended for managing multiple versions.
- Access to an OpenShift Container Platform (OCP) cluster with the monitoring plugin installed.

## Getting Started

Follow these steps to get up and running:

1. Set up the obs-mcp server. For details, see the [obs-mcp README](./obs-mcp/README.md).
2. Set up the layout-manager mcp server. For details, see the [layout-manager mcp](./layout-manager/README.md)
3. Once the servers are running, connect it to your Lightspeed Core (LSC) instance.
4. Start the console UI: in the `dynamic-plugin` package, run `yarn start-console`.
5. Start the UI plugin by running `yarn start` in the `dynamic-plugin` directory.
6. Open your browser and navigate to `http://localhost:9000/genie/widgets`.


## Getting Started - OLS, Kube MCP, Persers MCP, Next Gen MCP, Layout Manager MCP

1. Perses MCP - [obs-mcp/README.md](./obs-mcp/README.md)
2. Layout Manager MCP - [layout-manager/README.md](./layout-manager/README.md)
3. Kube MCP, NGUI & OLS - [ols-ngui/README.md](./ols-ngui/README.md)
4. Openshift Console - [dynamic-plugin/README.md](./dynamic-plugin/README.md)

Open your browser and navigate to [http://localhost:9000/genie/widgets](http://localhost:9000/genie/widgets).

### Prompt examples

Can you create a new dashboard for monitoring? I'd like to have a basic set of widgets that are displaying charts that monitor CPU, memory, and networking usage. I want to know the overall cluster utilization, by and namespaces.

What are mu namespaces, generate ui

Can you show me memory usage of a genie-plugin namespace? use pie chart. Divide it by pods.