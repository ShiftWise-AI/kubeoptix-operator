#!/usr/bin/env bash
# Deploy the ShiftWise operator to an OpenShift (OCP) cluster.
#
# Usage:
#   ./hack/deploy-ocp.sh
#   ./hack/deploy-ocp.sh --image quay.io/parraes/shiftwise-operator:0.2.1
#   ./hack/deploy-ocp.sh --build --push --internal-registry
#   ./hack/deploy-ocp.sh --with-credentials --with-instance
#   ./hack/deploy-ocp.sh --undeploy
#   ./hack/deploy-ocp.sh --undeploy --purge
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

DEFAULT_IMG="quay.io/parraes/shiftwise-operator:0.2.1"
DEFAULT_NAMESPACE="shiftwise-ai"
DEFAULT_TIMEOUT="180s"
DEPLOYMENT="shiftwise-operator-controller-manager"
CONTAINER_NAME="manager"
CONTAINERFILE="${CONTAINERFILE:-Containerfile}"

IMG="${IMG:-${DEFAULT_IMG}}"
NAMESPACE="${NAMESPACE:-${DEFAULT_NAMESPACE}}"
VERSION="${VERSION:-0.2.1}"
CONTAINER_TOOL="${CONTAINER_TOOL:-podman}"
TIMEOUT="${TIMEOUT:-${DEFAULT_TIMEOUT}}"

DO_BUILD=false
DO_PUSH=false
INTERNAL_REGISTRY=false
WITH_CREDENTIALS=false
WITH_INSTANCE=false
UNDEPLOY=false
PURGE=false
SKIP_WAIT=false

usage() {
  cat <<EOF
Deploy the ShiftWise operator to an OpenShift cluster.

Usage:
  $(basename "$0") [options]

Options:
  --image IMG              Operator image (default: ${DEFAULT_IMG} or \$IMG)
  --namespace NS           Operator namespace (must be ${DEFAULT_NAMESPACE})
  --timeout DURATION       Rollout wait timeout (default: ${DEFAULT_TIMEOUT})
  --build                  Build the operator image with ${CONTAINER_TOOL}
  --push                   Push the image after build (implies --build)
  --internal-registry      Push to the cluster internal registry and deploy that image
  --with-credentials       Apply sample Secrets (kubeoptix-db, llm, github-auth)
  --with-instance          Apply the sample ShiftWise CR
  --skip-wait              Do not wait for the operator Deployment to become ready
  --undeploy               Remove the operator (keeps CRD and namespace)
  --purge                  With --undeploy, also delete the CRD and namespace
  -h, --help               Show this help

Environment:
  IMG, NAMESPACE, VERSION, CONTAINER_TOOL, TIMEOUT

Examples:
  $(basename "$0")
  $(basename "$0") --build --push
  $(basename "$0") --build --push --internal-registry
  $(basename "$0") --with-credentials --with-instance
  $(basename "$0") --undeploy --purge
EOF
}

log()  { printf '==> %s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }
die()  { printf 'error: %s\n' "$*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --image)
      IMG="${2:?--image requires a value}"
      shift 2
      ;;
    --namespace)
      NAMESPACE="${2:?--namespace requires a value}"
      shift 2
      ;;
    --timeout)
      TIMEOUT="${2:?--timeout requires a value}"
      shift 2
      ;;
    --build)
      DO_BUILD=true
      shift
      ;;
    --push)
      DO_PUSH=true
      DO_BUILD=true
      shift
      ;;
    --internal-registry)
      INTERNAL_REGISTRY=true
      shift
      ;;
    --with-credentials)
      WITH_CREDENTIALS=true
      shift
      ;;
    --with-instance)
      WITH_INSTANCE=true
      shift
      ;;
    --skip-wait)
      SKIP_WAIT=true
      shift
      ;;
    --undeploy)
      UNDEPLOY=true
      shift
      ;;
    --purge)
      PURGE=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown option: $1 (see --help)"
      ;;
  esac
done

command -v oc >/dev/null 2>&1 || die "oc is not installed or not in PATH"

oc whoami >/dev/null 2>&1 || die "not logged in; run: oc login --server=<api> --token=<token>"

if ! oc api-resources --api-group=route.openshift.io >/dev/null 2>&1; then
  die "current cluster is not OpenShift (route.openshift.io is missing)"
fi

if [[ "${NAMESPACE}" != "${DEFAULT_NAMESPACE}" ]]; then
  die "manifests are pinned to namespace ${DEFAULT_NAMESPACE} (got ${NAMESPACE})"
fi

SERVER="$(oc whoami --show-server 2>/dev/null || true)"
USER="$(oc whoami 2>/dev/null || true)"
log "cluster: ${SERVER:-unknown}  user: ${USER:-unknown}"

require_file() {
  local path="$1"
  [[ -f "${path}" ]] || die "missing ${path}"
}

wait_for_crd() {
  log "waiting for CRD shiftwises.shiftwise.ai"
  oc wait --for=condition=Established crd/shiftwises.shiftwise.ai --timeout="${TIMEOUT}"
}

print_status() {
  log "operator status in namespace ${NAMESPACE}"
  oc get deployment "${DEPLOYMENT}" -n "${NAMESPACE}" || true
  oc get pods -n "${NAMESPACE}" -l app.kubernetes.io/name=shiftwise-operator || true
  oc get crd shiftwises.shiftwise.ai >/dev/null 2>&1 && oc get shiftwises -n "${NAMESPACE}" || true
}

undeploy_operator() {
  log "removing operator from ${NAMESPACE}"
  if [[ -d "${ROOT_DIR}/config/default" ]]; then
    oc delete --ignore-not-found=true -k "${ROOT_DIR}/config/default"
  else
    oc delete --ignore-not-found=true deployment "${DEPLOYMENT}" -n "${NAMESPACE}"
  fi

  if [[ "${PURGE}" == true ]]; then
    log "purging CRD and namespace ${NAMESPACE}"
    oc delete --ignore-not-found=true -f "${ROOT_DIR}/config/crd/bases/shiftwise.ai_shiftwises.yaml"
    oc delete --ignore-not-found=true namespace "${NAMESPACE}"
  else
    log "CRD and namespace kept (pass --purge to delete them)"
  fi
  log "undeploy complete"
}

ensure_namespace() {
  if oc get namespace "${NAMESPACE}" >/dev/null 2>&1; then
    return
  fi
  log "creating namespace ${NAMESPACE}"
  oc create namespace "${NAMESPACE}"
  oc label namespace "${NAMESPACE}" \
    app.kubernetes.io/name=shiftwise-operator \
    app.kubernetes.io/part-of=kubeoptix \
    --overwrite
}

enable_internal_registry_route() {
  local host
  host="$(oc get route default-route -n openshift-image-registry -o jsonpath='{.spec.host}' 2>/dev/null || true)"
  if [[ -n "${host}" ]]; then
    printf '%s' "${host}"
    return
  fi

  log "enabling default route on the OpenShift image registry"
  oc patch configs.imageregistry.operator.openshift.io/cluster --type merge \
    -p '{"spec":{"defaultRoute":true}}' >/dev/null

  local i
  for i in $(seq 1 30); do
    host="$(oc get route default-route -n openshift-image-registry -o jsonpath='{.spec.host}' 2>/dev/null || true)"
    if [[ -n "${host}" ]]; then
      printf '%s' "${host}"
      return
    fi
    sleep 2
  done
  die "timed out waiting for openshift-image-registry default-route"
}

prepare_internal_image() {
  local stream="shiftwise-operator"
  local tag="${VERSION}"
  local registry_host
  local external_img
  local in_cluster_img

  ensure_namespace

  if ! oc get imagestream "${stream}" -n "${NAMESPACE}" >/dev/null 2>&1; then
    log "creating ImageStream ${stream}"
    oc create imagestream "${stream}" -n "${NAMESPACE}" >/dev/null
  fi

  registry_host="$(enable_internal_registry_route)"
  external_img="${registry_host}/${NAMESPACE}/${stream}:${tag}"
  in_cluster_img="image-registry.openshift-image-registry.svc:5000/${NAMESPACE}/${stream}:${tag}"

  if [[ "${DO_PUSH}" == true ]]; then
    local token
    token="$(oc whoami -t)"
    [[ -n "${token}" ]] || die "could not obtain oc token for registry login"

    log "logging into internal registry ${registry_host}"
    "${CONTAINER_TOOL}" login -u "${USER}" -p "${token}" --tls-verify=false "${registry_host}" >/dev/null

    log "tagging ${IMG} -> ${external_img}"
    "${CONTAINER_TOOL}" tag "${IMG}" "${external_img}"

    log "pushing ${external_img}"
    "${CONTAINER_TOOL}" push --tls-verify=false "${external_img}"
  fi

  IMG="${in_cluster_img}"
  log "in-cluster image: ${IMG}"
}

build_image() {
  command -v "${CONTAINER_TOOL}" >/dev/null 2>&1 || die "${CONTAINER_TOOL} is not installed"
  require_file "${ROOT_DIR}/${CONTAINERFILE}"
  log "building ${IMG} (VERSION=${VERSION})"
  "${CONTAINER_TOOL}" build -f "${CONTAINERFILE}" -t "${IMG}" --build-arg "VERSION=${VERSION}" .
}

push_image() {
  log "pushing ${IMG}"
  "${CONTAINER_TOOL}" push "${IMG}"
}

apply_operator() {
  require_file "${ROOT_DIR}/config/default/kustomization.yaml"

  log "applying operator manifests (kustomize config/default)"
  oc apply -k "${ROOT_DIR}/config/default"

  wait_for_crd

  log "setting manager image to ${IMG}"
  oc set image "deployment/${DEPLOYMENT}" \
    "${CONTAINER_NAME}=${IMG}" \
    -n "${NAMESPACE}" >/dev/null

  if [[ "${INTERNAL_REGISTRY}" == true ]]; then
    oc set image-lookup "${DEPLOYMENT}" -n "${NAMESPACE}" >/dev/null 2>&1 || true
    oc patch "deployment/${DEPLOYMENT}" -n "${NAMESPACE}" --type=json \
      -p '[{"op":"replace","path":"/spec/template/spec/containers/0/imagePullPolicy","value":"Always"}]' >/dev/null
  fi
}

wait_for_operator() {
  if [[ "${SKIP_WAIT}" == true ]]; then
    return
  fi
  log "waiting for ${DEPLOYMENT} rollout (timeout ${TIMEOUT})"
  oc rollout status "deployment/${DEPLOYMENT}" -n "${NAMESPACE}" --timeout="${TIMEOUT}"
  oc wait --for=condition=Available "deployment/${DEPLOYMENT}" -n "${NAMESPACE}" --timeout="${TIMEOUT}"
}

apply_credentials() {
  local file="${ROOT_DIR}/config/samples/kubeoptix-credentials-example.yaml"
  require_file "${file}"
  warn "sample secrets contain placeholders; replace them before starting builds"
  log "applying credentials from ${file}"
  oc apply -f "${file}"
}

apply_instance() {
  local file="${ROOT_DIR}/config/samples/shiftwise.ai_v1alpha1_shiftwise.yaml"
  require_file "${file}"
  log "applying ShiftWise instance from ${file}"
  oc apply -f "${file}"
  oc get shiftwises -n "${NAMESPACE}"
}

if [[ "${UNDEPLOY}" == true ]]; then
  undeploy_operator
  exit 0
fi

if [[ "${PURGE}" == true ]]; then
  die "--purge requires --undeploy"
fi

if [[ "${DO_BUILD}" == true ]]; then
  build_image
fi

if [[ "${INTERNAL_REGISTRY}" == true ]]; then
  if [[ "${DO_BUILD}" == true ]]; then
    DO_PUSH=true
  fi
  prepare_internal_image
elif [[ "${DO_PUSH}" == true ]]; then
  push_image
fi

apply_operator
wait_for_operator

if [[ "${WITH_CREDENTIALS}" == true ]]; then
  apply_credentials
fi

if [[ "${WITH_INSTANCE}" == true ]]; then
  apply_instance
fi

print_status
log "deploy complete"
log "next: oc logs -n ${NAMESPACE} deploy/${DEPLOYMENT} -f"
