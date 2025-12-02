# OBS MCP Server

## About

This is an [MCP](https://modelcontextprotocol.io/introduction) server to allow LLMs to interact with a running [Prometheus](https://prometheus.io/) instance via the API.

## Development Quickstart

The easiest way to get the obs-mcp connected to the cluster is via a kubeconfig:

 1. log into your OpenShift cluster
 2. run the server with
 ```sh
 go run ./cmd/obs-mcp/ --listen 127.0.0.1:9100 --auth-mode kubeconfig --insecure
 ```

This will connect the obs-mcp to the thanos querier running in the cluster.

This procedure would not work if you're not using token-based auth (`oc whoami -t` to validate).
In that case, consider using serviceaccount + token auth. Alternatively, follow the procedure bellow.

NOTE: It is possible to hit the ground running locally as well:
```shell
helm install prometheus-community/prometheus --name-template <prefix> # sets up Prometheus (and exporters) on your local single-node k8s cluster
export POD_NAME=$(kubectl get pods --namespace default -l "app.kubernetes.io/name=alertmanager,app.kubernetes.io/instance=local" -o jsonpath="{.items[0].metadata.name}") && kubectl --namespace default port-forward $POD_NAME 9090
go run ./cmd/obs-mcp/ --auth-mode header --insecure --listen :9100 
```

### Port-forwarding alternative

This scenario opens a local port via port-forward that the obs-mcp will connect to:

 1. log into your OpenShift cluster
 2. port-forward the OpenShift thanos instance to a local port
``` sh
PROM_POD=$(kubectl get pods -n openshift-monitoring -l app.kubernetes.io/instance=thanos-querier -o jsonpath="{.items[0].met
adata.name}")
kubectl port-forward -n openshift-monitoring $PROM_POD 9090:9090
```
 3. run the server with

```sh
PROMETHEUS_URL=http://localhost:9090 go run ./cmd/obs-mcp/ --listen 127.0.0.1:9100 --auth-mode header
```
