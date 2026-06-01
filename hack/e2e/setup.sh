#!/usr/bin/env bash
# Configure kubernetes cluster for E2E testing

set -euo pipefail

# ---------------------------------------------------------------------------
# Utils
# ---------------------------------------------------------------------------

SCRIPT_DIR="$(dirname "$(realpath "${BASH_SOURCE[0]}")")"
RED='\033[0;31m' GREEN='\033[0;32m' CYAN='\033[0;36m' YELLOW='\033[0;33m' BOLD='\033[1m' NC='\033[0m'
phase() { echo -e "\n${BOLD}${YELLOW}==== $1 ====${NC}"; }
step()  { echo -e "\n${CYAN}==> $1${NC}"; }
ok()  { echo -e "    ${GREEN}✓${NC} $1"; }
run_info()  { echo -e "    ${CYAN}»${NC} $*"; }
info()  { echo -e "    ${CYAN}i${NC} $1"; }
warn()  { echo -e "    ${YELLOW}!${NC} $1"; }
fail()  { echo -e "    ${RED}✗${NC} $1"; exit 1; }

# Execute the command while capturing the output. Print the output to stderr on fail.
_run() {
    local _out
    _out=$(mktemp)
    run_info "$@"
    if "$@" >"${_out}" 2>&1; then
        rm -f "${_out}"
    else
        local _rc=$?
        echo -e "    ${RED}✗${NC} Command failed: $*" >&2
        cat "${_out}" >&2
        rm -f "${_out}"
        return ${_rc}
    fi
}

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------

PHASE_DEFAULT="up"
PROFILE="kind"
STACKS="prometheus,tempo"
SUPPORTED_PHASES=(provision prereqs extras upload deploy clean unprovision)
SUPPORTED_PROFILES=(kind k8s openshift)
SUPPORTED_STACKS=(prometheus tempo)

KUBE_PROMETHEUS_VERSION="${KUBE_PROMETHEUS_VERSION:-release-0.16}"
# Using relative-to to have cleaner output (skipping unnecessary absolute paths)
ROOT_DIR="$(realpath "${SCRIPT_DIR}/../.."  --relative-base .)"
KUBE_PROMETHEUS_DIR="${ROOT_DIR}/tmp/kube-prometheus"
CONTAINER_CLI="${CONTAINER_CLI:-docker}"

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-obs-mcp-e2e}"

usage() {
    cat <<EOF
Usage: $(basename "$0") [PHASE_EXP] [--profile PROFILE] [--stacks STACKS]

PHASE_EXP   Phase alias or space-separated list of concrete phases.
            Aliases  : up | down
            Phases   : ${SUPPORTED_PHASES[*]}
            Default  : ${PHASE_DEFAULT}
--profile   Cluster profile to target.
            Supported: ${SUPPORTED_PROFILES[*]// /, }
            Default  : ${PROFILE}
--stacks    Comma-separated list of observability stacks to install.
            Supported: ${SUPPORTED_STACKS[*]// /, }
            Default  : ${STACKS}
EOF
}

PHASE_EXP=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --profile)
            PROFILE="$2"; shift 2 ;;
        --stacks)
            STACKS="$2"; shift 2 ;;
        --help|-h)
            usage; exit 0 ;;
        --*)
            fail "Unknown argument: $1" ;;
        *)
            PHASE_EXP+=("$1"); shift ;;
    esac
done

# Expand aliases / defaults
if [[ ${#PHASE_EXP[@]} -eq 0 ]]; then
    PHASE_EXP=("${PHASE_DEFAULT}")
fi

PHASES=()
for arg in "${PHASE_EXP[@]}"; do
    case "${arg}" in
        up)  
            if [ "${PROFILE}" == "kind" ]; then
                PHASES+=(provision)
            fi
            PHASES+=(prereqs extras upload deploy)
            ;;
        down)
            PHASES+=(clean)
            if [ "${PROFILE}" == "kind" ]; then
                PHASES+=(unprovision)
            fi
            ;;
        *)
            PHASES+=("${arg}") ;;
    esac
done

for phase in "${PHASES[@]}"; do
    if [[ " ${SUPPORTED_PHASES[*]} " != *" ${phase} "* ]]; then
        fail "Unknown phase: '${phase}'. Supported aliases: up, down. Supported phases: ${SUPPORTED_PHASES[*]// /, }"
    fi
done

has_stack()  { [[ ",${STACKS}," == *",${1},"* ]]; }
has_phase()  { local p; for p in "${PHASES[@]}"; do [[ "$p" == "$1" ]] && return 0; done; return 1; }

info "Profile : ${PROFILE}"
info "Stacks  : ${STACKS}"
info "Phases  : ${PHASES[*]}"

# ---------------------------------------------------------------------------
# Phase implementations
# ---------------------------------------------------------------------------

phase_provision() {
    phase "provision"
    case "${PROFILE}" in
        kind)
            step "Creating Kind cluster '${KIND_CLUSTER_NAME}'"
            if kind get clusters 2>/dev/null | grep -q "^${KIND_CLUSTER_NAME}$"; then
                info "Cluster '${KIND_CLUSTER_NAME}' already exists, skipping creation"
            else
                _run kind create cluster --name "${KIND_CLUSTER_NAME}" \
                    --config "${SCRIPT_DIR}/kind-config.yaml" --wait 5m
            fi
            ;;
        *)
            fail "provision phase is only supported for the 'kind' profile" ;;
    esac
}

phase_prereqs() {
    phase "prereqs"

    if has_stack prometheus; then
        case ${PROFILE} in
            openshift)
                info "${KUBE_PROMETHEUS_DIR} already present, skipping."
            ;;
            *)
                step "Installing kube-prometheus stack (${KUBE_PROMETHEUS_VERSION})"
                if [ ! -d "${KUBE_PROMETHEUS_DIR}" ]; then
                    mkdir -p "${ROOT_DIR}/tmp"
                    _run git clone --depth 1 --branch "${KUBE_PROMETHEUS_VERSION}" \
                        https://github.com/prometheus-operator/kube-prometheus.git "${KUBE_PROMETHEUS_DIR}"
                else
                    info "${KUBE_PROMETHEUS_DIR} already present, skipping."
                fi

                step "Applying kube-prometheus CRDs and namespace setup"
                _run kubectl apply --server-side -f "${KUBE_PROMETHEUS_DIR}/manifests/setup"
                _run kubectl wait --for condition=Established --all CustomResourceDefinition --namespace=monitoring --timeout=5m

                step "Installing Prometheus"
                _run kubectl apply -f "${KUBE_PROMETHEUS_DIR}/manifests/prometheusOperator-*.yaml";
                _run kubectl apply -f "${KUBE_PROMETHEUS_DIR}/manifests/prometheus-*.yaml";
                _run kubectl apply -f "${KUBE_PROMETHEUS_DIR}/manifests/alertmanager-*.yaml";

                step "Waiting for deployments"
                _run kubectl -n monitoring rollout status deployment/prometheus-operator --timeout=5m
                _run kubectl -n monitoring rollout status statefulset/prometheus-k8s --timeout=5m
                _run kubectl -n monitoring rollout status statefulset/alertmanager-main --timeout=5m
        esac
    fi

    if has_stack tempo; then
        case ${PROFILE} in
            openshift)
                _run oc apply -f "${ROOT_DIR}/manifests/openshift_e2e/prereqs/01_tracing_operators.yaml"
                step "Installing OpenTelemetry operator"
                _run oc -n openshift-opentelemetry-operator wait --for=create deployment/opentelemetry-operator-controller-manager --timeout=10m
                _run oc -n openshift-opentelemetry-operator rollout status deployment/opentelemetry-operator-controller-manager --timeout=5m

                step "Installing Tempo operator"
                _run oc -n openshift-tempo-operator wait --for=create deployment/tempo-operator-controller --timeout=10m
                _run oc -n openshift-tempo-operator rollout status deployment/tempo-operator-controller --timeout=5m
            ;;
            *)
                step "Installing Cert Manager"
                _run kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.19.4/cert-manager.yaml
                _run kubectl -n cert-manager rollout status deployment/cert-manager --timeout=5m
                _run kubectl -n cert-manager rollout status deployment/cert-manager-cainjector --timeout=5m
                _run kubectl -n cert-manager rollout status deployment/cert-manager-webhook --timeout=5m

                step "Installing OpenTelemetry operator"
                _run kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/download/v0.146.0/opentelemetry-operator.yaml
                _run kubectl -n opentelemetry-operator-system rollout status deployment/opentelemetry-operator-controller-manager --timeout=5m

                step "Installing Tempo operator"
                _run kubectl apply -f https://github.com/grafana/tempo-operator/releases/download/v0.20.0/tempo-operator.yaml
                _run kubectl -n tempo-operator-system rollout status deployment/tempo-operator-controller --timeout=5m
        esac
    fi
}

phase_extras() {
    phase "extras"

    if has_stack prometheus; then
        step "Installing kubernetes related metrics sources"
        if [ "${PROFILE}" == "openshift" ]; then
            info "Skipping for openshift profile. Sources already installed."
        else
            if [ ! -d "${KUBE_PROMETHEUS_DIR}" ]; then
                fail "${KUBE_PROMETHEUS_DIR} not found. Run prereqs phase first."
            fi

            step "Installing kube-state-metrics"
            _run kubectl apply -f "${KUBE_PROMETHEUS_DIR}/manifests/kubeStateMetrics-*.yaml"
            _run kubectl -n monitoring rollout status deployment/kube-state-metrics --timeout=3m

            step "Installing node-exporter"
            _run kubectl apply -f "${KUBE_PROMETHEUS_DIR}/manifests/nodeExporter-*.yaml"
            _run kubectl -n monitoring rollout status daemonset/node-exporter --timeout=3m

            step "Installing kubelet/cAdvisor ServiceMonitors"
            _run kubectl apply -f "${KUBE_PROMETHEUS_DIR}/manifests/kubernetesControlPlane-*.yaml"
        fi
    fi

    if has_stack tempo; then
        step "Deploying sample traing app"
        # TODO: check differences betwen openshift and kuberentes tempo prereqs
        case ${PROFILE} in
            openshift)
                _run kubectl apply -f "${ROOT_DIR}/manifests/openshift_e2e/prereqs/02_tracing.yaml"
            ;;
            *)
                _run kubectl apply -f "${ROOT_DIR}/manifests/kubernetes/prereqs/01_tracing.yaml"
        esac
        _run kubectl -n tracing rollout status statefulset/tempo-tempo1-ingester --timeout=5m
        _run kubectl -n tracing rollout status statefulset/tempo-tempo2-ingester --timeout=5m

        step "Waiting for traces to appear in Tempo"
        for i in $(seq 1 20); do
            run_info "attempt ${i}"
            output=$(kubectl run -n tracing curl-check --image=quay.io/curl/curl --rm -q --restart=Never -i -- \
                curl -vvsf http://tempo-tempo1-query-frontend.tracing:3200/api/search 2>&1) || true
            if echo "$output" | grep -q '"traceID"'; then
                break
            fi
            warn "Attempt ${i}/20: no traces yet, retrying in 30s..."
            sleep 30
            [[ $i -eq 20 ]] && fail "No traces found after 20 attempts"
        done
    fi
}

phase_upload() {
    phase "upload"

    if [ "${PROFILE}" == "kind" ]; then
        step "Loading obs-mcp image into Kind cluster '${KIND_CLUSTER_NAME}'"
        if [ -n "${IMAGE_REF:-}" ]; then
            if [ "${CONTAINER_CLI}" == "podman" ]; then
                mkdir -p "${ROOT_DIR}/tmp"
                _run "${CONTAINER_CLI}" save --quiet -o "${ROOT_DIR}/tmp/obs-mcp.tar" "${IMAGE_REF}"
                _run kind load image-archive --name "${KIND_CLUSTER_NAME}" "${ROOT_DIR}/tmp/obs-mcp.tar"
                rm -f "${ROOT_DIR}/tmp/obs-mcp.tar"
            else
                _run kind load docker-image --name "${KIND_CLUSTER_NAME}" "${IMAGE_REF}"
            fi
        else
            info "IMAGE_REF not set, skipping image loading."
        fi
    else
        info "Image upload not supported for profile '${PROFILE}', skipping."
    fi
}

phase_deploy() {
    phase "deploy"

    step "Deploying obs-mcp"

    case ${PROFILE} in
        openshift)
            _run oc apply -f "${ROOT_DIR}/manifests/openshift_e2e/*.yaml"
            ;;
        *)
            _run kubectl apply -f "${ROOT_DIR}/manifests/kubernetes/*.yaml"
            _run kubectl apply -f "${ROOT_DIR}/manifests/kubernetes/prereqs/02_prometheus_network_policy.yaml"
    esac
    if [ -n "${IMAGE_REF:-}" ]; then
        _run kubectl set image deployment/obs-mcp -n obs-mcp obs-mcp="${IMAGE_REF}"
    fi

    step "Waiting for obs-mcp rollout"
    _run kubectl -n obs-mcp rollout status deployment/obs-mcp --timeout=3m
}

phase_clean() {
    phase "clean"
    # TODO: remove temporary files (e.g. tmp/kube-prometheus)
    warn "clean phase not yet implemented"
}

phase_unprovision() {
    phase "unprovision"
    case "${PROFILE}" in
        kind)
            step "Deleting Kind cluster '${KIND_CLUSTER_NAME}'"
            _run kind delete cluster --name "${KIND_CLUSTER_NAME}"
            if [ -d "${KUBE_PROMETHEUS_DIR}" ]; then
                step "Removing kube-prometheus checkout"
                _run rm -rf "${KUBE_PROMETHEUS_DIR}"
            fi
            ;;
        *)
            fail "unprovision phase is only supported for the 'kind' profile" ;;
    esac
}

# ---------------------------------------------------------------------------
# Phase dispatch
# ---------------------------------------------------------------------------

for phase in "${PHASES[@]}"; do
    case "${phase}" in
        provision)   phase_provision ;;
        prereqs)     phase_prereqs ;;
        extras)      phase_extras ;;
        upload)      phase_upload ;;
        deploy)      phase_deploy ;;
        clean)       phase_clean ;;
        unprovision) phase_unprovision ;;
        *)           fail "Unknown phase: ${phase}" ;;
    esac
done

step "Cluster setup complete!"
info "Run 'make test-e2e-deploy' to build and deploy obs-mcp"
info "Run 'make test-e2e' to run E2E tests"
