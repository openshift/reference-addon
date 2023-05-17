package integration

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/types"
	av1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	addoninstance "github.com/openshift/addon-operator/pkg/client"
	internaltesting "github.com/openshift/reference-addon/internal/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Status Controller", func() {
	var (
		ctx                  context.Context
		cancel               context.CancelFunc
		deleteLabel          string
		deleteLabelGen       = nameGenerator("ref-test-label")
		addonInstanceName    string
		addonInstanceNameGen = nameGenerator("ai-test-name")
		namespace            string
		namespaceGen         = nameGenerator("ref-test-namespace")
		operatorName         string
		operatorNameGen      = nameGenerator("ref-test-name")
		heartbeatInterval    = 1 * time.Second
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		addonInstanceName = addonInstanceNameGen()
		deleteLabel = deleteLabelGen()
		namespace = namespaceGen()
		operatorName = operatorNameGen()

		By("Starting manager")

		manager := exec.Command(_binPath,
			"-addon-instance-name", addonInstanceName,
			"-addon-instance-namespace", namespace,
			"-namespace", namespace,
			"-delete-label", deleteLabel,
			"-operator-name", operatorName,
			"-kubeconfig", _kubeConfigPath,
			"-heartbeat-interval", heartbeatInterval.String(),
		)

		session, err := Start(manager, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		By("Creating the addon namespace")

		ns := addonNamespace(namespace)
		addonInstance := addonInstanceObject(addonInstanceName, namespace)

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

	When("Addon Instance Object Exists", func() {
		Context("Reference Addon Status Available'", func() {
			It("Addon Instance should report Availalbe condition", func() {
				addonInstance := addonInstanceObject(addonInstanceName, namespace)
				_client.EventuallyObjectExists(ctx, &addonInstance, internaltesting.WithTimeout(10*time.Second))

				expectedCondition := addoninstance.NewAddonInstanceConditionInstalled(
					"True",
					av1alpha1.AddonInstanceInstalledReasonSetupComplete,
					"All Components Available",
				)

				Eventually(func() []metav1.Condition {
					_client.Get(ctx, &addonInstance)

					return addonInstance.Status.Conditions
				}, 10*time.Second).Should(ContainElements(EqualCondition(expectedCondition)))

				fmt.Printf("%+v\n", addonInstance)

				Expect(addonInstance.Spec.HeartbeatUpdatePeriod.Duration).To(Equal(heartbeatInterval))
			})
		})
	})
})

// Check if conditions match
func EqualCondition(expected metav1.Condition) types.GomegaMatcher {
	return And(
		HaveField("Type", expected.Type),
		HaveField("Status", expected.Status),
		HaveField("Reason", expected.Reason),
		HaveField("Message", expected.Message),
	)
}
