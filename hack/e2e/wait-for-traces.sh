#!/usr/bin/env bash
# Wait for traces to appear in Tempo by polling the search API from inside the cluster.
# Usage: wait-for-traces.sh <namespace> <tempo-query-frontend-base-url>
# Example: wait-for-traces.sh tracing http://tempo-tempo1-query-frontend.tracing:3200
#
# Uses TraceQL q={} — a bare /api/search often returns no rows even when traces exist.

set -euo pipefail

NAMESPACE="${1:?Usage: wait-for-traces.sh <namespace> <tempo-query-frontend-base-url>}"
URL="${2:?Usage: wait-for-traces.sh <namespace> <tempo-query-frontend-base-url>}"
CURL_IMAGE="${CURL_IMAGE:-quay.io/curl/curl:latest}"
MAX_ATTEMPTS="${MAX_ATTEMPTS:-40}"
SLEEP="${SLEEP:-15}"

CLI="$(command -v oc || true)"
if [[ -z "${CLI}" ]]; then
	CLI="$(command -v kubectl)"
fi
if [[ -z "${CLI}" ]]; then
	echo "error: neither oc nor kubectl found in PATH" >&2
	exit 1
fi

echo "==> Waiting for traces at ${URL} (TraceQL q={}, limit=5)..."

for i in $(seq 1 "${MAX_ATTEMPTS}"); do
	POD="curl-traces-$(date +%s)-${i}"
	# -f/--fail makes curl exit non-zero on HTTP errors so we retry.
	if "${CLI}" run -n "${NAMESPACE}" "${POD}" --image="${CURL_IMAGE}" --rm --restart=Never -i -- \
		curl -fsS -G "${URL}/api/search" \
			--data-urlencode 'q={}' \
			--data-urlencode 'limit=5' 2>/dev/null | grep -qE '"traceID"|"traceId"'; then
		echo "✓ Traces found"
		exit 0
	fi
	echo "    Attempt ${i}/${MAX_ATTEMPTS}: no traces yet, retrying in ${SLEEP}s..."
	sleep "${SLEEP}"
done

echo "✗ No traces found after ${MAX_ATTEMPTS} attempts"
exit 1
