# OLS & NGUI Integration

## Quickstart

### Prerequisities

1. Login to Openshift Cluster

    `oc login ...`

2. Run Openshift MCP Server with TABLE (default) output

    `npx kubernetes-mcp-server@latest --port 8081 --list-output table --read-only --toolsets core`

3. Run NGUI

From Cluster Pod:

  ```sh
  NGUI_POD=$(kubectl get pods -n openshift-lightspeed -l app.kubernetes.io/name=next-gen-ui-mcp -o jsonpath="{.items[0].metadata.name}")
  kubectl port-forward -n openshift-lightspeed $NGUI_POD 9200:9200
  ```

Or as local image

   ```sh
   export OPENAI_API_KEY="sk-..."
   podman run --rm -it -p 9200:9200 \
      -v $PWD/ols-ngui:/opt/app-root/config:z \
      --env MCP_PORT="9200" \
      --env NGUI_MODEL="gpt-4o-mini" \
      --env NGUI_PROVIDER_API_KEY=$OPENAI_API_KEY \
      --env NGUI_CONFIG_PATH="/opt/app-root/config/ngui_openshift_mcp_config.yaml" \
      --env MCP_TOOLS="generate_ui_multiple_components" \
      --env MCP_STRUCTURED_OUTPUT_ENABLED="false" \
      quay.io/next-gen-ui/mcp:dev
   ``` 

Or from git source:
    
```sh
PYTHONPATH=./libs python libs/next_gen_ui_mcp --provider langchain --model gpt-4o-mini  --port 9200 --transport streamable-http --config-path /Users/lkrzyzan/git/genie/genie-plugin/ols-ngui/ngui_openshift_mcp_config.yaml
```

### Start OLS

1. Clone repo https://github.com/lkrzyzanek/lightspeed-service, branch “ngui-mcp” and install deps.

    ```sh
    cd git/genie
    git clone https://github.com/lkrzyzanek/lightspeed-service.git
    git checkout ngui-mcp
    cd lightspeed-service
    make install-deps 
    ```

2. Copy `olsconfig.yaml`

    ```sh
    cp ../genie-plugin/olsconfig.yaml .
    ```

3. Run OLS

    ```sh
    export OPENAI_API_KEY="sk-..."
    pdm run python runner.py
    ```

## Test

```sh
curl --request POST \
  --url http://localhost:8080/v1/streaming_query \
  --header 'Content-Type: application/json' \
  --header 'User-Agent: insomnium/0.2.3-a' \
  --data '{
  "media_type": "application/json",
  "model": "gpt-4o-mini",
  "provider": "openai",
  "query": "what are my namespaces (and generate ui)?"
}'
```

You can change `"media_type": "application/json",` to `"media_type": "text/plain",`


## Conversation Examples

### Create a new dashboard
```
hi

Create a new empty dashboard called "Trying Stuff #1" and activate it.
```


### Openshift Namespace to Pod

```
hi

Create a new empty dashboard called "Demo dashboard" and activate it.

what are my namespaces, generate ui

what pods are running in openshift-lightspeed namespace, generate ui
    what pods are running in openshift-lightspeed namespace

tell me all details about pod lightspeed-app-server-8d87bd889-rhxm4, generate ui
	generate again the component about that pod
    tell me all details about pod lightspeed-app-server-8d87bd889-rhxm4 in namespace openshift-lightspeed, generate ui

what is restart policy for that pod?
what is restart policy for that pod, generate ui
    ^ This is fully generated one card component

show me logs of pod lightspeed-app-server-8d87bd889-rhxm4 in openshift-lightspeed namespace, container openshift-mcp-server, generate ui
```

### Openshift Lightspeed Service Dashboard

```
hi

Create a new empty dashboard called "Openshift Lightspeed Service" and activate it.

What pods are running in namespace "openshift-lightspeed", generate ui

Show me logs for pod next-gen-ui-mcp-695cbd79bb-npcdm, generate ui
```

### Unknown data - Dashboards

```
hi

Create a new empty dashboard called "My Dashboards" and activate it.


what are my dashboards? generate ui
	what are my dashboards? Include all possible information, generate ui

what are my dashboards? generate ui and use table

what are my dashboards? generate ui and use set of cards

```

### Perses

```
create a dashboard called Libor POC and add a widget showing me the CPU usage for the pods in my openshift-monitoring namespace over the last hour
```