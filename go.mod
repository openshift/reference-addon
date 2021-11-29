module github.com/openshift/reference-addon

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/openshift/addon-operator/apis v0.0.0-20211123091739-31282c5f436f
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.22.3
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/yaml v1.2.0
)
