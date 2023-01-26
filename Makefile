LD_FLAGS=-X $(MODULE)/internal/version.Version=$(VERSION) \
			-X $(MODULE)/internal/version.Branch=$(BRANCH) \
			-X $(MODULE)/internal/version.Commit=$(SHORT_SHA) \
			-X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE)

## Generate deepcopy code, kubernetes manifests and docs.
generate: 
	./mage generate
.PHONY: generate			

run: generate
	go run -ldflags "-w $(LD_FLAGS)" \
		./cmd/reference-addon-manager/*.go

setup-reference-addon-crds: generate
	@for crd in $(wildcard config/deploy/*.openshift.io_*.yaml); do \
		kubectl apply -f $$crd; \
	done
.PHONY: setup-reference-addon-crds