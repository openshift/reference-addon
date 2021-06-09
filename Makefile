SHELL=/bin/bash
.SHELLFLAGS=-euo pipefail -c

# Dependency Versions
CONTROLLER_GEN_VERSION:=v0.5.0
KIND_VERSION:=v0.10.0
YQ_VERSION:=v4@v4.7.0
GOIMPORTS_VERSION:=v0.1.0
GOLANGCI_LINT_VERSION:=v1.39.0
OLM_VERSION:=v0.17.0
OPM_VERSION:=v1.17.2

# Build Flags
export CGO_ENABLED:=0
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
SHORT_SHA=$(shell git rev-parse --short HEAD)
VERSION?=$(shell echo ${BRANCH} | tr / -)-${SHORT_SHA}
BUILD_DATE=$(shell date +%s)
MODULE:=github.com/openshift/reference-addon
LD_FLAGS=-X $(MODULE)/internal/version.Version=$(VERSION) \
			-X $(MODULE)/internal/version.Branch=$(BRANCH) \
			-X $(MODULE)/internal/version.Commit=$(SHORT_SHA) \
			-X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE)

UNAME_OS:=$(shell uname -s)
UNAME_ARCH:=$(shell uname -m)

# PATH/Bin
DEPENDENCIES:=.cache/dependencies/$(UNAME_OS)/$(UNAME_ARCH)
export GOBIN?=$(abspath .cache/dependencies/bin)
export PATH:=$(GOBIN):$(PATH)

# Config
KIND_KUBECONFIG:=.cache/e2e/kubeconfig
export KUBECONFIG?=$(abspath $(KIND_KUBECONFIG))
export GOLANGCI_LINT_CACHE=$(abspath .cache/golangci-lint)

# Container
IMAGE_ORG?=quay.io/app-sre
REFERENCE_ADDON_MANAGER_IMAGE?=$(IMAGE_ORG)/reference-addon-manager:$(VERSION)

# -------
# Compile
# -------

all: \
	bin/linux_amd64/reference-addon-manager

bin/linux_amd64/%: GOARGS = GOOS=linux GOARCH=amd64

bin/%: generate FORCE
	$(eval COMPONENT=$(shell basename $*))
	@echo -e -n "compiling cmd/$(COMPONENT)...\n  "
	$(GOARGS) go build -ldflags "-w $(LD_FLAGS)" -o bin/$* cmd/$(COMPONENT)/main.go
	@echo

FORCE:

# prints the version as used by build commands.
version:
	@echo $(VERSION)
.PHONY: version

clean:
	@rm -rf bin .cache
.PHONY: clean

# ------------
# Dependencies
# ------------

# setup kind
KIND:=$(DEPENDENCIES)/kind/$(KIND_VERSION)
$(KIND):
	@echo "installing kind $(KIND_VERSION)..."
	$(eval KIND_TMP := $(shell mktemp -d))
	@(cd "$(KIND_TMP)" \
		&& go mod init tmp \
		&& go get "sigs.k8s.io/kind@$(KIND_VERSION)" \
	) 2>&1 | sed 's/^/  /'
	@rm -rf "$(KIND_TMP)" "$(dir $(KIND))" \
		&& mkdir -p "$(dir $(KIND))" \
		&& touch "$(KIND)" \
		&& echo

# setup controller-gen
CONTROLLER_GEN:=$(DEPENDENCIES)/controller-gen/$(CONTROLLER_GEN_VERSION)
$(CONTROLLER_GEN):
	@echo "installing controller-gen $(CONTROLLER_GEN_VERSION)..."
	$(eval CONTROLLER_GEN_TMP := $(shell mktemp -d))
	@(cd "$(CONTROLLER_GEN_TMP)" \
		&& go mod init tmp \
		&& go get "sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)" \
	) 2>&1 | sed 's/^/  /'
	@rm -rf "$(CONTROLLER_GEN_TMP)" "$(dir $(CONTROLLER_GEN))" \
		&& mkdir -p "$(dir $(CONTROLLER_GEN))" \
		&& touch "$(CONTROLLER_GEN)" \
		&& echo

# setup yq
YQ:=$(DEPENDENCIES)/yq/$(YQ_VERSION)
$(YQ):
	@echo "installing yq $(YQ_VERSION)..."
	$(eval YQ_TMP := $(shell mktemp -d))
	@(cd "$(YQ_TMP)" \
		&& go mod init tmp \
		&& go get "github.com/mikefarah/yq/$(YQ_VERSION)" \
	) 2>&1 | sed 's/^/  /'
	@rm -rf "$(YQ_TMP)" "$(dir $(YQ))" \
		&& mkdir -p "$(dir $(YQ))" \
		&& touch "$(YQ)" \
		&& echo

# setup goimports
GOIMPORTS:=$(DEPENDENCIES)/goimports/$(GOIMPORTS_VERSION)
$(GOIMPORTS):
	@echo "installing goimports $(GOIMPORTS_VERSION)..."
	$(eval GOIMPORTS_TMP := $(shell mktemp -d))
	@(cd "$(GOIMPORTS_TMP)" \
		&& go mod init tmp \
		&& go get "golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)" \
	) 2>&1 | sed 's/^/  /'
	@rm -rf "$(GOIMPORTS_TMP)" "$(dir $(GOIMPORTS))" \
		&& mkdir -p "$(dir $(GOIMPORTS))" \
		&& touch "$(GOIMPORTS)" \
		&& echo

goimports: $(GOIMPORTS)

# setup golangci-lint
GOLANGCI_LINT:=$(DEPENDENCIES)/golangci-lint/$(GOLANGCI_LINT_VERSION)
$(GOLANGCI_LINT):
	@echo "installing golangci-lint $(GOLANGCI_LINT_VERSION)..."
	$(eval GOLANGCI_LINT_TMP := $(shell mktemp -d))
	@(cd "$(GOLANGCI_LINT_TMP)" \
		&& go mod init tmp \
		&& go get "github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)" \
	) 2>&1 | sed 's/^/  /'
	@rm -rf "$(GOLANGCI_LINT_TMP)" "$(dir $(GOLANGCI_LINT))" \
		&& mkdir -p "$(dir $(GOLANGCI_LINT))" \
		&& touch "$(GOLANGCI_LINT)" \
		&& echo

OPM:=$(DEPENDENCIES)/opm/$(OPM_VERSION)
$(OPM):
	@echo "installing opm $(OPM_VERSION)..."
	$(eval OPM_TMP := $(shell mktemp -d))
	@(cd "$(OPM_TMP)"; \
		curl -L --fail \
		https://github.com/operator-framework/operator-registry/releases/download/$(OPM_VERSION)/linux-amd64-opm -o opm; \
		chmod +x opm; \
		mv opm $(GOBIN); \
	) 2>&1 | sed 's/^/  /'
	@rm -rf "$(OPM_TMP)" "$(dir $(OPM))" \
		&& mkdir -p "$(dir $(OPM))" \
		&& touch "$(OPM)" \
		&& echo

# installs all project dependencies
dependencies: \
	$(KIND) \
	$(CONTROLLER_GEN) \
	$(YQ) \
	$(GOIMPORTS) \
	$(GOLANGCI_LINT)
.PHONY: dependencies

# ----------
# Development
# ----------

# Run against the configured Kubernetes cluster in ~/.kube/config or $KUBECONFIG
run: generate
	go run -ldflags "-w $(LD_FLAGS)" \
		./cmd/reference-addon-manager/main.go \
			-pprof-addr="127.0.0.1:8065"
.PHONY: run

# ----------
# Generators
# ----------

# Generate code and manifests e.g. CRD, RBAC etc.
generate: $(CONTROLLER_GEN)
	@echo "generating kubernetes manifests..."
	@controller-gen crd:crdVersions=v1 \
		rbac:roleName=reference-addon \
		paths="./..." \
		output:crd:artifacts:config=config/deploy 2>&1 | sed 's/^/  /'
	@echo
	@echo "generating code..."
	@controller-gen object paths=./apis/... 2>&1 | sed 's/^/  /'
	@echo

# -------------------
# Testing and Linting
# -------------------

# Runs code-generators, checks for clean directory and lints the source code.
lint: generate $(GOLANGCI_LINT)
	go fmt ./...
	golangci-lint run ./... --deadline=15m
	@hack/validate-directory-clean.sh
.PHONY: lint

# Runs unittests
test-unit: generate
	CGO_ENABLED=1 go test -race -v ./internal/... ./cmd/...
.PHONY: test-unit

# Runs the E2E testsuite against the currently selected cluster.
# FORCE_FLAGS ensures that the tests will not be cached
FORCE_FLAGS = -count=1
test-e2e: config/deploy/deployment.yaml
	@echo "running e2e tests..."

	@go test -v $(FORCE_FLAGS) ./e2e/...
.PHONY: test-e2e

# Sets up a local kind cluster and runs E2E tests against this local cluster.
test-e2e-local: export KUBECONFIG=$(abspath $(KIND_KUBECONFIG))
test-e2e-local: | setup-e2e-kind test-e2e
.PHONY: test-e2e-local

# Run the E2E testsuite after installing the Reference Addon into the cluster.
test-e2e-ci: | apply-reference-addon test-e2e

# make sure that we install our components into the kind cluster and disregard normal $KUBECONFIG
setup-e2e-kind: export KUBECONFIG=$(abspath $(KIND_KUBECONFIG))
setup-e2e-kind: | \
	create-kind-cluster \
	load-reference-addon

create-kind-cluster: $(KIND)
	@echo "creating kind cluster reference-addon-e2e..."
	@mkdir -p .cache/e2e
	@(source hack/determine-container-runtime.sh; \
		$$KIND_COMMAND create cluster \
			--kubeconfig="$(KIND_KUBECONFIG)" \
			--name="reference-addon-e2e"; \
		if [[ ! -O "$(KIND_KUBECONFIG)" ]]; then \
			sudo chown $$USER: "$(KIND_KUBECONFIG)"; \
		fi; \
		echo) 2>&1 | sed 's/^/  /'
.PHONY: create-kind-cluster

delete-kind-cluster: $(KIND)
	@echo "deleting kind cluster reference-addon-e2e..."
	@(source hack/determine-container-runtime.sh; \
		$$KIND_COMMAND delete cluster \
			--kubeconfig="$(KIND_KUBECONFIG)" \
			--name "reference-addon-e2e"; \
		rm -rf "$(KIND_KUBECONFIG)"; \
		echo) 2>&1 | sed 's/^/  /'
.PHONY: delete-kind-cluster

# Load Addon Operator Image into kind
load-reference-addon: build-image-reference-addon-manager
	@source hack/determine-container-runtime.sh; \
		$$KIND_COMMAND load image-archive \
			.cache/image/reference-addon-manager.tar \
			--name reference-addon-e2e
.PHONY: load-reference-addon

# Template deployment
config/deploy/deployment.yaml: FORCE $(YQ)
	@yq eval '.spec.template.spec.containers[0].image = "$(REFERENCE_ADDON_MANAGER_IMAGE)"' \
		config/deploy/deployment.tpl.yaml > config/deploy/deployment.yaml

# Installs OLM (Operator Lifecycle Manager) into the currently selected cluster.
apply-olm:
	@echo "installing OLM $(OLM_VERSION)..."
	@(kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/$(OLM_VERSION)/crds.yaml \
		&& kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/$(OLM_VERSION)/olm.yaml \
		&& echo -e "\nwaiting for deployment/olm-operator..." \
		&& kubectl wait --for=condition=available deployment/olm-operator -n olm --timeout=240s \
		&& echo -e "\nwaiting for deployment/catalog-operator..." \
		&& kubectl wait --for=condition=available deployment/catalog-operator -n olm --timeout=240s \
		&& echo) 2>&1 | sed 's/^/  /'
.PHONY: apply-olm

# Installs the OpenShift/OKD console into the currently selected cluster.
apply-openshift-console:
	@echo "installing OpenShift console :latest..."
	@(kubectl apply -f hack/openshift-console.yaml \
		&& echo) 2>&1 | sed 's/^/  /'
.PHONY: apply-openshift-console

# Installs the Addon Operator into the kind e2e cluster.
apply-reference-addon: $(YQ) load-reference-addon config/deploy/deployment.yaml
	@echo "installing Addon Operator $(VERSION)..."
	@(source hack/determine-container-runtime.sh; \
		kubectl apply -f config/deploy; \
		echo -e "\nwaiting for deployment/reference-addon..."; \
		kubectl wait --for=condition=available deployment/reference-addon -n reference-addon --timeout=240s; \
		echo) 2>&1 | sed 's/^/  /'
.PHONY: apply-reference-addon

# ------
# OLM
# ------

# Template Cluster Service Version / CSV
# By setting the container image to deploy.
config/olm/reference-addon.csv.yaml: FORCE $(YQ)
	@yq eval '.spec.install.spec.deployments[0].spec.template.spec.containers[0].image = "$(REFERENCE_ADDON_MANAGER_IMAGE)" | .metadata.annotations.containerImage = "$(REFERENCE_ADDON_MANAGER_IMAGE)"' \
	config/olm/reference-addon.csv.tpl.yaml > config/olm/reference-addon.csv.yaml

# Bundle image contains the manifests and CSV for a single version of this operator.
build-image-reference-addon-bundle: \
	clean-image-cache-reference-addon-bundle \
	config/olm/reference-addon.csv.yaml
	$(eval IMAGE_NAME := reference-addon-bundle)
	@echo "building image ${IMAGE_ORG}/${IMAGE_NAME}:${VERSION}..."
	@(source hack/determine-container-runtime.sh; \
		mkdir -p ".cache/image/${IMAGE_NAME}/manifests"; \
		mkdir -p ".cache/image/${IMAGE_NAME}/metadata"; \
		cp -a "config/olm/reference-addon.csv.yaml" ".cache/image/${IMAGE_NAME}/manifests"; \
		cp -a "config/olm/annotations.yaml" ".cache/image/${IMAGE_NAME}/metadata"; \
		cp -a "config/docker/${IMAGE_NAME}.Dockerfile" ".cache/image/${IMAGE_NAME}/Dockerfile"; \
		$$CONTAINER_COMMAND build -t "${IMAGE_ORG}/${IMAGE_NAME}:${VERSION}" ".cache/image/${IMAGE_NAME}"; \
		$$CONTAINER_COMMAND image save -o ".cache/image/${IMAGE_NAME}.tar" "${IMAGE_ORG}/${IMAGE_NAME}:${VERSION}"; \
		echo) 2>&1 | sed 's/^/  /'
.PHONY: build-image-reference-addon-bundle

# Index image contains a list of bundle images for use in a CatalogSource.
# Warning!
# The bundle image needs to be pushed so the opm CLI can create the index image.
build-image-reference-addon-index: $(OPM) \
	clean-image-cache-reference-addon-index \
	| build-image-reference-addon-bundle \
	push-image-reference-addon-bundle
	$(eval IMAGE_NAME := reference-addon-index)
	@echo "building image ${IMAGE_ORG}/${IMAGE_NAME}:${VERSION}..."
	@(source hack/determine-container-runtime.sh; \
		echo "building ${IMAGE_ORG}/${IMAGE_NAME}:${VERSION}"; \
		opm index add --container-tool $$CONTAINER_COMMAND \
		--bundles ${IMAGE_ORG}/reference-addon-bundle:${VERSION} \
		--tag ${IMAGE_ORG}/${IMAGE_NAME}:${VERSION}; \
		$$CONTAINER_COMMAND image save -o ".cache/image/${IMAGE_NAME}.tar" "${IMAGE_ORG}/${IMAGE_NAME}:${VERSION}"; \
		echo) 2>&1 | sed 's/^/  /'
.PHONY: build-image-reference-addon-index

# ----------------
# Container Images
# ----------------

build-images: \
	build-image-reference-addon-manager
.PHONY: build-images

push-images: \
	push-image-reference-addon-manager
.PHONY: push-images

.SECONDEXPANSION:
# cleans the built image .tar and image build directory
clean-image-cache-%:
	@rm -rf ".cache/image/$*" ".cache/image/$*.tar"
	@mkdir -p ".cache/image/$*"

build-image-%: clean-image-cache-%
	@echo "building image ${IMAGE_ORG}/$*:${VERSION}..."
	@(source hack/determine-container-runtime.sh; \
		cp -a "bin/linux_amd64/$*" ".cache/image/$*"; \
		cp -a "config/docker/passwd" ".cache/image/$*/passwd"; \
		cp -a "config/docker/$*.Dockerfile" ".cache/image/$*/Dockerfile"; \
		echo "building ${IMAGE_ORG}/$*:${VERSION}"; \
		$$CONTAINER_COMMAND build -t "${IMAGE_ORG}/$*:${VERSION}" ".cache/image/$*"; \
		$$CONTAINER_COMMAND image save -o ".cache/image/$*.tar" "${IMAGE_ORG}/$*:${VERSION}"; \
		echo) 2>&1 | sed 's/^/  /'

push-image-%: build-image-$$*
	@echo "pushing image ${IMAGE_ORG}/$*:${VERSION}..."
	@(source hack/determine-container-runtime.sh; \
		$$CONTAINER_COMMAND push "${IMAGE_ORG}/$*:${VERSION}"; \
		echo pushed "${IMAGE_ORG}/$*:${VERSION}"; \
		echo) 2>&1 | sed 's/^/  /'
