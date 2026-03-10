#!/bin/bash
# Setup Kind cluster with kube-prometheus stack for E2E testing

set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-obs-mcp-e2e}"
KUBE_PROMETHEUS_VERSION="${KUBE_PROMETHEUS_VERSION:-release-0.16}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
KUBE_PROMETHEUS_DIR="${ROOT_DIR}/tmp/kube-prometheus"

echo "==> Creating Kind cluster: ${CLUSTER_NAME}"
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    echo "    Cluster '${CLUSTER_NAME}' already exists, skipping creation"
else
    kind create cluster --name "${CLUSTER_NAME}" --config "${ROOT_DIR}/tests/e2e/kind/kind-config.yaml" --wait 5m
fi

echo "==> Installing kube-prometheus stack (${KUBE_PROMETHEUS_VERSION})"
if [ ! -d "${KUBE_PROMETHEUS_DIR}" ]; then
    mkdir -p "${ROOT_DIR}/tmp"
    git clone --depth 1 --branch "${KUBE_PROMETHEUS_VERSION}" \
        https://github.com/prometheus-operator/kube-prometheus.git "${KUBE_PROMETHEUS_DIR}"
fi

# Apply CRDs and namespace setup first
kubectl apply --server-side -f "${KUBE_PROMETHEUS_DIR}/manifests/setup"
echo "==> Waiting for CRDs to be established..."
kubectl wait --for condition=Established --all CustomResourceDefinition --namespace=monitoring --timeout=5m

echo "==> Installing Prometheus Operator..."
for f in "${KUBE_PROMETHEUS_DIR}"/manifests/prometheusOperator-*.yaml; do
    kubectl apply -f "$f"
done

echo "==> Installing Prometheus..."
for f in "${KUBE_PROMETHEUS_DIR}"/manifests/prometheus-*.yaml; do
    kubectl apply -f "$f"
done

echo "==> Installing Alertmanager..."
for f in "${KUBE_PROMETHEUS_DIR}"/manifests/alertmanager-*.yaml; do
    kubectl apply -f "$f"
done

echo "==> Installing Cert Manager..."
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.19.4/cert-manager.yaml
kubectl -n cert-manager rollout status deployment/cert-manager --timeout=5m
kubectl -n cert-manager rollout status deployment/cert-manager-cainjector --timeout=5m
kubectl -n cert-manager rollout status deployment/cert-manager-webhook --timeout=5m

echo "==> Installing OpenTelemetry operator..."
kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/download/v0.146.0/opentelemetry-operator.yaml
kubectl -n opentelemetry-operator-system rollout status deployment/opentelemetry-operator-controller-manager --timeout=5m

echo "==> Installing Tempo operator..."
kubectl apply -f https://github.com/grafana/tempo-operator/releases/download/v0.20.0/tempo-operator.yaml
kubectl -n tempo-operator-system rollout status deployment/tempo-operator-controller --timeout=5m

echo "==> Waiting for Prometheus Operator to be ready..."
kubectl -n monitoring rollout status deployment/prometheus-operator --timeout=5m

echo "==> Waiting for Prometheus to be ready..."
kubectl -n monitoring rollout status statefulset/prometheus-k8s --timeout=5m

echo "==> Waiting for Alertmanager to be ready..."
kubectl -n monitoring rollout status statefulset/alertmanager-main --timeout=5m

echo "==> Setting up MinIO, OTEL collector, Tempo and example traces"
kubectl apply -f "${SCRIPT_DIR}/manifests/tracing"
kubectl -n obs-mcp-tracing wait --for=condition=Ready tempostack/tempo1 --timeout=5m
kubectl -n obs-mcp-tracing wait --for=condition=Ready tempostack/tempo2 --timeout=5m
kubectl -n obs-mcp-tracing rollout status statefulset/tempo-tempo1-ingester --timeout=5m
kubectl -n obs-mcp-tracing rollout status statefulset/tempo-tempo2-ingester --timeout=5m

echo "==> Cluster setup complete!"
echo "    Run 'make test-e2e-deploy' to build and deploy obs-mcp"
echo "    Run 'make test-e2e' to run E2E tests"
