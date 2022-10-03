SHELL=/bin/bash
.SHELLFLAGS=-euo pipefail -c

all:
	./mage build:manager

clean:
	@rm -rf bin .cache
.PHONY: clean

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
