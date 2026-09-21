#!/usr/bin/env bash
# Publish the ShiftWise operator catalog to an OpenShift cluster's internal
# registry and register the CatalogSource so the operator shows up in
# OperatorHub.
#
# This builds/pushes three images into the internal registry, under the
# "shiftwise-ai" namespace:
#   - shiftwise-operator:<version>          (operator image)
#   - shiftwise-operator-bundle:v<version>  (OLM bundle)
#   - shiftwise-operator-catalog:v<version> (file-based catalog, served by opm)
#
# It also grants the "openshift-marketplace" service account permission to
# pull images from "shiftwise-ai", since the CatalogSource pod runs in
# openshift-marketplace but the catalog/bundle images live in shiftwise-ai.
# Without this grant the catalog pod fails with ImagePullBackOff /
# "authentication required" and the operator never appears in OperatorHub.
#
# Usage:
#   ./hack/install-catalog.sh
#   ./hack/install-catalog.sh --version 1.0.1
#   ./hack/install-catalog.sh --skip-build   # only apply CatalogSource + RBAC
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

NAMESPACE="shiftwise-ai"
MARKETPLACE_NAMESPACE="openshift-marketplace"
CATALOG_NAME="shiftwise-operator-catalog"
VERSION="${VERSION:-1.0.0}"
CONTAINER_TOOL="${CONTAINER_TOOL:-podman}"
CONTAINERFILE="${CONTAINERFILE:-Containerfile}"
DEFAULT_TIMEOUT="180s"
TIMEOUT="${TIMEOUT:-${DEFAULT_TIMEOUT}}"

SKIP_BUILD=false

usage() {
  cat <<EOF
Publish the ShiftWise operator catalog and register it in OperatorHub.

Usage:
  $(basename "$0") [options]

Options:
  --version VERSION   Operator/bundle/catalog version (default: ${VERSION})
  --skip-build        Skip image build/push; only (re)apply CatalogSource and RBAC
  --timeout DURATION  Wait timeout for the catalog pod to become ready (default: ${DEFAULT_TIMEOUT})
  -h, --help          Show this help

Environment:
  VERSION, CONTAINER_TOOL, TIMEOUT
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      VERSION="${2:?--version requires a value}"
      shift 2
      ;;
    --skip-build)
      SKIP_BUILD=true
      shift
      ;;
    --timeout)
      TIMEOUT="${2:?--timeout requires a value}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown option: $1 (see --help)" >&2
      exit 1
      ;;
  esac
done

log()  { printf '==> %s\n' "$*"; }
die()  { printf 'error: %s\n' "$*" >&2; exit 1; }

command -v oc >/dev/null 2>&1 || die "oc is not installed or not in PATH"
oc whoami >/dev/null 2>&1 || die "not logged in; run: oc login --server=<api> --token=<token>"
oc api-resources --api-group=route.openshift.io >/dev/null 2>&1 || die "current cluster is not OpenShift"

IMG="default-route-openshift-image-registry.apps-crc.testing/${NAMESPACE}/shiftwise-operator:${VERSION}"
BUNDLE_IMG="default-route-openshift-image-registry.apps-crc.testing/${NAMESPACE}/shiftwise-operator-bundle:v${VERSION}"
CATALOG_IMG="default-route-openshift-image-registry.apps-crc.testing/${NAMESPACE}/shiftwise-operator-catalog:v${VERSION}"

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

ensure_registry_route() {
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
    [[ -n "${host}" ]] && { printf '%s' "${host}"; return; }
    sleep 2
  done
  die "timed out waiting for openshift-image-registry default-route"
}

ensure_imagestreams() {
  local stream
  for stream in shiftwise-operator shiftwise-operator-bundle shiftwise-operator-catalog; do
    if ! oc get imagestream "${stream}" -n "${NAMESPACE}" >/dev/null 2>&1; then
      log "creating ImageStream ${stream}"
      oc create imagestream "${stream}" -n "${NAMESPACE}" >/dev/null
    fi
  done
}

ensure_rbac() {
  # Allow this user/SA to push into the shiftwise-ai imagestreams.
  local user
  user="$(oc whoami)"
  oc policy add-role-to-user system:image-builder "${user}" -n "${NAMESPACE}" >/dev/null

  # Allow the CatalogSource pod (running in openshift-marketplace) to pull
  # the catalog/bundle images from shiftwise-ai. Without this, the catalog
  # pod fails with ImagePullBackOff even though the images exist.
  log "granting system:image-puller on ${NAMESPACE} to ${MARKETPLACE_NAMESPACE} service accounts"
  oc policy add-role-to-user system:image-puller \
    "system:serviceaccount:${MARKETPLACE_NAMESPACE}:${CATALOG_NAME}" \
    -n "${NAMESPACE}" >/dev/null
  oc policy add-role-to-user system:image-puller \
    "system:serviceaccount:${MARKETPLACE_NAMESPACE}:default" \
    -n "${NAMESPACE}" >/dev/null
}

registry_login() {
  local host="$1"
  local token
  token="$(oc whoami -t)"
  [[ -n "${token}" ]] || die "could not obtain oc token for registry login"
  log "logging into internal registry ${host}"
  "${CONTAINER_TOOL}" login -u "$(oc whoami)" -p "${token}" --tls-verify=false "${host}" >/dev/null
}

build_and_push() {
  command -v "${CONTAINER_TOOL}" >/dev/null 2>&1 || die "${CONTAINER_TOOL} is not installed"
  command -v python3 >/dev/null 2>&1 || die "python3 is required to generate the OLM bundle"

  log "generating OLM bundle manifests (VERSION=${VERSION})"
  VERSION="${VERSION}" python3 "${ROOT_DIR}/hack/generate-olm.py"

  log "building operator image ${IMG}"
  "${CONTAINER_TOOL}" build -f "${CONTAINERFILE}" -t "${IMG}" --build-arg "VERSION=${VERSION}" .
  log "pushing ${IMG}"
  "${CONTAINER_TOOL}" push --tls-verify=false "${IMG}"

  log "building bundle image ${BUNDLE_IMG}"
  "${CONTAINER_TOOL}" build -f bundle.Dockerfile -t "${BUNDLE_IMG}" .
  log "pushing ${BUNDLE_IMG}"
  "${CONTAINER_TOOL}" push --tls-verify=false "${BUNDLE_IMG}"

  log "building catalog image ${CATALOG_IMG}"
  "${CONTAINER_TOOL}" build -f catalog.Dockerfile -t "${CATALOG_IMG}" .
  log "pushing ${CATALOG_IMG}"
  "${CONTAINER_TOOL}" push --tls-verify=false "${CATALOG_IMG}"
}

apply_catalogsource() {
  log "applying CatalogSource ${CATALOG_NAME}"
  oc apply -f "${ROOT_DIR}/config/olm/catalogsource.yaml"
  log "restarting catalog pod to pick up the latest image"
  oc delete pod -n "${MARKETPLACE_NAMESPACE}" -l "olm.catalogSource=${CATALOG_NAME}" --ignore-not-found=true
}

wait_for_catalog() {
  log "waiting for catalog pod to become ready (timeout ${TIMEOUT})"
  oc wait --for=condition=Ready pod -n "${MARKETPLACE_NAMESPACE}" -l "olm.catalogSource=${CATALOG_NAME}" --timeout="${TIMEOUT}"

  log "waiting for CatalogSource connection state to be READY"
  local i
  for i in $(seq 1 30); do
    state="$(oc get catalogsource "${CATALOG_NAME}" -n "${MARKETPLACE_NAMESPACE}" -o jsonpath='{.status.connectionState.lastObservedState}' 2>/dev/null || true)"
    [[ "${state}" == "READY" ]] && break
    sleep 2
  done
  [[ "${state}" == "READY" ]] || die "CatalogSource did not reach READY (last state: ${state:-unknown})"
}

print_status() {
  log "CatalogSource status:"
  oc get catalogsource "${CATALOG_NAME}" -n "${MARKETPLACE_NAMESPACE}"
  log "PackageManifest (should list shiftwise-operator):"
  oc get packagemanifest -n "${MARKETPLACE_NAMESPACE}" 2>/dev/null | grep -i shiftwise || \
    echo "  (not yet synced; wait a few seconds and re-run: oc get packagemanifest -n ${MARKETPLACE_NAMESPACE} | grep shiftwise)"
}

ensure_namespace
ensure_imagestreams
ensure_rbac

if [[ "${SKIP_BUILD}" == false ]]; then
  registry_login "$(ensure_registry_route)"
  build_and_push
fi

apply_catalogsource
wait_for_catalog
print_status

log "done. Open Operators -> OperatorHub in the console and search for 'ShiftWise Operator'."
