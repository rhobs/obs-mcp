# obs-mcp

This is an [MCP](https://modelcontextprotocol.io/introduction) server to allow LLMs to interact with a running [Prometheus](https://prometheus.io/) instance via the API.

> [!NOTE]
> This project is moved from [jhadvig/genie-plugin](https://github.com/jhadvig/genie-plugin/tree/main/obs-mcp) preserving the history of commits.

## Pre-requisites

Before starting, ensure you have the following:

- A working Lightspeed Core-based server with the capability to integrate the MCP server located in the `obs-mcp` directory of this project.
- Access to a model capable of tool calling. This project was tested with `gpt-4o-mini`.
- An environment where both Node.js (version 20 or higher) and Golang are available. Using `nvm` (Node Version Manager) and `gvm` (Go Version Manager) is recommended for managing multiple versions.
- Access to an OpenShift Container Platform (OCP) cluster with the monitoring plugin installed.

## Getting Started

See the [obs-mcp README](./obs-mcp/README.md)
