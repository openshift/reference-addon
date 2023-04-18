package addoninstance

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	av1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	addoninstance "github.com/openshift/addon-operator/pkg/client"
	rv1alpha1 "github.com/openshift/reference-addon/apis/reference/v1alpha1"

	"k8s.io/apimachinery/pkg/api/meta"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type StatusControllerReconciler struct {
	cfg                 StatusControllerReconcilerConfig
	client              client.Client
	addonInstanceClient *addoninstance.AddonInstanceClientImpl
	installed           bool
}

// Grabbing namespace/name needs to be an option
func NewStatusControllerReconciler(client client.Client, opts ...StatusControllerReconcilerOption) *StatusControllerReconciler {
	var cfg StatusControllerReconcilerConfig

	cfg.Option(opts...)
	cfg.Default()

	return &StatusControllerReconciler{
		cfg:                 cfg,
		client:              client,
		addonInstanceClient: addoninstance.NewAddonInstanceClient(client),
	}
}

type StatusControllerReconcilerConfig struct {
	Log                       logr.Logger
	statusControllerNamespace string
	statusControllerName      string
	referenceAddonNamespace   string
	referenceAddonName        string
	retryAfterTime            time.Duration
}

type StatusControllerReconcilerOption interface {
	ConfigureStatusControllerReconciler(*StatusControllerReconcilerConfig)
}

// Status controller option
func (c *StatusControllerReconcilerConfig) Option(opts ...StatusControllerReconcilerOption) {
	for _, opt := range opts {
		opt.ConfigureStatusControllerReconciler(c)
	}
}

func (c *StatusControllerReconcilerConfig) Default() {
	if c.Log.GetSink() == nil {
		c.Log = logr.Discard()
	}
	if c.retryAfterTime == 0 {
		c.retryAfterTime = 10 * time.Second
	}
}

// Watch reference addon actions to trigger addon instance
func (r *StatusControllerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rv1alpha1.ReferenceAddon{}).
		Complete(r)
}

// Utilize info gathered from SetupWithManager to perform logic against
func (s *StatusControllerReconciler) Reconcile(ctx context.Context, req reconcile.Request) (ctrl.Result, error) {
	log := log.Log.WithName("reference addon - addon instance")

	// Create reference addon key
	referenceAddonKey := client.ObjectKey{
		Namespace: s.cfg.referenceAddonNamespace,
		Name:      s.cfg.referenceAddonName,
	}

	//Build reference-addon constructor with options to grab namespace/name (do it for Addon-instance as well)
	// Obtain current reference addon state
	log.Info("find reference addon: ", referenceAddonKey.Namespace, referenceAddonKey.Name)
	referenceAddon := &rv1alpha1.ReferenceAddon{}
	if err := s.client.Get(ctx, referenceAddonKey, referenceAddon); err != nil {
		return ctrl.Result{}, fmt.Errorf("getting Reference-Addon Object'%s/%s': %w", referenceAddonKey.Namespace, referenceAddonKey.Name, err)
	}
	log.Info("found reference addon: ", referenceAddon)

	// Reference addon is available
	if !s.installed && meta.IsStatusConditionTrue(referenceAddon.Status.Conditions, "Ready") {
		s.installed = true
		log.Info("Reference Addon Successfully Installed")
	}

	// Create addon instance key
	statusControllerKey := client.ObjectKey{
		Namespace: s.cfg.statusControllerNamespace,
		Name:      s.cfg.statusControllerName,
	}

	// Obtain current addon instance
	log.Info("find addon instance: ", statusControllerKey.Namespace, statusControllerKey.Name)
	addonInstance := &av1alpha1.AddonInstance{}
	if err := s.client.Get(ctx, statusControllerKey, addonInstance); err != nil {
		return ctrl.Result{}, fmt.Errorf("getting AddonInstance Object'%s/%s': %w", statusControllerKey.Namespace, statusControllerKey.Name, err)
	}
	log.Info("found addon instance: ", addonInstance)

	// update the status before sending pulse
	addonInstance.Status.Conditions = referenceAddon.Status.Conditions

	// Send Pulse to addon operator to report health of reference addon
	err := s.addonInstanceClient.SendPulse(ctx, *addonInstance, nil)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("successfully reconciled AddonInstance")

	return ctrl.Result{RequeueAfter: s.cfg.retryAfterTime}, nil
}
