# Image URL to use all building/pushing image targets
IMG ?= quay.io/parraes/shiftwise-operator:0.2.0
VERSION ?= 0.2.0
NAMESPACE ?= shiftwise-ai

CONTAINER_TOOL ?= podman
CONTAINERFILE ?= Containerfile

ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

ifeq ($(shell command -v oc >/dev/null 2>&1 && echo oc),oc)
KUBECTL ?= oc
else
KUBECTL ?= kubectl
endif

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate CRDs and RBAC from markers.
	"$(CONTROLLER_GEN)" rbac:roleName=shiftwise-operator-manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases output:rbac:artifacts:config=config/rbac

.PHONY: generate
generate: controller-gen ## Generate DeepCopy methods.
	"$(CONTROLLER_GEN)" object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run unit tests.
	go test ./... -coverprofile cover.out

.PHONY: lint
lint: ## Run golangci-lint when available.
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed; skipping"

##@ Build

.PHONY: build
build: fmt vet ## Build manager binary.
	CGO_ENABLED=0 go build -ldflags "-X github.com/ShiftWise-AI/kubeoptix-operator/internal/version.Version=$(VERSION)" -o bin/manager cmd/main.go

.PHONY: run
run: fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

.PHONY: image-build
image-build: ## Build operator image with Podman using UBI.
	$(CONTAINER_TOOL) build -f $(CONTAINERFILE) -t ${IMG} --build-arg VERSION=$(VERSION) .

.PHONY: image-push
image-push: ## Push operator image.
	$(CONTAINER_TOOL) push ${IMG}

##@ Deployment

.PHONY: install
install: ## Install CRDs into the cluster.
	$(KUBECTL) apply -f config/crd/bases/shiftwise.ai_shiftwises.yaml

.PHONY: uninstall
uninstall: ## Remove CRDs from the cluster.
	$(KUBECTL) delete --ignore-not-found=true -f config/crd/bases/shiftwise.ai_shiftwises.yaml

.PHONY: deploy
deploy: ## Deploy the operator into shiftwise-ai.
	$(KUBECTL) apply -k config/default

.PHONY: undeploy
undeploy: ## Remove the operator from the cluster.
	$(KUBECTL) delete --ignore-not-found=true -k config/default

##@ Dependencies

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p "$(LOCALBIN)"

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
CONTROLLER_TOOLS_VERSION ?= v0.17.3

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN)
$(CONTROLLER_GEN): $(LOCALBIN)
	@test -s "$(CONTROLLER_GEN)" && "$(CONTROLLER_GEN)" --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
		GOBIN="$(LOCALBIN)" go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)
