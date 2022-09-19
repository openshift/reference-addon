module github.com/openshift/reference-addon

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo/v2 v2.1.6
	github.com/onsi/gomega v1.20.1
	github.com/operator-framework/api v0.10.7
	github.com/prometheus/client_golang v1.11.0
	github.com/stretchr/testify v1.7.0
	go.uber.org/multierr v1.6.0
	k8s.io/api v0.22.1
	k8s.io/apiextensions-apiserver v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	sigs.k8s.io/controller-runtime v0.10.0
	sigs.k8s.io/yaml v1.2.0
)
