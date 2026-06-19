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

# Wait for a resource to be created by an operator, then wait for its rollout to complete.
# Usage: _wait_rollout <namespace> <type/name> [timeout]
_wait_rollout() {
    local namespace="$1"
    local resource="$2"
    local timeout="${3:-5m}"
    _run $KUBECTL -n "${namespace}" wait --for=create "${resource}" --timeout="${timeout}"
    _run $KUBECTL -n "${namespace}" rollout status "${resource}" --timeout="${timeout}"
}

# Kill background port-forward processes.
# Usage: _cleanup_pf PID [PID ...]
_cleanup_pf() {
    step "Clean up port-forward processes"
    for pid in "$@"; do
        kill "$pid" 2>/dev/null || true
    done
}

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

# Can't use realpath --relative-base for macos compatibility.
_relativepath() {
    local _abs
    _abs="$(realpath "$1")"
    if [[ "${_abs}" == "${PWD}" ]]; then
        echo "."
    else
        echo "${_abs#"${PWD}"/}"
    fi
}
# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------

PHASE_DEFAULT="up"
PROFILE="kind"
STACKS="prometheus,tempo,loki"
SUPPORTED_PHASES=(provision prereqs extras upload deploy run clean unprovision)
SUPPORTED_PROFILES=(kind k8s openshift)
SUPPORTED_STACKS=(prometheus tempo loki)

KUBE_PROMETHEUS_VERSION="${KUBE_PROMETHEUS_VERSION:-release-0.16}"

# Preferring relative paths to have cleaner output.
ROOT_DIR="$(_relativepath "${SCRIPT_DIR}/../..")"

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

# Set the kubectl command based on profile
if [ "${PROFILE}" == "openshift" ]; then
    KUBECTL="oc"
else
    KUBECTL="kubectl"
fi

# ---------------------------------------------------------------------------
# Loki helper functions
# ---------------------------------------------------------------------------

# Detect the cluster's default StorageClass.
_detect_storage_class() {
    local _sc="${LOKI_STORAGE_CLASS:-}"
    if [[ -n "${_sc}" ]]; then
        echo "${_sc}"
        return
    fi
    _sc="$($KUBECTL get storageclass -o json | jq -r '.items[] | select(.metadata.annotations["storageclass.kubernetes.io/is-default-class"] == "true" or .metadata.annotations["storageclass.beta.kubernetes.io/is-default-class"] == "true") | .metadata.name' | head -n1)"
    if [[ -z "${_sc}" ]]; then
        echo "Unable to detect a default StorageClass. Set LOKI_STORAGE_CLASS explicitly and retry." >&2
        echo "Available StorageClasses:" >&2
        $KUBECTL get storageclass >&2
        return 1
    fi
    echo "${_sc}"
}

# ---------------------------------------------------------------------------
# Phase implementations
# ---------------------------------------------------------------------------

phase_provision() {
    phase "provision"
    case "${PROFILE}" in
        kind)
            step "Creating Kind cluster '${KIND_CLUSTER_NAME}'"
            if kind get clusters 2>/dev/null | grep -q "^${KIND_CLUSTER_NAME}$"; then
                info "Cluster '${KIND_CLUSTER_NAME}' already exists. Reusing."
                if [[ "$($KUBECTL config current-context)" != "kind-${KIND_CLUSTER_NAME}" ]]; then
                    _run $KUBECTL config use-context "kind-${KIND_CLUSTER_NAME}"
                fi
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
                _run $KUBECTL apply --server-side -f "${KUBE_PROMETHEUS_DIR}/manifests/setup"
                _run $KUBECTL wait --for condition=Established --all CustomResourceDefinition --namespace=monitoring --timeout=5m

                step "Installing Prometheus"
                _run $KUBECTL apply -f "${KUBE_PROMETHEUS_DIR}/manifests/prometheusOperator-*.yaml";
                _run $KUBECTL apply -f "${KUBE_PROMETHEUS_DIR}/manifests/prometheus-*.yaml";
                _run $KUBECTL apply -f "${KUBE_PROMETHEUS_DIR}/manifests/alertmanager-*.yaml";

                step "Waiting for deployments"
                _wait_rollout monitoring deployment/prometheus-operator 5m
                _wait_rollout monitoring statefulset/prometheus-k8s 5m
                _wait_rollout monitoring statefulset/alertmanager-main 5m
        esac
    fi

    if (has_stack tempo || has_stack loki) && [ "${PROFILE}" != "openshift" ]; then
        if ! $KUBECTL get crd certificates.cert-manager.io &>/dev/null; then
            step "Installing Cert Manager"
            _run $KUBECTL apply -f https://github.com/jetstack/cert-manager/releases/download/v1.19.4/cert-manager.yaml
            _wait_rollout cert-manager deployment/cert-manager 5m
            _wait_rollout cert-manager deployment/cert-manager-cainjector 5m
            _wait_rollout cert-manager deployment/cert-manager-webhook 5m
        fi
    fi

    if has_stack loki; then
        case ${PROFILE} in
            openshift)
                step "Installing Loki operator"
                _run $KUBECTL apply -k "${ROOT_DIR}/manifests/loki/prereqs/openshift/"
                _wait_rollout openshift-loki-operator deployment/loki-operator-controller-manager 10m

                step "Waiting for LokiStack CRD"
                _run $KUBECTL wait --for=condition=Established crd/lokistacks.loki.grafana.com --timeout=15m
            ;;
            kind|k8s)
                step "Installing Loki operator"
                _run $KUBECTL apply --server-side -k "${ROOT_DIR}/manifests/loki/prereqs/kubernetes/"
                _wait_rollout loki-operator deployment/loki-operator-controller-manager 5m

                step "Waiting for LokiStack CRD"
                _run $KUBECTL wait --for=condition=Established crd/lokistacks.loki.grafana.com --timeout=5m
            ;;
        esac
    fi

    if has_stack tempo; then
        case ${PROFILE} in
            openshift)
                _run $KUBECTL apply -f "${ROOT_DIR}/manifests/tempo/prereqs/openshift/"
                step "Installing OpenTelemetry operator"
                _wait_rollout openshift-opentelemetry-operator deployment/opentelemetry-operator-controller-manager 10m

                step "Installing Tempo operator"
                _wait_rollout openshift-tempo-operator deployment/tempo-operator-controller 10m
            ;;
            *)
                step "Installing OpenTelemetry operator"
                _run $KUBECTL apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/download/v0.146.0/opentelemetry-operator.yaml
                _wait_rollout opentelemetry-operator-system deployment/opentelemetry-operator-controller-manager 5m

                step "Installing Tempo operator"
                _run $KUBECTL apply -f https://github.com/grafana/tempo-operator/releases/download/v0.20.0/tempo-operator.yaml
                _wait_rollout tempo-operator-system deployment/tempo-operator-controller 5m
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
            _run $KUBECTL apply -f "${KUBE_PROMETHEUS_DIR}/manifests/kubeStateMetrics-*.yaml"
            _wait_rollout monitoring deployment/kube-state-metrics 3m

            step "Installing node-exporter"
            _run $KUBECTL apply -f "${KUBE_PROMETHEUS_DIR}/manifests/nodeExporter-*.yaml"
            _wait_rollout monitoring daemonset/node-exporter 3m

            step "Installing kubelet/cAdvisor ServiceMonitors"
            _run $KUBECTL apply -f "${KUBE_PROMETHEUS_DIR}/manifests/kubernetesControlPlane-*.yaml"
        fi
    fi

    if has_stack tempo; then
        step "Deploying sample tracing app"
        case ${PROFILE} in
            openshift)  _run $KUBECTL apply -k "${ROOT_DIR}/manifests/tempo/extras/openshift" ;;
            *)          _run $KUBECTL apply -k "${ROOT_DIR}/manifests/tempo/extras/kubernetes" ;;
        esac
        _wait_rollout tracing statefulset/tempo-tempo1-ingester 5m
        _wait_rollout tracing statefulset/tempo-tempo2-ingester 5m

        step "Waiting for traces to appear in Tempo"
        # Just in case the some residual pod stayed there from last attempt.
        $KUBECTL delete pods -n tracing curl-check 2>/dev/null || true
        for i in $(seq 1 20); do
            run_info "attempt ${i}"
            output=$($KUBECTL run -n tracing curl-check --image=quay.io/curl/curl --rm -q --restart=Never -i -- \
                curl -vvsf http://tempo-tempo1-query-frontend.tracing:3200/api/search 2>&1) || true
            if echo "$output" | grep -q '"traceID"'; then
                break
            fi
            warn "Attempt ${i}/20: no traces yet, retrying in 30s..."
            sleep 30
            [[ $i -eq 20 ]] && fail "No traces found after 20 attempts"
        done
    fi

    if has_stack loki; then
        case ${PROFILE} in
            openshift) _loki_extras_base="../openshift" ;;
            *)         _loki_extras_base="../kubernetes" ;;
        esac

        step "Deploying Loki test stack (obs-mcp-loki)"

        # Detect the cluster's default StorageClass
        _loki_sc=$(_detect_storage_class)
        info "Using StorageClass: ${_loki_sc}"

        # Build a temporary overlay that patches storageClassName into
        # the LokiStack CR.
        _overlay="${ROOT_DIR}/manifests/loki/extras/overlay"
        mkdir -p "${_overlay}"
        cat > "${_overlay}/kustomization.yaml" <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ${_loki_extras_base}
patches:
  - patch: |
      apiVersion: loki.grafana.com/v1
      kind: LokiStack
      metadata:
        name: obs-mcp-loki
        namespace: obs-mcp-loki
      spec:
        storageClassName: "${_loki_sc}"
EOF
        _run $KUBECTL apply -k "${_overlay}"

        # Wait for dependencies
        step "Waiting for MinIO"
        _wait_rollout obs-mcp-loki deployment/minio 5m

        step "Waiting for LokiStack Ready"
        _run $KUBECTL wait --for=jsonpath='{.status.conditions[?(@.type=="Ready")].status}'=True \
            lokistack/obs-mcp-loki -n obs-mcp-loki --timeout=10m

        step "Waiting for log generator"
        _wait_rollout obs-mcp-loki deployment/obs-mcp-log-generator 2m
    fi
}

phase_upload() {
    phase "upload"

    if [ -z "${IMAGE_REF:-}" ]; then
        info "IMAGE_REF not set, skipping image upload."
        return
    fi

    case "${PROFILE}" in
        kind)
            step "Loading obs-mcp image into Kind cluster '${KIND_CLUSTER_NAME}'"
            if [ "${CONTAINER_CLI}" == "podman" ]; then
                mkdir -p "${ROOT_DIR}/tmp"
                _run "${CONTAINER_CLI}" save --quiet -o "${ROOT_DIR}/tmp/obs-mcp.tar" "${IMAGE_REF}"
                _run kind load image-archive --name "${KIND_CLUSTER_NAME}" "${ROOT_DIR}/tmp/obs-mcp.tar"
                rm -f "${ROOT_DIR}/tmp/obs-mcp.tar"
            else
                _run kind load docker-image --name "${KIND_CLUSTER_NAME}" "${IMAGE_REF}"
            fi
            _run kubectl delete pod -l app=obs-mcp --ignore-not-found=true
            ;;
        openshift)
            step "Pushing obs-mcp image to OpenShift internal registry"

            if ! command -v skopeo >/dev/null; then
                fail "skopeo is needed for the upload functionality to work"
            fi

            # Derive the image tag from IMAGE_REF (use the tag portion, or fall back to 'latest')
            local _img_tag
            _img_tag=${IMAGE_REF##*:}
            if [ "${_img_tag}" == "${IMAGE_REF}" ]; then
               # no substitution = tag not found, use latest;
               _img_tag="latest"
            fi

            # Ensure the target namespace exists (otherwise pushing an image fails with "denied")
            if ! $KUBECTL get namespace obs-mcp &>/dev/null; then
                _run $KUBECTL create namespace obs-mcp
            fi

            # Enable the default external route for the image registry (idempotent)
            _run $KUBECTL patch configs.imageregistry.operator.openshift.io/cluster \
                --patch '{"spec":{"defaultRoute":true}}' --type=merge

            # Retrieve the external registry hostname
            local _ext_registry
            _ext_registry=$(oc get route default-route \
                -n openshift-image-registry \
                --template='{{ .spec.host }}')
            info "External registry route: ${_ext_registry}"

            # Save the image to a tar archive so that skopeo can use the
            # docker-archive: transport, which avoids the docker-daemon: transport's
            # hard-coded /var/tmp dependency (may not exist in all environments).
            local _tmp_tar
            _tmp_tar=$(mktemp -t obs-mcp-XXXXXX.tar)
            _run "${CONTAINER_CLI}" save "${IMAGE_REF}" -o "${_tmp_tar}"

            # Copy directly with skopeo — no separate login or tag step needed.
            # The token is fed via an anonymous pipe (<(...)) so it never touches
            # disk and is not logged by run_info (only /dev/fd/N appears in args).
            _run skopeo --insecure-policy copy \
                --dest-tls-verify=false \
                --dest-authfile <(printf '{"auths":{"%s":{"auth":"%s"}}}' \
                    "${_ext_registry}" \
                    "$(printf 'unused:%s' "$(oc whoami -t)" | base64 -w0)") \
                "docker-archive:${_tmp_tar}" \
                "docker://${_ext_registry}/obs-mcp/obs-mcp:${_img_tag}"
            rm -f "${_tmp_tar}"

            # Override IMAGE_REF to the in-cluster registry address used by the deployment
            IMAGE_REF="image-registry.openshift-image-registry.svc:5000/obs-mcp/obs-mcp:${_img_tag}"
            info "Updated IMAGE_REF for deployment: ${IMAGE_REF}"
            ;;
        *)
            info "Image upload not supported for profile '${PROFILE}', skipping."
            ;;
    esac
}

phase_deploy() {
    phase "deploy"

    step "Deploying obs-mcp"

    # Compute toolsets from enabled stacks
    _toolsets_parts=(otelcol)
    has_stack prometheus && _toolsets_parts+=(metrics)
    has_stack tempo      && _toolsets_parts+=(traces)
    has_stack loki       && _toolsets_parts+=(logs)
    _toolsets=$(IFS=,; echo "${_toolsets_parts[*]}")

    # Build a temporary kustomize overlay to inject runtime values (toolsets, image)
    # in a single apply so no post-apply mutations are needed.
    # The overlay is created inside the deploy tree so that the relative resource
    # path (../kubernetes or ../openshift) resolves correctly — kustomize forbids
    # absolute paths in resources.
    case ${PROFILE} in
        openshift) _profile=openshift ;;
        *)         _profile=kubernetes ;;
    esac
    _overlay="${ROOT_DIR}/manifests/core/deploy/overlay"
    mkdir -p "${_overlay}"
    cat > "${_overlay}/kustomization.yaml" <<EOF
resources:
  - ../${_profile}
patches:
  - patch: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: obs-mcp-config
        namespace: obs-mcp
      data:
        toolsets: "${_toolsets}"
EOF
    if [ -n "${IMAGE_REF:-}" ]; then
        cat >> "${_overlay}/kustomization.yaml" <<EOF
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: obs-mcp
        namespace: obs-mcp
      spec:
        template:
          spec:
            containers:
              - name: obs-mcp
                image: "${IMAGE_REF}"
EOF
    fi
    _run $KUBECTL apply -k "${_overlay}"

    # Monitoring integration
    if has_stack prometheus; then
        case ${PROFILE} in
            openshift)  _run $KUBECTL apply -f "${ROOT_DIR}/manifests/prometheus/deploy/openshift/" ;;
            *)          _run $KUBECTL apply -f "${ROOT_DIR}/manifests/prometheus/deploy/kubernetes/" ;;
        esac
    fi

    # Tempo tracing RBAC
    if has_stack tempo; then
        _run $KUBECTL apply -f "${ROOT_DIR}/manifests/tempo/deploy/"
    fi

    # Loki discovery RBAC
    if has_stack loki; then
        _run $KUBECTL apply -f "${ROOT_DIR}/manifests/loki/deploy/"
    fi

    step "Waiting for obs-mcp rollout"
    _wait_rollout obs-mcp deployment/obs-mcp 3m
}

phase_run() {
    phase "run"

    step "Building obs-mcp"
    _run go build -mod=mod -o "${ROOT_DIR}/obs-mcp" ./cmd/obs-mcp

    # Collect background port-forward PIDs for cleanup on exit.
    local _pf_pids=()
    trap '_cleanup_pf "${_pf_pids[@]}"' EXIT INT TERM

    local _env_vars=()
    local _flags=(
        --listen ":9100"
        --auth-mode kubeconfig
        --insecure
        --log-level debug
    )

    # Toolsets — always include otelcol.
    local _toolsets_parts=(otelcol)

    # -- Prometheus & Alertmanager --
    if has_stack prometheus; then
        _toolsets_parts+=(metrics)
        case ${PROFILE} in
            openshift)
                step "Port-forwarding Prometheus (openshift-monitoring)"
                _run $KUBECTL port-forward -n openshift-monitoring pod/prometheus-k8s-0 9090:9090 &
                _pf_pids+=($!)
                step "Port-forwarding Alertmanager (openshift-monitoring)"
                _run $KUBECTL port-forward -n openshift-monitoring pod/alertmanager-main-0 9093:9093 &
                _pf_pids+=($!)
                ;;
            *)
                step "Port-forwarding Prometheus (monitoring)"
                _run $KUBECTL port-forward -n monitoring svc/prometheus-k8s 9090:9090 &
                _pf_pids+=($!)
                step "Port-forwarding Alertmanager (monitoring)"
                _run $KUBECTL port-forward -n monitoring svc/alertmanager-main 9093:9093 &
                _pf_pids+=($!)
                ;;
        esac
        _env_vars+=(PROMETHEUS_URL=http://localhost:9090 ALERTMANAGER_URL=http://localhost:9093)
    fi

    # -- Tempo --
    if has_stack tempo; then
        _toolsets_parts+=(traces)
        case ${PROFILE} in
            openshift)
                _flags+=(--traces.use-route)
                ;;
            *)
                step "Port-forwarding Tempo query-frontend (tracing/tempo1)"
                _run $KUBECTL port-forward -n tracing svc/tempo-tempo1-query-frontend 3200:3200 &
                _pf_pids+=($!)
                _flags+=(--traces.tempo-url http://localhost:3200)
                ;;
        esac
    fi

    # -- Loki --
    if has_stack loki; then
        _toolsets_parts+=(logs)
        case ${PROFILE} in
            openshift)
                _flags+=(--loki.use-route)
                ;;
            *)
                step "Port-forwarding Loki gateway (obs-mcp-loki)"
                $KUBECTL port-forward -n obs-mcp-loki svc/obs-mcp-loki-gateway-http 8080:8080 &
                _pf_pids+=($!)
                _flags+=(--loki-url http://localhost:8080)
                ;;
        esac
    fi

    _flags+=(--toolsets "$(IFS=,; echo "${_toolsets_parts[*]}")") 

    # Give port-forwards a moment to bind.
    if [[ ${#_pf_pids[@]} -gt 0 ]]; then
        sleep 2
        # check the pids are still alive and fail if some failed
        for _pid in "${_pf_pids[@]}"; do
            if ! kill -0 "$_pid" 2>/dev/null; then
                fail "Port-forward process $_pid failed to start or exited unexpectedly"
            fi
        done
    fi

    step "Starting obs-mcp"
    info "Flags: ${_flags[*]}"
    info "Env:   ${_env_vars[*]}"

    env "${_env_vars[@]}" "${ROOT_DIR}/obs-mcp" "${_flags[@]}"
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
        run)         phase_run ;;
        clean)       phase_clean ;;
        unprovision) phase_unprovision ;;
        *)           fail "Unknown phase: ${phase}" ;;
    esac
done
