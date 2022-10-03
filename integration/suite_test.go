package integration

import (
	"os"
	"path/filepath"
	"testing"

	refapis "github.com/openshift/reference-addon/apis"
	internaltesting "github.com/openshift/reference-addon/internal/testing"
	olmcrds "github.com/operator-framework/api/crds"
	opsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	managerUser  = "reference-addon"
	managerGroup = "reference-addon"
)

var (
	_binPath        string
	_client         *internaltesting.TestClient
	_kubeConfigPath string
	_testEnv        *envtest.Environment
)

// To run these tests with an external cluster
// set the following environment variables:
// USE_EXISTING_CLUSTER=true
// KUBECONFIG=<path_to_kube.config>.
// The external cluster must have authentication
// enabled on the API server.
func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "controllers suite")
}

var _ = BeforeSuite(func() {
	root, err := projectRoot()
	Expect(err).ToNot(HaveOccurred())

	By("Registering schemes")

	scheme := runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).Should(Succeed())
	Expect(opsv1alpha1.AddToScheme(scheme)).Should(Succeed())
	Expect(refapis.AddToScheme(scheme)).Should(Succeed())

	By("Starting test environment")

	_testEnv = &envtest.Environment{
		Scheme: scheme,
	}

	cfg, err := _testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	DeferCleanup(cleanup(_testEnv))

	By("Installing CRD's")

	_, err = envtest.InstallCRDs(cfg, envtest.CRDInstallOptions{
		CRDs: []v1.CustomResourceDefinition{
			*olmcrds.ClusterServiceVersion(),
		},
		Paths: []string{
			filepath.Join(root, "config", "deploy", "reference.addons.managed.openshift.io_referenceaddons.yaml"),
		},
		Scheme: scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	By("Initializing k8s client")

	client, err := client.New(cfg, client.Options{
		Scheme: scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	_client = internaltesting.NewTestClient(client)

	By("Building manager binary")

	_binPath, err = gexec.BuildWithEnvironment(
		filepath.Join(root, "cmd", "reference-addon-manager"),
		[]string{"CGO_ENABLED=0"},
	)
	Expect(err).ToNot(HaveOccurred())

	By("writing kube.config")

	user, err := _testEnv.AddUser(
		envtest.User{
			Name:   managerUser,
			Groups: []string{managerGroup},
		},
		nil,
	)
	Expect(err).ToNot(HaveOccurred())

	configFile, err := os.CreateTemp("", "refernce-addon-integration-*")
	Expect(err).ToNot(HaveOccurred())

	data, err := user.KubeConfig()
	Expect(err).ToNot(HaveOccurred())

	_, err = configFile.Write(data)
	Expect(err).ToNot(HaveOccurred())

	_kubeConfigPath = configFile.Name()
})

func cleanup(env *envtest.Environment) func() {
	return func() {
		By("Stopping the test environment")

		Expect(env.Stop()).Should(Succeed())

		By("Cleaning up test artifacts")

		gexec.CleanupBuildArtifacts()

		Expect(remove(_kubeConfigPath)).Should(Succeed())
	}
}

func usingExistingCluster() bool {
	return _testEnv.UseExistingCluster != nil && *_testEnv.UseExistingCluster
}
