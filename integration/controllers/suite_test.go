package controllers

import (
	"context"
	"testing"

	refapis "github.com/openshift/reference-addon/apis"
	olmcrds "github.com/operator-framework/api/crds"
	opsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	_cfg     *rest.Config
	_client  client.Client
	_testEnv *envtest.Environment
	_ctx     context.Context
	_cancel  context.CancelFunc
	_scheme  = runtime.NewScheme()
)

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "controllers suite")
}

var _ = BeforeSuite(func() {
	_ctx, _cancel = context.WithCancel(context.Background())

	By("Registering schemes")

	Expect(clientgoscheme.AddToScheme(_scheme)).Should(Succeed())
	Expect(opsv1alpha1.AddToScheme(_scheme)).Should(Succeed())
	Expect(refapis.AddToScheme(_scheme)).Should(Succeed())

	By("Starting test environment")

	_testEnv = &envtest.Environment{
		Scheme: _scheme,
	}

	var err error

	_cfg, err = _testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(_cfg).ToNot(BeNil())

	DeferCleanup(cleanup)

	By("Installing CRD's")

	_, err = envtest.InstallCRDs(_cfg, envtest.CRDInstallOptions{
		CRDs: []v1.CustomResourceDefinition{
			*olmcrds.ClusterServiceVersion(),
		},
		Paths: []string{
			"../../config/deploy/reference.addons.managed.openshift.io_referenceaddons.yaml",
		},
		Scheme: _scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	By("Initializing k8s client")

	_client, err = client.New(_cfg, client.Options{
		Scheme: _scheme,
	})
	Expect(err).ToNot(HaveOccurred())
})

func cleanup() {
	_cancel()

	By("Stopping the test environment")

	Expect(_testEnv.Stop()).Should(Succeed())
}
