package addoninstance

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	av1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	addoninstance "github.com/openshift/addon-operator/pkg/client"
	rv1alpha1 "github.com/openshift/reference-addon/apis/reference/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// Default timeout when we do a manual RequeueAfter
	defaultRetryAfterTime = 10 * time.Second
)

// Global Flag for Install State
var installState int

type StatusControllerReconciler struct {
	client              client.Client
	addonInstanceClient *addoninstance.AddonInstanceClientImpl
}

// Review Grab Namespace at this stage (TO BE FIXED)
func NewStatusControllerReconciler(client client.Client) *StatusControllerReconciler {
	ctx, desiredAddonInstance := addoninstancereconcile.ensureReferenceAddon(&StatusControllerReconciler)
	return &StatusControllerReconciler{
		client:              client,
		addonInstanceClient: addoninstance.NewAddonInstanceClient(client),
		name:                desiredAddonInstance.Name,
		namespace:           desiredAddonInstance.Namespace,
	}
}

// Watch reference addon actions to trigger addon instance
func (r *StatusControllerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rv1alpha1.ReferenceAddon{}).
		Complete(r)
}

// Utilize info gathered from SetupWithManager to perform logic against
func (s *StatusControllerReconciler) Reconcile(ctx context.Context) (ctrl.Result, error) {
	log := log.Log.WithName("reference addon - addon instance")
	addonClient := NewStatusControllerReconciler(s.client)
	key := client.ObjectKey{
		//Review Put values obtained from constructor here instead
		Namespace: addonClient.namespace,
		Name:      addonClient.name,
	}

	log.Info("find reference addon: ", key.Namespace, key.Name)
	referenceaddon := &rv1alpha1.ReferenceAddon{}
	if err := s.client.Get(ctx, key, referenceaddon); err != nil {
		return ctrl.Result{}, fmt.Errorf("getting Reference-Addon Object'%s/%s': %w", key.Namespace, key.Name, err)
	}

	log.Info("find addon instance: ", key.Namespace, key.Name)
	addonInstance := &av1alpha1.AddonInstance{}
	if err := s.client.Get(ctx, key, addonInstance); err != nil {
		return ctrl.Result{}, fmt.Errorf("getting AddonInstance Object'%s/%s': %w", key.Namespace, key.Name, err)
	}
	log.Info("found addon instance: ", addonInstance)

	getAddonInstanceUpdateCondition(referenceaddon, addonInstance, log)

	var updatedInstance av1alpha1.AddonInstance

	//Condition for install state successful
	installSuccess := []metav1.Condition{
		addoninstance.NewAddonInstanceConditionInstalled(
			"True",
			av1alpha1.AddonInstanceInstalledReasonSetupComplete,
			"All components up",
		),
	}

	degradedState := []metav1.Condition{
		addoninstance.NewAddonInstanceConditionDegraded(
			"True",
			string(av1alpha1.AddonInstanceConditionDegraded),
			"Service Degraded",
		),
	}
	// Send Pulse to gather info from installation of addon instance
	err := s.addonInstanceClient.SendPulse(ctx, *addonInstance, nil)
	if err != nil {
		err := s.client.Get(ctx, client.ObjectKeyFromObject(addonInstance), &updatedInstance)
		if err != nil {
			// Check install state 0 = not installed, 1 = installed
			for i := 0; i < len(updatedInstance.Status.Conditions); i++ {
				if installState == 0 {
					log.Info("Checking for Installed State: ", key.Namespace, key.Name)
					if updatedInstance.Status.Conditions[i] == installSuccess[0] {
						installState = 1
						// Write that it was installed for first time as this will be skipped after each time
						return ctrl.Result{}, nil
						// Not installed throw error
					} else {
						return ctrl.Result{}, fmt.Errorf("AddonInstance not Installed '%s/%s': %w", key.Namespace, key.Name, err)
					}
				}
				// Check Degraded State
				log.Info("Checking for Degraded State: ", key.Namespace, key.Name)
				if updatedInstance.Status.Conditions[i] == degradedState[0] {
					return ctrl.Result{}, fmt.Errorf("AddonInstance service is Degraded '%s/%s': %w", key.Namespace, key.Name, err)
				}
			}
			log.Info("successfully reconciled AddonInstance")
			return ctrl.Result{}, nil
		}
	}
	return ctrl.Result{}, fmt.Errorf("Failed getting AddonInstance State '%s/%s': %w", key.Namespace, key.Name, err)
}

func getAddonInstanceUpdateCondition(referenceaddon *rv1alpha1.ReferenceAddon, addonInstance *av1alpha1.AddonInstance, log logr.Logger) {
	if referenceaddon.HasConditionAvailable() {
		log.Info("condition available")
	}
}
