package integration

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

var (
	_binPath        string
	_client         *internaltesting.TestClient
	_kubeConfigPath string
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

	testEnv := &envtest.Environment{
		Scheme: scheme,
	}

	cfg, err := testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	DeferCleanup(cleanup(testEnv))

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

	_binPath, err = gexec.Build(filepath.Join(root, "cmd", "reference-addon-manager"))
	Expect(err).ToNot(HaveOccurred())

	By("writing kube.config")

	user, err := testEnv.AddUser(
		envtest.User{
			Name:   "reference-addon",
			Groups: []string{"reference-addon"},
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

var errSetup = errors.New("test setup failed")

func projectRoot() (string, error) {
	var buf bytes.Buffer

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Stdout = &buf
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("determining top level directory from git: %w", errSetup)
	}

	return strings.TrimSpace(buf.String()), nil
}

func remove(path string) error {
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		return nil
	}

	return os.Remove(_kubeConfigPath)
}
