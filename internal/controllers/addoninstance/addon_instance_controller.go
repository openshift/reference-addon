package addoninstance

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/openshift/addon-operator/apis/addons/v1alpha1"
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

type StatusControllerReconciler struct {
	client              client.Client
	addonInstanceClient *addoninstance.AddonInstanceClientImpl
}

// TODO Grab Namespace at this stage
func NewStatusControllerReconciler(client client.Client) *StatusControllerReconciler {
	ctx, desiredAddonInstance := addoninstancereconcile.ensureReferenceAddon(&StatusControllerReconciler)
	return &StatusControllerReconciler{
		client:              client,
		addonInstanceClient: addoninstance.NewAddonInstanceClient(client),
		name:				 desiredAddonInstance.Name,
		namespace:			 desiredAddonInstance.Namespace,
	};
}

// Watch reference addon actions to trigger addon instance
func (r *StatusControllerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rv1alpha1.ReferenceAddon{}).
		Complete(r)
}

// Utilize info gathered from SetupWithManager to perform logic against 
func (s *StatusControllerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.Log.WithName("reference addon - addon instance")
	key := client.ObjectKey{
	//TODO Put values obtained from constructor here instead
		Namespace: req.Namespace,
		Name:      req.Name,
	}

	log.Info("find reference addon: ", req.Namespace, req.Name)
	referenceaddon := &rv1alpha1.ReferenceAddon{}
	if err := s.client.Get(ctx, key, referenceaddon); err != nil {
		return ctrl.Result{}, fmt.Errorf("getting Reference-Addon Object'%s/%s': %w", req.Namespace, req.Name, err)
	}

	log.Info("find addon instance: ", req.Namespace, req.Name)
	addonInstance := &av1alpha1.AddonInstance{}
	if err := s.client.Get(ctx, key, addonInstance); err != nil {
		return ctrl.Result{}, fmt.Errorf("getting AddonInstance Object'%s/%s': %w", req.Namespace, req.Name, err)
	}
	log.Info("found addon instance: ", addonInstance)

	//TODO Find Install State
	log.Info("find install condition: ", req.Namespace, req.Name)
	addonInstanceInstallState := addoninstance.NewAddonInstanceConditionInstalled(
		metav1.ConditionStatus("good"), 
		av1alpha1.AddonInstanceInstalledReason("Installed"), 
		"test")
	if err := s.client.Get(ctx, key, addonInstanceInstallState.); err != nil {
		return ctrl.Result{}, fmt.Errorf("getting AddonInstance Install State'%s/%s': %w", req.Namespace, req.Name, err)
	}
	log.Info("found install state: ", addonInstance)

	//TODO Find Degraded State
	log.Info("find install condition: ", req.Namespace, req.Name)
	if err := s.client.Get(ctx, key, addonInstance); err != nil {
		return ctrl.Result{}, fmt.Errorf("getting AddonInstance Object'%s/%s': %w", req.Namespace, req.Name, err)
	}
	log.Info("found install instance: ", addonInstance)

	getAddonInstanceUpdateCondition(referenceaddon, addonInstance, log)

	//TODO add conditions (if necessary)
	err := s.addonInstanceClient.SendPulse(ctx, *addonInstance, nil)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("successfully reconciled AddonInstance")

	return ctrl.Result{RequeueAfter: defaultRetryAfterTime}, nil
}

func getAddonInstanceUpdateCondition(referenceaddon *rv1alpha1.ReferenceAddon, addonInstance *av1alpha1.AddonInstance, log logr.Logger) {
	if referenceaddon.HasConditionAvailable() {
		log.Info("condition available")
	}
}
