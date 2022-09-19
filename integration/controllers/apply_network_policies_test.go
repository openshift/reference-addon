package controllers

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift/reference-addon/internal/controllers"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Apply Network Policies Phase", Ordered, func() {
	var (
		ctx                    context.Context
		cancel                 context.CancelFunc
		deleteLabel            string
		deleteLabelGen         = nameGenerator("policies-test-label")
		namespace              string
		namespaceGen           = nameGenerator("policies-test-namespace")
		operatorName           string
		operatorNameGen        = nameGenerator("policies-test-operator")
		parameterSecretName    string
		parameterSecretNameGen = nameGenerator("policies-test-secret")
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

		DeferCleanup(cancel)
	})

	When("no parameter secret exists", func() {
		It("should not create a NetworkPolicy", func() {
			secret := addonParameterSecret(parameterSecretName, namespace)
			EventuallyObjectDoesNotExist(ctx, &secret)

			np := addonNetworkPolicy(fmt.Sprintf("%s-ingress", operatorName), namespace)
			EventuallyObjectDoesNotExist(ctx, &np)
		})
	})

	When("parameter secret exists", func() {
		BeforeEach(func() {
			By("Creating the parameter Secret")

			secret := addonParameterSecret(parameterSecretName, namespace)
			Expect(_client.Create(ctx, &secret)).Should(Succeed())

			EventuallyObjectExists(ctx, &secret)

			DeferCleanup(func() {
				By("Deleting the parameter Secret")

				err := _client.Delete(ctx, &secret)
				Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred())
			})
		})

		Context("ApplyNetworkPolicies set to 'nil'", func() {
			It("should not create a NetworkPolicy", func() {
				np := addonNetworkPolicy(fmt.Sprintf("%s-ingress", operatorName), namespace)
				EventuallyObjectDoesNotExist(ctx, &np)
			})
		})

		Context("ApplyNetworkPolicies set to 'false'", func() {
			It("should not create a NetworkPolicy", func() {
				secret := addonParameterSecret(parameterSecretName, namespace)
				secret.Data = map[string][]byte{
					"applynetworkpolicies": []byte("false"),
				}
				Expect(_client.Update(ctx, &secret)).Should(Succeed())

				np := addonNetworkPolicy(fmt.Sprintf("%s-ingress", operatorName), namespace)
				EventuallyObjectDoesNotExist(ctx, &np)
			})
		})

		Context("ApplyNetworkPolicies set to 'true'", func() {
			It("should create a NetworkPolicy", func() {
				secret := addonParameterSecret(parameterSecretName, namespace)
				secret.Data = map[string][]byte{
					"applynetworkpolicies": []byte("true"),
				}
				Expect(_client.Update(ctx, &secret)).Should(Succeed())

				np := addonNetworkPolicy(fmt.Sprintf("%s-ingress", operatorName), namespace)
				EventuallyObjectExists(ctx, &np)
			})
		})
	})
})

func addonParameterSecret(name, ns string) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
}

func addonNetworkPolicy(name, ns string) netv1.NetworkPolicy {
	return netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeIngress,
			},
		},
	}
}
