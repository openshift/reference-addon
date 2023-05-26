package status

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	av1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	addoninstance "github.com/openshift/addon-operator/pkg/client"
	rv1alpha1 "github.com/openshift/reference-addon/apis/reference/v1alpha1"
	"github.com/openshift/reference-addon/internal/controllers"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type StatusControllerReconciler struct {
	cfg                 StatusControllerReconcilerConfig
	client              client.Client
	addonInstanceClient *addoninstance.AddonInstanceClientImpl
}

// Grabbing namespace/name needs to be an option
func NewStatusControllerReconciler(client client.Client, opts ...StatusControllerReconcilerOption) (*StatusControllerReconciler, error) {
	var cfg StatusControllerReconcilerConfig

	cfg.Option(opts...)
	cfg.Default()

	return &StatusControllerReconciler{
		cfg:                 cfg,
		client:              client,
		addonInstanceClient: addoninstance.NewAddonInstanceClient(client),
	}, nil
}

type StatusControllerReconcilerConfig struct {
	Log logr.Logger

	AddonInstanceNamespace  string
	AddonInstanceName       string
	ReferenceAddonNamespace string
	ReferenceAddonName      string
	HeartBeatInterval       time.Duration
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
	if c.HeartBeatInterval == 0 {
		c.HeartBeatInterval = 10 * time.Second
	}
}

// Watch reference addon actions to trigger addon instance
func (r *StatusControllerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	requestObject := types.NamespacedName{
		Name:      r.cfg.AddonInstanceName,
		Namespace: r.cfg.AddonInstanceNamespace,
	}

	referenceAddonHandler := handler.EnqueueRequestsFromMapFunc(func(context.Context, client.Object) []reconcile.Request {
		return []reconcile.Request{
			{
				NamespacedName: requestObject,
			},
		}
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&av1alpha1.AddonInstance{}).
		Watches(
			&rv1alpha1.ReferenceAddon{},
			referenceAddonHandler,
			builder.WithPredicates(controllers.HasNamePrefix(r.cfg.ReferenceAddonName)),
		).
		Complete(r)
}

// Utilize info gathered from SetupWithManager to perform logic against
func (r *StatusControllerReconciler) Reconcile(ctx context.Context, req reconcile.Request) (ctrl.Result, error) {
	ai, err := r.getAddonInstance(ctx)
	if err != nil {
		r.cfg.Log.Error(err, "getting addon instance")

		return ctrl.Result{RequeueAfter: r.cfg.HeartBeatInterval}, nil
	}

	if ai.Spec.HeartbeatUpdatePeriod.Duration != r.cfg.HeartBeatInterval {
		r.cfg.Log.Info("patching heartbeat interval")

		if err := r.patchHeartbeatInterval(ctx, ai); err != nil {
			r.cfg.Log.Error(err, "patching heartbeat interval")

			return ctrl.Result{RequeueAfter: r.cfg.HeartBeatInterval}, nil
		}
	}

	refAddon, err := r.getReferenceAddon(ctx)
	if err != nil {
		r.cfg.Log.Error(err, "getting reference addon")

		return ctrl.Result{RequeueAfter: r.cfg.HeartBeatInterval}, nil
	}

	conditions := r.getConditions(refAddon)

	if err := r.addonInstanceClient.SendPulse(ctx, ai, addoninstance.WithConditions(conditions)); err != nil {
		r.cfg.Log.Error(err, "sending pulse to addon instance")

		return ctrl.Result{}, err
	}

	r.cfg.Log.Info("successfully reconciled AddonInstance")

	return ctrl.Result{RequeueAfter: r.cfg.HeartBeatInterval}, nil
}

func (r *StatusControllerReconciler) getAddonInstance(ctx context.Context) (av1alpha1.AddonInstance, error) {
	log := r.cfg.Log.WithValues(
		"namespace", r.cfg.AddonInstanceNamespace,
		"name", r.cfg.AddonInstanceName,
	)

	addonInstanceKey := client.ObjectKey{
		Namespace: r.cfg.AddonInstanceNamespace,
		Name:      r.cfg.AddonInstanceName,
	}

	var addonInstance av1alpha1.AddonInstance
	if err := r.client.Get(ctx, addonInstanceKey, &addonInstance); err != nil {
		return addonInstance, fmt.Errorf("getting addon instance: %w", err)
	}

	log.Info("found addon instance")

	return addonInstance, nil
}

func (r *StatusControllerReconciler) patchHeartbeatInterval(ctx context.Context, ai av1alpha1.AddonInstance) error {
	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"resourceVersion": ai.GetResourceVersion(),
		},
		"spec": map[string]interface{}{
			"heartbeatUpdatePeriod": metav1.Duration{
				Duration: r.cfg.HeartBeatInterval,
			},
		},
	}

	patchJson, err := json.Marshal(&patch)
	if err != nil {
		return fmt.Errorf("marshalling raw patch: %w", err)
	}

	return r.client.Patch(ctx, &ai, client.RawPatch(types.MergePatchType, patchJson))
}

func (r *StatusControllerReconciler) getReferenceAddon(ctx context.Context) (rv1alpha1.ReferenceAddon, error) {
	log := r.cfg.Log.WithValues(
		"namespace", r.cfg.ReferenceAddonNamespace,
		"name", r.cfg.ReferenceAddonName,
	)

	log.Info("getting reference addon")

	referenceAddonKey := client.ObjectKey{
		Namespace: r.cfg.ReferenceAddonNamespace,
		Name:      r.cfg.ReferenceAddonName,
	}

	var referenceAddon rv1alpha1.ReferenceAddon
	if err := r.client.Get(ctx, referenceAddonKey, &referenceAddon); err != nil {
		return referenceAddon, fmt.Errorf("getting reference addon: %w", err)
	}

	log.Info("found reference addon")

	return referenceAddon, nil
}

func (r *StatusControllerReconciler) getConditions(ra rv1alpha1.ReferenceAddon) []metav1.Condition {
	var conditions []metav1.Condition

	isAvailable := meta.IsStatusConditionTrue(
		ra.Status.Conditions,
		rv1alpha1.ReferenceAddonConditionAvailable.String(),
	)

	// Check if Reference addon is available for the first time
	if isAvailable {
		r.cfg.Log.Info("Reference Addon Successfully Installed")

		conditions = append(conditions, addoninstance.NewAddonInstanceConditionInstalled(
			"True",
			av1alpha1.AddonInstanceInstalledReasonSetupComplete,
			"All Components Available",
		))
	}

	return conditions
}
