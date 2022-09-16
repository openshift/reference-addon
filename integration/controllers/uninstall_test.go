package controllers

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift/reference-addon/internal/controllers"
	opsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Uninstall Phase", Ordered, func() {
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

		By("Starting manager with controllers")

		mgr, err := ctrl.NewManager(_cfg, ctrl.Options{
			Scheme: _scheme,
		})
		Expect(err).ToNot(HaveOccurred())

		r, err := controllers.NewReferenceAddonReconciler(
			mgr.GetClient(),
			controllers.NewSecretParameterGetter(
				mgr.GetClient(),
				controllers.WithNamespace(namespace),
				controllers.WithName(parameterSecretName),
			),
			controllers.WithAddonNamespace(namespace),
			controllers.WithAddonParameterSecretName(parameterSecretName),
			controllers.WithOperatorName(operatorName),
			controllers.WithDeleteLabel(deleteLabel),
		)
		Expect(err).ToNot(HaveOccurred())

		Expect(r.SetupWithManager(mgr)).Should(Succeed())

		go func() {
			defer GinkgoRecover()

			Expect(mgr.Start(ctx)).Should(Succeed())
		}()

		By("Creating the addon namespace")

		ns := addonNamespace(namespace)

		Expect(_client.Create(ctx, &ns)).Should(Succeed())
		EventuallyObjectExists(ctx, &ns)

		By("Ensuring the addon CSV exists")

		csv := addonCSV(operatorName, namespace)

		Expect(_client.Create(ctx, &csv)).Should(Succeed())

		DeferCleanup(func() {
			defer cancel()

			By("Ensuring the addon CSV does not exist")

			err := _client.Delete(ctx, &csv)
			Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred())
		})
	})

	When("no uninstall ConfigMap exists", func() {
		It("should not remove the addon CSV", func() {
			cm := deleteConfigMap(operatorName, namespace)
			EventuallyObjectDoesNotExist(ctx, &cm)

			csv := addonCSV(operatorName, namespace)
			EventuallyObjectExists(ctx, &csv)
		})
	})

	When("uninstall ConfigMap exists", func() {
		BeforeEach(func() {
			By("Creating the uninstall ConfigMap")

			cm := deleteConfigMap(operatorName, namespace)
			Expect(_client.Create(ctx, &cm)).Should(Succeed())

			EventuallyObjectExists(ctx, &cm)

			DeferCleanup(func() {
				By("Deleting the uninstall ConfigMap")

				err := _client.Delete(ctx, &cm)
				Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred())
			})
		})

		Context("without a delete label", func() {
			It("should not remove the addon CSV", func() {
				csv := addonCSV(operatorName, namespace)
				EventuallyObjectExists(ctx, &csv)
			})
		})

		Context("with a delete label", func() {
			It("should remove the addon CSV", func() {
				updatedCM := deleteConfigMapWithLabel(operatorName, namespace, deleteLabel)
				Expect(_client.Update(ctx, &updatedCM)).Should(Succeed())

				csv := addonCSV(operatorName, namespace)
				EventuallyObjectDoesNotExist(ctx, &csv)
			})
		})
	})
})

func EventuallyObjectExists(ctx context.Context, obj client.Object) bool {
	get := func() error {
		return _client.Get(ctx, client.ObjectKeyFromObject(obj), obj)
	}

	return Eventually(get).Should(Succeed())
}

func EventuallyObjectDoesNotExist(ctx context.Context, obj client.Object) bool {
	get := func() error {
		return _client.Get(ctx, client.ObjectKeyFromObject(obj), obj)
	}

	return EventuallyWithOffset(1, get).ShouldNot(Succeed())
}

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

func nameGenerator(pfx string) func() string {
	i := 0

	return func() string {
		name := fmt.Sprintf("%s-%d", pfx, i)

		i++

		return name
	}
}
