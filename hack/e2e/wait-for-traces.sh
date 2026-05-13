#!/bin/bash
# Wait for traces to appear in a Tempo instance by polling the search API using a curl pod.
# Usage: wait-for-traces.sh <namespace> <tempo-query-frontend-url>

set -euo pipefail

NAMESPACE="${1:?Usage: wait-for-traces.sh <namespace> <tempo-query-frontend-url>}"
URL="${2:?Usage: wait-for-traces.sh <namespace> <tempo-query-frontend-url>}"
CURL_IMAGE="quay.io/curl/curl"
MAX_ATTEMPTS=20
SLEEP=30

if command -v kubectl &>/dev/null; then
    KUBECTL=kubectl
elif command -v oc &>/dev/null; then
    KUBECTL=oc
else
    echo "Error: neither kubectl nor oc found in PATH"
    exit 1
fi

echo "==> Waiting for traces to appear at ${URL}..."
for i in $(seq 1 "${MAX_ATTEMPTS}"); do
    output=$($KUBECTL run -n "${NAMESPACE}" curl-check --image="${CURL_IMAGE}" --rm -q --restart=Never -i -- \
        curl -vvsf "${URL}/api/search" 2>&1) || true
    if echo "$output" | grep -q '"traceID"'; then
        echo "✓ Traces found"
        exit 0
    fi

    echo "$output"
    echo "    Attempt ${i}/${MAX_ATTEMPTS}: no traces yet, retrying in ${SLEEP}s..."
    sleep "${SLEEP}"
done

echo "✗ No traces found after ${MAX_ATTEMPTS} attempts"
exit 1
