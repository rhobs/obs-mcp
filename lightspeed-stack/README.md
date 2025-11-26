# Genie MVP Lightspeed Stack

This guide captures the steps to set up a Lightspeed Stack integrated with multiple MCP servers, including the OBS MCP server, Kube MCP server, and Next Gen UI MCP server.

## Getting Started


### MCP Servers

Login to OCP cluster
```
oc login ...
```

1. Terminal 1: Kube MCP Server
   1. `npx kubernetes-mcp-server@latest --port 8081 --list-output table --read-only --toolsets core`
2. Terminal 2: [OBS MCP](../obs-mcp/README.md)
   1. `cd obs-mcp`
   2. `go run ./cmd/obs-mcp/ --listen 127.0.0.1:9100 --auth-mode kubeconfig --insecure`

3. Terminal 3: Next Gen UI MCP
```sh
   podman run --rm -it -p 9200:9200 \
      -v $PWD/mvp-lightspeed-stack/ngui_openshift_mcp_config.yaml:/opt/app-root/config/ngui_openshift_mcp_config.yaml:z \
      --env MCP_PORT="9200" \
      --env NGUI_MODEL="gpt-4.1-nano" \
      --env NGUI_PROVIDER_API_KEY=$OPENAI_API_KEY \
      --env NGUI_CONFIG_PATH="/opt/app-root/config/ngui_openshift_mcp_config.yaml" \
      --env MCP_TOOLS="generate_ui_component" \
      --env MCP_STRUCTURED_OUTPUT_ENABLED="false" \
      quay.io/next-gen-ui/mcp:dev
```

### Lightspeed Stack

Terminal 4:

#### Podman way
```sh
cd mvp-lightspeed-stack
podman run \
  -p 8080:8080 --rm \
  -v ./lightspeed-stack.yaml:/app-root/lightspeed-stack.yaml:Z \
  -v ./run.yaml:/app-root/run.yaml:Z \
  -e OPENAI_API_KEY=$OPENAI_API_KEY \
  quay.io/lightspeed-core/lightspeed-stack:0.3.0
```
Verify on [http://localhost:8080/v1/models](http://localhost:8080/v1/models).

#### Lightspeed-stack Source Code way

```sh
cd ..
git clone https://github.com/lightspeed-core/lightspeed-stack.git
cp genie-plugin/lightspeed-stack/lightspeed-stack.yaml lightspeed-stack/lightspeed-stack.yaml
cp genie-plugin/lightspeed-stack/run.yaml lightspeed-stack/run.yaml

cd lightspeed-stack
uv sync --group dev --group llslibdev
make run
```

## Test

### Lightspeed Stack setup

1. [http://localhost:8080/v1/models](http://localhost:8080/v1/models) - all models
2. [http://localhost:8080/v1/tools](http://localhost:8080/v1/tools) - all registered tools (MCP servers)

### Questions
1. `who are you?` - in the response you should see "Genie..."
2. `what are my pods in namespace openshift-lightspeed, generate ui` - next gen should be involved.

```sh
curl --request POST \
  --url http://localhost:8080/v1/streaming_query \
  --header 'Content-Type: application/json' \
  --data '{
  "media_type": "application/json",
  "model": "gpt-4o-mini",
  "provider": "openai",
  "query": "what are my pods in namespace openshift-lightspeed, generate ui"
}'
```
