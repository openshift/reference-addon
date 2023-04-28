package integration

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"github.com/openshift/addon-operator/integration"
	internaltesting "github.com/openshift/reference-addon/internal/testing"
)

var _ = Describe("Apply Network Policies Phase", func() {
	var (
		ctx                       context.Context
		cancel                    context.CancelFunc
		deleteLabel               string
		deleteLabelGen            = nameGenerator("reference-test-label")
		namespace                 string
		namespaceGen              = nameGenerator("reference-test-namespace")
		addonInstanceNamespace    string
		addonInstanceNamespaceGen = nameGenerator("addon-instance-test-namespace")
		operatorName              string
		operatorNameGen           = nameGenerator("reference-test-operator")
		AddonInstanceName         string
		AddonInstanceNameGen      = nameGenerator("addon-instance-test-operator")
		parameterSecretName       string
		parameterSecretNameGen    = nameGenerator("reference-test-secret")
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		deleteLabel = deleteLabelGen()
		namespace = namespaceGen()
		addonInstanceNamespace = addonInstanceNamespaceGen()
		operatorName = operatorNameGen()
		AddonInstanceName = AddonInstanceNameGen()
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
		addonInstanceNS := addonNamespace(addonInstanceNamespace)

		_client.Create(ctx, &ns)

		rbac, err := getRBAC(namespace, managerGroup)
		Expect(err).ToNot(HaveOccurred())

		ai_rbac, err := getRBAC(addonInstanceNamespace, managerGroup)
		Expect(err).ToNot(HaveOccurred())

		for _, obj := range rbac {
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

	//Tests needed

	//Is addon instance available?

	// check that there is an addonInstance in the target namespace.
	When("Addon Instance is deployed", func() {
		It("Should be available", func() {
			addonInstance := &addonsv1alpha1.AddonInstance{}
			err := _client.Get(ctx, client.ObjectKey{
				Name:      AddonInstanceName,
				Namespace: AddonInstanceNamespace,
			}, addonInstance)
			s.Require().NoError(err)

			// Check Default of 10s for AddonInstanceReconciler
			s.Assert().Equal(10*time.Second, addonInstance.Spec.HeartbeatUpdatePeriod.Duration)
		})
	})
	// Default of 10s is hardcoded in AddonInstanceReconciler

	//Does it trigger off no reference addon actions

	//Does it trigger when reference addon does do something?

	//Is addon instance unavilable during uninstall?

	// TODO Addon Instance should not trigger due to no change in Reference Operator
	When("no parameter secret exists", func() {
		It("should not create a NetworkPolicy and addon instance should not trigger", func() {
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
				_client.EventuallyObjectExists(ctx, &np, internaltesting.WithTimeout(10*time.Second))
			})
		})
	})
})
