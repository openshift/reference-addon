package integration

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	internaltesting "github.com/openshift/reference-addon/internal/testing"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Apply Network Policies Phase", func() {
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

		DeferCleanup(func() {
			cancel()

			By("Stopping the managers")

			session.Interrupt()

			if usingExistingCluster() {
				By("Deleting test namspace")

				_client.Delete(ctx, &ns)
			}
		})
	})

	When("no parameter secret exists", func() {
		It("should not create a NetworkPolicy", func() {
			secret := addonParameterSecret(parameterSecretName, namespace)
			_client.EventuallyObjectDoesNotExist(ctx, &secret)

			np := addonNetworkPolicy(fmt.Sprintf("%s-ingress", operatorName), namespace)
			_client.EventuallyObjectDoesNotExist(ctx, &np)
		})
	})

	When("parameter secret exists", func() {
		BeforeEach(func() {
			By("Creating the parameter Secret")

			secret := addonParameterSecret(parameterSecretName, namespace)
			_client.Create(ctx, &secret)

			DeferCleanup(func() {
				By("Deleting the parameter Secret")

				_client.Delete(ctx, &secret)
			})
		})

		Context("ApplyNetworkPolicies set to 'nil'", func() {
			It("should not create a NetworkPolicy", func() {
				np := addonNetworkPolicy(fmt.Sprintf("%s-ingress", operatorName), namespace)
				_client.EventuallyObjectDoesNotExist(ctx, &np)
			})
		})

		Context("ApplyNetworkPolicies set to 'false'", func() {
			It("should not create a NetworkPolicy", func() {
				secret := addonParameterSecret(parameterSecretName, namespace)
				secret.Data = map[string][]byte{
					"applynetworkpolicies": []byte("false"),
				}
				_client.Update(ctx, &secret)

				np := addonNetworkPolicy(fmt.Sprintf("%s-ingress", operatorName), namespace)
				_client.EventuallyObjectDoesNotExist(ctx, &np)
			})
		})

		Context("ApplyNetworkPolicies set to 'true'", func() {
			It("should create a NetworkPolicy", func() {
				secret := addonParameterSecret(parameterSecretName, namespace)
				secret.Data = map[string][]byte{
					"applynetworkpolicies": []byte("true"),
				}
				_client.Update(ctx, &secret)

				np := addonNetworkPolicy(fmt.Sprintf("%s-ingress", operatorName), namespace)
				_client.EventuallyObjectExists(ctx, &np, internaltesting.WithTimeout(5*time.Second))
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
