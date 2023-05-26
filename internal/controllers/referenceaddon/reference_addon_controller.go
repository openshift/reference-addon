package referenceaddon

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	refv1alpha1 "github.com/openshift/reference-addon/apis/reference/v1alpha1"
	"github.com/openshift/reference-addon/internal/controllers"
	"github.com/openshift/reference-addon/internal/metrics"
	opsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func NewReferenceAddonReconciler(client client.Client, getter ParameterGetter, opts ...ReferenceAddonReconcilerOption) (*ReferenceAddonReconciler, error) {
	var cfg ReferenceAddonReconcilerConfig

	cfg.Option(opts...)
	cfg.Default()

	signaler, err := NewConfigMapUninstallSignaler(
		client,
		WithAddonNamespace(cfg.AddonNamespace),
		WithOperatorName(cfg.OperatorName),
		WithDeleteLabel(cfg.DeleteLabel),
	)
	if err != nil {
		return nil, fmt.Errorf("initializing uninstall signaler: %w", err)
	}

	var (
		phaseLog                     = cfg.Log.WithName("phase")
		phaseApplyNetworkPoliciesLog = phaseLog.WithName("applyNetworkPolicies")
		phaseUninstallLog            = phaseLog.WithName("uninstall")
		uninstallerLog               = phaseUninstallLog.WithName("uninstaller")
	)

	return &ReferenceAddonReconciler{
		cfg:         cfg,
		client:      NewReferenceAddonClient(client),
		paramGetter: getter,
		orderedPhases: []Phase{
			NewPhaseUninstall(
				signaler,
				NewUninstallerImpl(
					NewCSVClientImpl(
						client,
						WithLog{Log: uninstallerLog.WithName("client")},
					),
					WithLog{Log: uninstallerLog},
				),
				WithLog{Log: phaseUninstallLog},
				WithAddonNamespace(cfg.AddonNamespace),
				WithOperatorName(cfg.OperatorName),
			),
			NewPhaseSendDummyMetrics(
				metrics.NewResponseSamplerImpl(),
				WithSampleURLs{"https://httpstat.us/503", "https://httpstat.us/200"},
			),
			NewPhaseApplyNetworkPolicies(
				NewNetworkPolicyClientImpl(client),
				WithLog{Log: phaseApplyNetworkPoliciesLog},
				WithPolicies{
					netv1.NetworkPolicy{
						ObjectMeta: metav1.ObjectMeta{
							Name:      generateIngressPolicyName(cfg.OperatorName),
							Namespace: cfg.AddonNamespace,
						},
						Spec: netv1.NetworkPolicySpec{
							PodSelector: metav1.LabelSelector{},
							PolicyTypes: []netv1.PolicyType{
								netv1.PolicyTypeIngress,
							},
						},
					},
				},
			),
		},
	}, nil
}

type ReferenceAddonReconciler struct {
	cfg ReferenceAddonReconcilerConfig

	client      ReferenceAddonClient
	paramGetter ParameterGetter

	orderedPhases []Phase
}

func (r *ReferenceAddonReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	params, err := r.paramGetter.GetParameters(ctx)
	if err != nil {
		// Log error and continue reconcilliation so subsequent phases
		// can fail if required parameters are missing.
		r.cfg.Log.Error(err, "unable to sync addon parameters")
	}

	addon, err := r.ensureReferenceAddon(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("ensuring ReferenceAddon: %w", err)
	}

	defer func() {
		if err := r.client.UpdateStatus(ctx, addon); err != nil {
			r.cfg.Log.Error(err, "updating ReferenceAddon status")
		}
	}()

	if !addon.HasConditionAvailable() {
		meta.SetStatusCondition(
			&addon.Status.Conditions,
			newAvailableCondition(
				refv1alpha1.ReferenceAddonAvailableReasonPending,
				"starting reconciliation",
			),
		)
	}

	phaseReq := PhaseRequest{
		Addon:  *addon,
		Params: params,
	}

	for _, p := range r.orderedPhases {
		res := p.Execute(ctx, phaseReq)

		for _, cond := range res.Conditions() {
			cond.ObservedGeneration = addon.Generation

			meta.SetStatusCondition(&addon.Status.Conditions, cond)
		}

		switch res.Status() {
		case PhaseStatusError:
			return ctrl.Result{}, res.Error()
		case PhaseStatusFailure:
			return ctrl.Result{Requeue: true}, nil
		case PhaseStatusBlocking:
			return ctrl.Result{}, nil
		}
	}

	meta.SetStatusCondition(&addon.Status.Conditions,
		newAvailableCondition(
			refv1alpha1.ReferenceAddonAvailableReasonReady,
			"all reconcile phases completed successfully",
		),
	)

	return ctrl.Result{}, nil
}

func (r *ReferenceAddonReconciler) ensureReferenceAddon(ctx context.Context) (*refv1alpha1.ReferenceAddon, error) {
	actual, err := r.client.CreateOrUpdate(ctx, r.desiredReferenceAddon())
	if err != nil {
		return nil, fmt.Errorf("creating/updating desired ReferenceAddon: %w", err)
	}

	return actual, nil
}

func (r *ReferenceAddonReconciler) SetupWithManager(mgr ctrl.Manager) error {
	desired := r.desiredReferenceAddon()
	requestObject := types.NamespacedName{
		Name:      desired.Name,
		Namespace: desired.Namespace,
	}

	refAddonHandler := handler.EnqueueRequestsFromMapFunc(func(context.Context, client.Object) []reconcile.Request {
		return []reconcile.Request{
			{
				NamespacedName: requestObject,
			},
		}
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&refv1alpha1.ReferenceAddon{}).
		WatchesRawSource(
			controllers.EnqueueObject(requestObject),
			refAddonHandler,
		).
		Owns(
			&netv1.NetworkPolicy{},
			builder.WithPredicates(controllers.HasName(generateIngressPolicyName(r.cfg.OperatorName))),
		).
		Watches(
			&opsv1alpha1.ClusterServiceVersion{},
			refAddonHandler,
			builder.WithPredicates(controllers.HasNamePrefix(r.cfg.OperatorName)),
		).
		Watches(
			&corev1.ConfigMap{},
			refAddonHandler,
			builder.WithPredicates(controllers.HasName(r.cfg.OperatorName)),
		).
		Watches(
			&corev1.Secret{},
			refAddonHandler,
			builder.WithPredicates(controllers.HasName(r.cfg.AddonParameterSecretname)),
		).
		Complete(r)
}

func (r *ReferenceAddonReconciler) desiredReferenceAddon() refv1alpha1.ReferenceAddon {
	return refv1alpha1.ReferenceAddon{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.cfg.OperatorName,
			Namespace: r.cfg.AddonNamespace,
		},
	}
}

type ReferenceAddonReconcilerConfig struct {
	Log logr.Logger

	AddonNamespace           string
	AddonParameterSecretname string
	OperatorName             string
	DeleteLabel              string
}

func (c *ReferenceAddonReconcilerConfig) Option(opts ...ReferenceAddonReconcilerOption) {
	for _, opt := range opts {
		opt.ConfigureReferenceAddonReconciler(c)
	}
}

func (c *ReferenceAddonReconcilerConfig) Default() {
	if c.Log.GetSink() == nil {
		c.Log = logr.Discard()
	}
}

type ReferenceAddonReconcilerOption interface {
	ConfigureReferenceAddonReconciler(*ReferenceAddonReconcilerConfig)
}

type ReferenceAddonClient interface {
	CreateOrUpdate(ctx context.Context, addon refv1alpha1.ReferenceAddon) (*refv1alpha1.ReferenceAddon, error)
	UpdateStatus(ctx context.Context, addon *refv1alpha1.ReferenceAddon) error
}

func NewReferenceAddonClient(client client.Client) *ReferenceAddonClientImpl {
	return &ReferenceAddonClientImpl{
		client: client,
	}
}

type ReferenceAddonClientImpl struct {
	client client.Client
}

func (c *ReferenceAddonClientImpl) CreateOrUpdate(ctx context.Context, addon refv1alpha1.ReferenceAddon) (*refv1alpha1.ReferenceAddon, error) {
	actualAddon := &refv1alpha1.ReferenceAddon{
		ObjectMeta: metav1.ObjectMeta{
			Name:      addon.Name,
			Namespace: addon.Namespace,
		},
	}

	if _, err := ctrl.CreateOrUpdate(ctx, c.client, actualAddon, func() error {
		actualAddon.Labels = labels.Merge(actualAddon.Labels, addon.Labels)
		actualAddon.Spec = addon.Spec

		return nil
	}); err != nil {
		return nil, fmt.Errorf("creating/updating ReferenceAddon: %w", err)
	}

	return actualAddon, nil
}

func (c *ReferenceAddonClientImpl) UpdateStatus(ctx context.Context, addon *refv1alpha1.ReferenceAddon) error {
	if err := c.client.Status().Update(ctx, addon); err != nil {
		return fmt.Errorf("updating ReferenceAddon status: %w", err)
	}

	return nil
}
