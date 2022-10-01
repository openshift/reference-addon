SHELL=/bin/bash
.SHELLFLAGS=-euo pipefail -c

# Dependency Versions
YQ_VERSION:=v4@v4.7.0
OPM_VERSION:=v1.17.2

# Build Flags
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
SHORT_SHA=$(shell git rev-parse --short HEAD)
VERSION?=$(shell echo ${BRANCH} | tr / -)-${SHORT_SHA}

UNAME_OS:=$(shell uname -s)
UNAME_ARCH:=$(shell uname -m)

# PATH/Bin
DEPENDENCIES:=.cache/dependencies/$(UNAME_OS)/$(UNAME_ARCH)
export GOBIN?=$(abspath .cache/dependencies/bin)
export PATH:=$(GOBIN):$(PATH)
export GOLANGCI_LINT_CACHE=$(abspath .cache/golangci-lint)

# Container
IMAGE_ORG?=quay.io/app-sre
REFERENCE_ADDON_MANAGER_IMAGE?=$(IMAGE_ORG)/reference-addon-manager:$(VERSION)

# -------
# Compile
# -------

all: FORCE
	./mage build:manager

FORCE:

clean:
	@rm -rf bin .cache
.PHONY: clean

# ------------
# Dependencies
# ------------

# setup yq
YQ:=$(DEPENDENCIES)/yq/$(YQ_VERSION)
$(YQ):
	@echo "installing yq $(YQ_VERSION)..."
	$(eval YQ_TMP := $(shell mktemp -d))
	@(cd "$(YQ_TMP)" \
		&& go mod init tmp \
		&& go install "github.com/mikefarah/yq/$(YQ_VERSION)" \
	) 2>&1 | sed 's/^/  /'
	@rm -rf "$(YQ_TMP)" "$(dir $(YQ))" \
		&& mkdir -p "$(dir $(YQ))" \
		&& touch "$(YQ)" \
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
	$(YQ) \
	$(OPM)
.PHONY: dependencies

# ----------
# Development
# ----------

# Run against the configured Kubernetes cluster in ~/.kube/config or $KUBECONFIG
run: generate
	go run -ldflags "-w $(LD_FLAGS)" \
		./cmd/reference-addon-manager \
			-pprof-addr="127.0.0.1:8065"
.PHONY: run

# ----------
# Generators
# ----------

# Generate code and manifests e.g. CRD, RBAC etc.
generate:
	./mage generate

# -------------------
# Testing and Linting
# -------------------

# Runs code-generators, checks for clean directory and lints the source code.
lint:
	./mage lint
.PHONY: lint

test:
	./mage test
.PHONY: test

# Runs unittests
test-unit:
	./mage test:unit
.PHONY: test-unit

# Runs integration tests
test-integration:
	./mage test:integration
.PHONY: test-integration

# Template deployment
config/deploy/deployment.yaml: FORCE $(YQ)
	@yq eval '.spec.template.spec.containers[0].image = "$(REFERENCE_ADDON_MANAGER_IMAGE)"' \
		config/deploy/deployment.tpl.yaml > config/deploy/deployment.yaml

# Template Addon
# TODO(ykukreja): shift to making use of image digests here instead of tag. Currently blocked!
config/addon/reference-addon.yaml: FORCE $(YQ)
	$(eval IMAGE_NAME := reference-addon-index)
	@yq eval '.spec.install.olmOwnNamespace.catalogSourceImage = "${IMAGE_ORG}/${IMAGE_NAME}:${VERSION}"' \
		config/addon/reference-addon.tpl.yaml > config/addon/reference-addon.yaml

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
