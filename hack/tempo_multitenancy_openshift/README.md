# Tracing Stack
Manifests to setup a tracing stack with OpenTelemetry and Tempo + test traces

## Prerequisites
* installed Tempo Operator
* installed Red Hat build of OpenTelemetry

## Endpoints
* Tempo API: https://tempo-tempo1-gateway-obs-mcp-tracing.apps-crc.testing/api/traces/v1/project1/tempo
```
kubectl create serviceaccount demo
TOKEN=$(kubectl create token demo)
curl -G -k \
  --header "Authorization: Bearer $TOKEN" \
  --data-urlencode 'q={ resource.service.name="article-service" }' \
  https://tempo-tempo1-gateway-obs-mcp-tracing.apps-crc.testing/api/traces/v1/project1/tempo/api/search | jq
```

* Jaeger UI: https://tempo-tempo1-gateway-obs-mcp-tracing.apps-crc.testing/api/traces/v1/project1/search
