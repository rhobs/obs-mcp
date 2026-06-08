#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

MCP_URL="${MCP_URL:-http://127.0.0.1:9100/mcp}"
LOKI_NS="${LOKI_NS:-obs-mcp-loki}"
STACK_NS="${STACK_NS:-obs-mcp-loki}"
STACK_NAME="${STACK_NAME:-obs-mcp-loki}"
LOKI_TENANT="${LOKI_TENANT:-network}"
NETOBSERV_NS="${NETOBSERV_NS:-netobserv}"
VERIFY_DURATION="${VERIFY_DURATION:-1h}"
LOKI_QUERY="${LOKI_QUERY:-}"
LOKI_STORAGE_CLASS="${LOKI_STORAGE_CLASS:-}"
LOKI_OPERATOR_NS="${LOKI_OPERATOR_NS:-openshift-loki-operator}"
LOKI_OPERATOR_CATALOG="${LOKI_OPERATOR_CATALOG:-}"
LOKI_OPERATOR_SOURCE_NAMESPACE="${LOKI_OPERATOR_SOURCE_NAMESPACE:-}"
LOKI_OPERATOR_CHANNEL="${LOKI_OPERATOR_CHANNEL:-}"
CURL_CONNECT_TIMEOUT="${CURL_CONNECT_TIMEOUT:-5}"
CURL_HEALTH_MAX_TIME="${CURL_HEALTH_MAX_TIME:-10}"
CURL_MCP_MAX_TIME="${CURL_MCP_MAX_TIME:-120}"
LOG_GENERATOR_MAX_ATTEMPTS="${LOG_GENERATOR_MAX_ATTEMPTS:-15}"
LOG_GENERATOR_SLEEP="${LOG_GENERATOR_SLEEP:-4}"
LOKI_QUERY_MAX_ATTEMPTS="${LOKI_QUERY_MAX_ATTEMPTS:-12}"
LOKI_QUERY_SLEEP="${LOKI_QUERY_SLEEP:-5}"

echo "==> Ensuring Loki Operator subscription"
PACKAGE_MANIFEST_JSON="$(oc get packagemanifest -n openshift-marketplace loki-operator -o json 2>/dev/null || true)"
if [[ -z "${PACKAGE_MANIFEST_JSON}" ]]; then
  echo "Unable to find PackageManifest 'loki-operator' in openshift-marketplace."
  echo "Available packages containing 'loki':"
  oc get packagemanifest -n openshift-marketplace -o json | jq -r '.items[].metadata.name' | awk '/loki/'
  exit 1
fi

PM_DEFAULT_CHANNEL="$(echo "${PACKAGE_MANIFEST_JSON}" | jq -r '.status.defaultChannel // empty')"
mapfile -t PM_CHANNELS < <(echo "${PACKAGE_MANIFEST_JSON}" | jq -r '.status.channels[]?.name')

if [[ -z "${LOKI_OPERATOR_CATALOG}" ]]; then
  LOKI_OPERATOR_CATALOG="redhat-operators"
fi
if [[ -z "${LOKI_OPERATOR_SOURCE_NAMESPACE}" ]]; then
  LOKI_OPERATOR_SOURCE_NAMESPACE="openshift-marketplace"
fi

if [[ -z "${LOKI_OPERATOR_CHANNEL}" ]]; then
  # defaultChannel is often "alpha" but Red Hat catalogs expose versioned channels
  # such as stable-6.5; prefer the newest stable-* channel when present.
  stable_channels=()
  for ch in "${PM_CHANNELS[@]}"; do
    if [[ "${ch}" == stable-* ]]; then
      stable_channels+=("${ch}")
    fi
  done
  if [[ "${#stable_channels[@]}" -gt 0 ]]; then
    LOKI_OPERATOR_CHANNEL="$(printf '%s\n' "${stable_channels[@]}" | sort -V | tail -n1)"
  elif [[ -n "${PM_DEFAULT_CHANNEL}" ]]; then
    LOKI_OPERATOR_CHANNEL="${PM_DEFAULT_CHANNEL}"
  elif [[ "${#PM_CHANNELS[@]}" -gt 0 ]]; then
    LOKI_OPERATOR_CHANNEL="${PM_CHANNELS[0]}"
  fi
fi
if [[ -z "${LOKI_OPERATOR_CHANNEL}" ]]; then
  echo "Unable to resolve a valid channel for PackageManifest 'loki-operator'."
  echo "Set LOKI_OPERATOR_CHANNEL explicitly and retry."
  exit 1
fi

if [[ "${#PM_CHANNELS[@]}" -gt 0 ]]; then
  channel_found="false"
  for ch in "${PM_CHANNELS[@]}"; do
    if [[ "${ch}" == "${LOKI_OPERATOR_CHANNEL}" ]]; then
      channel_found="true"
      break
    fi
  done
  if [[ "${channel_found}" != "true" ]]; then
    echo "Channel '${LOKI_OPERATOR_CHANNEL}' is not available for loki-operator."
    echo "Available channels: ${PM_CHANNELS[*]}"
    exit 1
  fi
fi

if [[ -z "${LOKI_OPERATOR_CATALOG}" ]]; then
  echo "Unable to resolve catalog source for PackageManifest 'loki-operator'."
  exit 1
fi

echo "Using Loki operator catalog: ${LOKI_OPERATOR_CATALOG}/${LOKI_OPERATOR_SOURCE_NAMESPACE}"
echo "Using Loki operator channel: ${LOKI_OPERATOR_CHANNEL}"
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: loki-operator
  namespace: ${LOKI_OPERATOR_NS}
spec:
  channel: ${LOKI_OPERATOR_CHANNEL}
  installPlanApproval: Automatic
  name: loki-operator
  source: ${LOKI_OPERATOR_CATALOG}
  sourceNamespace: ${LOKI_OPERATOR_SOURCE_NAMESPACE}
EOF

echo "==> Waiting for Loki operator rollout"
oc wait -n "${LOKI_OPERATOR_NS}" --for=create deployment/loki-operator-controller-manager --timeout=10m
oc -n "${LOKI_OPERATOR_NS}" rollout status deployment/loki-operator-controller-manager --timeout=10m

echo "==> Waiting for LokiStack CRD"
oc wait --for=condition=Established crd/lokistacks.loki.grafana.com --timeout=15m

if [[ -z "${LOKI_STORAGE_CLASS}" ]]; then
  LOKI_STORAGE_CLASS="$(oc get storageclass -o json | jq -r '.items[] | select(.metadata.annotations["storageclass.kubernetes.io/is-default-class"] == "true" or .metadata.annotations["storageclass.beta.kubernetes.io/is-default-class"] == "true") | .metadata.name' | head -n1)"
fi
if [[ -z "${LOKI_STORAGE_CLASS}" ]]; then
  echo "Unable to detect a default StorageClass. Set LOKI_STORAGE_CLASS explicitly and retry."
  echo "Available StorageClasses:"
  oc get storageclass
  exit 1
fi

echo "==> Applying LokiStack (storageClassName=${LOKI_STORAGE_CLASS})"
sed "s|__LOKI_STORAGE_CLASS__|${LOKI_STORAGE_CLASS}|g" "${SCRIPT_DIR}/03_lokistack.yaml" | oc apply -f -

echo "==> Waiting for MinIO and bucket job"
oc -n "${LOKI_NS}" rollout status deployment/minio --timeout=5m
oc wait -n "${LOKI_NS}" --for=condition=complete job/minio-create-loki-buckets --timeout=5m

echo "==> Checking LokiStack availability"
oc get lokistack -n "${STACK_NS}" "${STACK_NAME}" >/dev/null
oc get lokistack -A
echo "==> Waiting for LokiStack Ready condition"
oc wait --for=jsonpath='{.status.conditions[?(@.type=="Ready")].status}'=True \
  "lokistack/${STACK_NAME}" -n "${STACK_NS}" --timeout=10m

GATEWAY_SVC="${STACK_NAME}-gateway-http"
echo "==> Checking gateway service"
oc get svc -n "${STACK_NS}" "${GATEWAY_SVC}" >/dev/null

echo "==> Checking gateway route"
GATEWAY_ROUTE="$(oc get route -n "${STACK_NS}" -o json | jq -r --arg svc "${GATEWAY_SVC}" '.items[] | select(.spec.to.name==$svc) | .metadata.name' | head -n1)"
if [[ -z "${GATEWAY_ROUTE}" ]]; then
  GATEWAY_ROUTE="$(oc get route -n "${STACK_NS}" "${STACK_NAME}" -o jsonpath='{.metadata.name}' 2>/dev/null || true)"
fi
if [[ -n "${GATEWAY_ROUTE}" ]]; then
  echo "Found route ${GATEWAY_ROUTE}"
  oc get route -n "${STACK_NS}" "${GATEWAY_ROUTE}"
else
  echo "WARN: no OpenShift Route for gateway service ${GATEWAY_SVC}."
  echo "      For local obs-mcp without --loki.use-route, port-forward instead:"
  echo "      oc port-forward -n ${STACK_NS} svc/${GATEWAY_SVC} 8080:8080"
fi

echo "==> Checking generator rollout"
oc -n "${LOKI_NS}" rollout status deployment/obs-mcp-log-generator --timeout=2m

echo "==> Waiting for log generator output"
for attempt in $(seq 1 "${LOG_GENERATOR_MAX_ATTEMPTS}"); do
  if oc logs -n "${LOKI_NS}" deployment/obs-mcp-log-generator --tail=20 2>/dev/null | grep -q obs-mcp-loki-hack; then
    echo "OK: log generator is producing lines (attempt ${attempt}/${LOG_GENERATOR_MAX_ATTEMPTS})"
    break
  fi
  if [[ "${attempt}" -eq "${LOG_GENERATOR_MAX_ATTEMPTS}" ]]; then
    echo "ERROR: log generator did not produce expected lines after ${LOG_GENERATOR_MAX_ATTEMPTS} attempts"
    exit 1
  fi
  echo "    Attempt ${attempt}/${LOG_GENERATOR_MAX_ATTEMPTS}: no log lines yet, retrying in ${LOG_GENERATOR_SLEEP}s..."
  sleep "${LOG_GENERATOR_SLEEP}"
done

mcp_health_url="${MCP_URL%/mcp}/health"
if [[ "${MCP_URL}" != */mcp ]]; then
  mcp_health_url="${MCP_URL}/health"
fi

echo "==> Checking obs-mcp health at ${mcp_health_url}"
if ! curl -sf --connect-timeout "${CURL_CONNECT_TIMEOUT}" --max-time "${CURL_HEALTH_MAX_TIME}" "${mcp_health_url}" >/dev/null; then
  echo "ERROR: obs-mcp is not reachable."
  echo "Start it in another terminal, for example:"
  echo "  go run ./cmd/obs-mcp --listen 127.0.0.1:9100 --toolsets logs --auth-mode kubeconfig --insecure --loki.use-route"
  if [[ -z "${GATEWAY_ROUTE}" ]]; then
    echo "If no Route exists, omit --loki.use-route and port-forward Loki:"
    echo "  oc port-forward -n ${STACK_NS} svc/${GATEWAY_SVC} 8080:8080"
    echo "  go run ./cmd/obs-mcp --listen 127.0.0.1:9100 --toolsets logs --auth-mode kubeconfig --insecure --loki-url http://127.0.0.1:8080"
  fi
  exit 1
fi

mcp_parse_sse() {
  local body="$1"
  if [[ "${body}" == *$'\ndata:'* ]] || [[ "${body}" == data:* ]]; then
    printf '%s\n' "${body}" | awk '/^data: / { sub(/^data: /, ""); print; exit }'
    return
  fi
  printf '%s' "${body}"
}

mcp_call_raw() {
  local payload="$1"
  local raw result
  if ! raw="$(curl -sS --connect-timeout "${CURL_CONNECT_TIMEOUT}" --max-time "${CURL_MCP_MAX_TIME}" \
    -X POST "${MCP_URL}" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json, text/event-stream" \
    -d "${payload}")"; then
    echo "ERROR: MCP request to ${MCP_URL} failed" >&2
    exit 1
  fi
  result="$(mcp_parse_sse "${raw}")"
  if [[ -z "${result}" ]]; then
    echo "ERROR: MCP response contained no JSON payload:" >&2
    echo "${raw}" >&2
    exit 1
  fi
  if ! echo "${result}" | jq -e . >/dev/null 2>&1; then
    echo "ERROR: MCP response is not valid JSON:" >&2
    echo "${raw}" >&2
    exit 1
  fi
  if echo "${result}" | jq -e '.error' >/dev/null 2>&1; then
    echo "ERROR: MCP tool call returned an error" >&2
    exit 1
  fi
  if echo "${result}" | jq -e '.result.isError == true' >/dev/null 2>&1; then
    echo "ERROR: MCP tool call returned isError=true" >&2
    exit 1
  fi
  printf '%s' "${result}"
}

mcp_call() {
  local result
  result="$(mcp_call_raw "$1")"
  echo "${result}" | jq .
}

if [[ -z "${LOKI_QUERY}" ]]; then
  LOKI_QUERY="{SrcK8S_Namespace=\"${NETOBSERV_NS}\"} or {DstK8S_Namespace=\"${NETOBSERV_NS}\"}"
fi

echo "==> Checking OpenShift RBAC for NetObserv namespace ${NETOBSERV_NS}"
if ! oc auth can-i list pods -n "${NETOBSERV_NS}" >/dev/null 2>&1; then
  echo "WARN: $(oc whoami) cannot list pods in ${NETOBSERV_NS}."
  echo "      The Loki gateway may return empty streams even when logs exist."
  echo "      Apply optional RBAC and bind your user:"
  echo "        oc apply -f hack/loki_multitenancy_openshift/install/02_gateway_query_rbac.yaml"
  echo "        oc create clusterrolebinding obs-mcp-loki-gateway-read-\$(oc whoami | tr '@:' '-') \\"
  echo "          --clusterrole=obs-mcp-loki-gateway-read --user=\$(oc whoami)"
else
  echo "OK: can list pods in ${NETOBSERV_NS}"
fi

echo "==> MCP call: loki_list_instances"
mcp_call '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"loki_list_instances","arguments":{}}}'

echo "==> MCP call: loki_label_names (tenant ${LOKI_TENANT})"
LABELS_PAYLOAD="$(jq -nc \
  --arg ns "${STACK_NS}" \
  --arg name "${STACK_NAME}" \
  --arg tenant "${LOKI_TENANT}" \
  '{jsonrpc:"2.0",id:2,method:"tools/call",params:{name:"loki_label_names",arguments:{lokiNamespace:$ns,lokiName:$name,tenant:$tenant}}}')"
mcp_call "${LABELS_PAYLOAD}"

echo "==> MCP call: loki_query_range (tenant ${LOKI_TENANT})"
echo "    query: ${LOKI_QUERY}"
QUERY_PAYLOAD="$(jq -nc \
  --arg ns "${STACK_NS}" \
  --arg name "${STACK_NAME}" \
  --arg tenant "${LOKI_TENANT}" \
  --arg query "${LOKI_QUERY}" \
  --arg duration "${VERIFY_DURATION}" \
  '{jsonrpc:"2.0",id:3,method:"tools/call",params:{name:"loki_query_range",arguments:{lokiNamespace:$ns,lokiName:$name,tenant:$tenant,query:$query,duration:$duration,limit:50}}}')"
QUERY_RESULT=""
STREAM_COUNT=0
for attempt in $(seq 1 "${LOKI_QUERY_MAX_ATTEMPTS}"); do
  QUERY_RESULT="$(mcp_call_raw "${QUERY_PAYLOAD}")"
  STREAM_COUNT="$(echo "${QUERY_RESULT}" | jq -r '.result.structuredContent.streams | length // 0')"
  if [[ "${STREAM_COUNT}" != "0" ]]; then
    echo "OK: query returned ${STREAM_COUNT} stream(s) (attempt ${attempt}/${LOKI_QUERY_MAX_ATTEMPTS})"
    break
  fi
  if [[ "${attempt}" -lt "${LOKI_QUERY_MAX_ATTEMPTS}" ]]; then
    echo "    Attempt ${attempt}/${LOKI_QUERY_MAX_ATTEMPTS}: query returned 0 streams, retrying in ${LOKI_QUERY_SLEEP}s..."
    sleep "${LOKI_QUERY_SLEEP}"
  fi
done
echo "${QUERY_RESULT}" | jq .

if [[ "${STREAM_COUNT}" == "0" ]]; then
  echo "WARN: query returned 0 streams."
  echo "      - NetObserv flow logs use SrcK8S_Namespace/DstK8S_Namespace, not kubernetes_namespace_name."
  echo "      - Set NETOBSERV_NS to the namespace you filter in the UI (default: netobserv)."
  echo "      - Ensure FlowCollector exports to this LokiStack and your user can view ${NETOBSERV_NS}."
  echo "      - Try: LOKI_QUERY='{K8S_FlowLayer=\"app\"}' VERIFY_DURATION=1h $0"
fi

echo "==> Verification complete"
