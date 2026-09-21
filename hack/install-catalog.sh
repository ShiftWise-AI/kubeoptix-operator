#!/usr/bin/env bash
# Register the ShiftWise Operator Catalog in an OpenShift cluster.
#
# By default this points the CatalogSource straight at the pre-built,
# publicly readable images on quay.io (quay.io/parraes/...). No build/push
# and no RBAC grants are required for a normal install: OLM (running in
# openshift-marketplace) and the operator Deployment (running in
# openshift-operators) both pull directly from quay.io.
#
# Usage:
#   ./hack/install-catalog.sh
#   ./hack/install-catalog.sh --version 0.2.1
#   ./hack/install-catalog.sh --catalog-image quay.io/parraes/shiftwise-operator-catalog:v1.0.1
#
# Maintainers publishing a new version to quay.io can pass --build (requires
# `podman login quay.io` beforehand, with push access to the "parraes" org):
#   ./hack/install-catalog.sh --build --version 0.2.3
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

MARKETPLACE_NAMESPACE="openshift-marketplace"
CATALOG_NAME="shiftwise-operator-catalog"
QUAY_ORG="${QUAY_ORG:-parraes}"
VERSION="${VERSION:-1.0.1}"
CONTAINER_TOOL="${CONTAINER_TOOL:-podman}"
CONTAINERFILE="${CONTAINERFILE:-Containerfile}"
DEFAULT_TIMEOUT="180s"
TIMEOUT="${TIMEOUT:-${DEFAULT_TIMEOUT}}"

DO_BUILD=false
CATALOG_IMG_OVERRIDE=""

usage() {
  cat <<EOF
Register the ShiftWise Operator Catalog, pulling images from quay.io.

Usage:
  $(basename "$0") [options]

Options:
  --version VERSION      Operator/bundle/catalog version (default: ${VERSION})
  --catalog-image IMAGE  Full CatalogSource image reference (overrides --version)
  --build                Build and push new images to quay.io before applying
                         the CatalogSource (requires 'podman login quay.io')
  --timeout DURATION     Wait timeout for the catalog pod to become ready (default: ${DEFAULT_TIMEOUT})
  -h, --help             Show this help

Environment:
  QUAY_ORG, VERSION, CONTAINER_TOOL, TIMEOUT
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      VERSION="${2:?--version requires a value}"
      shift 2
      ;;
    --catalog-image)
      CATALOG_IMG_OVERRIDE="${2:?--catalog-image requires a value}"
      shift 2
      ;;
    --build)
      DO_BUILD=true
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

OPERATOR_IMG="quay.io/${QUAY_ORG}/shiftwise-operator:${VERSION}"
BUNDLE_IMG="quay.io/${QUAY_ORG}/shiftwise-operator-bundle:v${VERSION}"
CATALOG_IMG="${CATALOG_IMG_OVERRIDE:-quay.io/${QUAY_ORG}/shiftwise-operator-catalog:v${VERSION}}"

build_and_push() {
  command -v "${CONTAINER_TOOL}" >/dev/null 2>&1 || die "${CONTAINER_TOOL} is not installed"
  command -v python3 >/dev/null 2>&1 || die "python3 is required to generate the OLM bundle"

  log "generating OLM bundle manifests (VERSION=${VERSION})"
  VERSION="${VERSION}" OPERATOR_IMG="${OPERATOR_IMG}" BUNDLE_IMG="${BUNDLE_IMG}" \
    python3 "${ROOT_DIR}/hack/generate-olm.py"

  log "building operator image ${OPERATOR_IMG}"
  "${CONTAINER_TOOL}" build -f "${CONTAINERFILE}" -t "${OPERATOR_IMG}" --build-arg "VERSION=${VERSION}" .
  log "pushing ${OPERATOR_IMG}"
  "${CONTAINER_TOOL}" push "${OPERATOR_IMG}"

  log "building bundle image ${BUNDLE_IMG}"
  "${CONTAINER_TOOL}" build -f bundle.Dockerfile -t "${BUNDLE_IMG}" .
  log "pushing ${BUNDLE_IMG}"
  "${CONTAINER_TOOL}" push "${BUNDLE_IMG}"

  log "building catalog image ${CATALOG_IMG}"
  "${CONTAINER_TOOL}" build -f catalog.Dockerfile -t "${CATALOG_IMG}" .
  log "pushing ${CATALOG_IMG}"
  "${CONTAINER_TOOL}" push "${CATALOG_IMG}"
}

apply_catalogsource() {
  log "applying CatalogSource ${CATALOG_NAME} (image: ${CATALOG_IMG})"
  oc apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ${CATALOG_NAME}
  namespace: ${MARKETPLACE_NAMESPACE}
spec:
  sourceType: grpc
  image: ${CATALOG_IMG}
  displayName: ShiftWise Operator Catalog
  publisher: ShiftWise AI
  grpcPodConfig:
    securityContextConfig: restricted
  updateStrategy:
    registryPoll:
      interval: 10m
EOF
  log "restarting catalog pod to pick up the latest image"
  oc delete pod -n "${MARKETPLACE_NAMESPACE}" -l "olm.catalogSource=${CATALOG_NAME}" --ignore-not-found=true
}

wait_for_catalog() {
  log "waiting for catalog pod to become ready (timeout ${TIMEOUT})"
  oc wait --for=condition=Ready pod -n "${MARKETPLACE_NAMESPACE}" -l "olm.catalogSource=${CATALOG_NAME}" --timeout="${TIMEOUT}"

  log "waiting for CatalogSource connection state to be READY"
  local i state
  for i in $(seq 1 30); do
    state="$(oc get catalogsource "${CATALOG_NAME}" -n "${MARKETPLACE_NAMESPACE}" -o jsonpath='{.status.connectionState.lastObservedState}' 2>/dev/null || true)"
    [[ "${state}" == "READY" ]] && break
    sleep 2
  done
  [[ "${state}" == "READY" ]] || die "CatalogSource did not reach READY (last state: ${state:-unknown})"
}

wait_for_package() {
  log "waiting for ShiftWise package to appear in PackageManifest"
  local i
  for i in $(seq 1 30); do
    if oc get packagemanifest shiftwise-operator -n "${MARKETPLACE_NAMESPACE}" >/dev/null 2>&1; then
      return
    fi
    sleep 2
  done
  die "ShiftWise package did not appear in PackageManifest (CatalogSource is READY but OLM has not synced it)"
}

print_status() {
  log "CatalogSource status:"
  oc get catalogsource "${CATALOG_NAME}" -n "${MARKETPLACE_NAMESPACE}"
  log "PackageManifest (should list shiftwise-operator):"
  oc get packagemanifest -n "${MARKETPLACE_NAMESPACE}" 2>/dev/null | grep -i shiftwise || \
    echo "  (not yet synced; wait a few seconds and re-run: oc get packagemanifest -n ${MARKETPLACE_NAMESPACE} | grep shiftwise)"
}

if [[ "${DO_BUILD}" == true ]]; then
  build_and_push
fi

apply_catalogsource
wait_for_catalog
wait_for_package
print_status

log "done. Open Operators -> OperatorHub in the console and search for 'ShiftWise Operator'."
