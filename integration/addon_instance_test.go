package integration

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	av1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	addoninstance "github.com/openshift/addon-operator/pkg/client"

	//"github.com/openshift/reference-addon/internal/controllers/addoninstance"
	internaltesting "github.com/openshift/reference-addon/internal/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Apply Network Policies Phase", func() {
	var (
		ctx                  context.Context
		cancel               context.CancelFunc
		deleteLabel          string
		deleteLabelGen       = nameGenerator("ref-test-label")
		addonInstanceName    string
		addonInstanceNameGen = nameGenerator("ai-test-name")
		//addonInstanceNamespace string
		//addonInstanceNamespaceGen = nameGenerator("ai-test-namespace")
		namespace       string
		namespaceGen    = nameGenerator("ref-test-namespace")
		operatorName    string
		operatorNameGen = nameGenerator("ref-test-name")
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		addonInstanceName = addonInstanceNameGen()
		deleteLabel = deleteLabelGen()
		//addonInstanceNamespace = addonInstanceNamespaceGen()
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
		)

		session, err := Start(manager, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		By("Creating the addon namespace")

		ns := addonNamespace(namespace)
		//ains := addonNamespace(addonInstanceNamespace)
		addonInstance := addonInstanceObject(addonInstanceName, namespace)

		_client.Create(ctx, &ns)
		//_client.Create(ctx, &ains)
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

	//Is addon instance available?
	// check that there is an addonInstance in the target namespace.
	When("Test starts", func() {
		It("AddonInstance object should be created", func() {
			addonInstance := addonInstanceObject(addonInstanceName, namespace)
			_client.EventuallyObjectExists(ctx, &addonInstance, internaltesting.WithTimeout(10*time.Second))
		})
	})

	When("Addon Instance Object Exists", func() {
		Context("Reference Addon Status Available'", func() {
			It("Addon Instance should report Availalbe condition", func() {
				addonInstance := addonInstanceObject(addonInstanceName, namespace)
				_client.EventuallyObjectExists(ctx, &addonInstance, internaltesting.WithTimeout(10*time.Second))

				var conditions []metav1.Condition
				conditions = append(conditions, addoninstance.NewAddonInstanceConditionInstalled(
					"True",
					av1alpha1.AddonInstanceInstalledReasonSetupComplete,
					"All Components Available",
				))
				//Expect(addonInstance.Status.Conditions).To(Equal(conditions))
				fmt.Printf("Conditions: %v\n", conditions)
				print(meta.IsStatusConditionTrue(addonInstance.Status.Conditions, av1alpha1.Available))
				fmt.Printf("Other Conditions: %v\n", addonInstance.Status.Conditions)
				//print(addonInstance.Status.Conditions[0])
				print("RESULT HEREREEREREEEEEEE")
			})
		})
	})
})

//Tests needed

// Check Default of 10s for AddonInstanceReconciler
// Assert().Equal(10*time.Second, addonInstance.Spec.HeartbeatUpdatePeriod.Duration)
//	})
//})
// Default of 10s is hardcoded in AddonInstanceReconciler

//Does it trigger off no reference addon actions

//Does it trigger when reference addon does do something?

//Is addon instance unavilable during uninstall?
