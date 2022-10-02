package integration

import (
	"context"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	internaltesting "github.com/openshift/reference-addon/internal/testing"
	opsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Uninstall Phase", func() {
	var (
		ctx                    context.Context
		cancel                 context.CancelFunc
		deleteLabel            string
		deleteLabelGen         = nameGenerator("uninstall-test-label")
		namespace              string
		namespaceGen           = nameGenerator("uninstall-test-namespace")
		operatorName           string
		operatorNameGen        = nameGenerator("uninstall-test-operator")
		parameterSecretName    string
		parameterSecretNameGen = nameGenerator("uninstall-test-secret")
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		deleteLabel = deleteLabelGen()
		namespace = namespaceGen()
		operatorName = operatorNameGen()
		parameterSecretName = parameterSecretNameGen()

		By("Starting manager")

		manager := exec.Command(_binPath,
			"-namespace", namespace,
			"-delete-label", deleteLabel,
			"-operator-name", operatorName,
			"-parameter-secret-name", parameterSecretName,
			"-kubeconfig", _kubeConfigPath,
		)

		session, err := Start(manager, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		By("Creating the addon namespace")

		ns := addonNamespace(namespace)

		_client.Create(ctx, &ns)

		for _, obj := range generateRBAC("reference-addon", namespace) {
			_client.Create(ctx, obj)
		}

		By("Ensuring the addon CSV exists")

		csv := addonCSV(operatorName, namespace)

		_client.Create(ctx, &csv)

		DeferCleanup(func() {
			defer cancel()

			By("Stopping the managers")

			session.Interrupt()

			if usingExistingCluster() {
				By("Deleting test namspace")

				_client.Delete(ctx, &ns)
			}
		})
	})

	When("no uninstall ConfigMap exists", func() {
		It("should not remove the addon CSV", func() {
			cm := deleteConfigMap(operatorName, namespace)
			_client.EventuallyObjectDoesNotExist(ctx, &cm)

			csv := addonCSV(operatorName, namespace)
			_client.EventuallyObjectExists(ctx, &csv)
		})
	})

	When("uninstall ConfigMap exists", func() {
		BeforeEach(func() {
			By("Creating the uninstall ConfigMap")

			cm := deleteConfigMap(operatorName, namespace)
			_client.Create(ctx, &cm)

			DeferCleanup(func() {
				By("Deleting the uninstall ConfigMap")

				_client.Delete(ctx, &cm)
			})
		})

		Context("without a delete label", func() {
			It("should not remove the addon CSV", func() {
				csv := addonCSV(operatorName, namespace)
				_client.EventuallyObjectExists(ctx, &csv)
			})
		})

		Context("with a delete label", func() {
			It("should remove the addon CSV", func() {
				updatedCM := deleteConfigMapWithLabel(operatorName, namespace, deleteLabel)
				_client.Update(ctx, &updatedCM)

				csv := addonCSV(operatorName, namespace)
				_client.EventuallyObjectDoesNotExist(ctx, &csv, internaltesting.WithTimeout(2*time.Second))
			})
		})
	})
})

func addonNamespace(name string) corev1.Namespace {
	return corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func addonCSV(name, ns string) opsv1alpha1.ClusterServiceVersion {
	return opsv1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: opsv1alpha1.ClusterServiceVersionSpec{
			InstallStrategy: opsv1alpha1.NamedInstallStrategy{
				StrategySpec: opsv1alpha1.StrategyDetailsDeployment{
					DeploymentSpecs: []opsv1alpha1.StrategyDeploymentSpec{},
				},
			},
		},
	}
}

func deleteConfigMapWithLabel(name, ns, label string) corev1.ConfigMap {
	cm := deleteConfigMap(name, ns)
	cm.Labels = map[string]string{
		label: "true",
	}

	return cm
}

func deleteConfigMap(name, ns string) corev1.ConfigMap {
	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
}
