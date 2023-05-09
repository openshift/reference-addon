package integration

import (
	"context"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	internaltesting "github.com/openshift/reference-addon/internal/testing"
)

var _ = Describe("Apply Network Policies Phase", func() {
	var (
		ctx             context.Context
		cancel          context.CancelFunc
		namespace       string
		namespaceGen    = nameGenerator("ai-test-namespace")
		operatorName    string
		operatorNameGen = nameGenerator("ai-test-operator")
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		namespace = namespaceGen()
		operatorName = operatorNameGen()

		By("Starting manager")

		manager := exec.Command(_binPath,
			"-namespace", namespace,
			"-operator-name", operatorName,
			"-kubeconfig", _kubeConfigPath,
		)

		session, err := Start(manager, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		By("Creating the addon namespace")

		ns := addonNamespace(namespace)
		addonInstance := addonInstanceObject(operatorName, namespace)

		_client.Create(ctx, &ns)
		_client.Create(ctx, &addonInstance)

		rbac, err := getRBAC(namespace, managerGroup)
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

	When("Test starts", func() {
		It("AddonInstance object should be available", func() {
			addonInstance := addonInstanceObject(operatorName, namespace)
			_client.EventuallyObjectExists(ctx, &addonInstance, internaltesting.WithTimeout(10*time.Second))
		})
	})
})

//Tests needed

//Is addon instance available?
// check that there is an addonInstance in the target namespace.
//When("Addon Instance is deployed", func() {
//	It("Should be available", func() {
//		addonInstance := &addonsv1alpha1.AddonInstance{}
//		err := aiClient.Get(ctx, client.ObjectKey{
//			Name:      AddonInstanceName,
//			Namespace: addonInstanceNS,
//		}, addonInstance)
//		_client.EventuallyObjectExists(ctx)

//secret := addonParameterSecret(parameterSecretName, namespace)
//_client.EventuallyObjectDoesNotExist(ctx, &secret)

// Check Default of 10s for AddonInstanceReconciler
// Assert().Equal(10*time.Second, addonInstance.Spec.HeartbeatUpdatePeriod.Duration)
//	})
//})
// Default of 10s is hardcoded in AddonInstanceReconciler

//Does it trigger off no reference addon actions

//Does it trigger when reference addon does do something?

//Is addon instance unavilable during uninstall?
