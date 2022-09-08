package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	refapisv1alpha1 "github.com/openshift/reference-addon/apis/reference/v1alpha1"
	"github.com/openshift/reference-addon/internal/controllers/phase"
	"github.com/openshift/reference-addon/internal/metrics"
	opsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		phaseLog                       = cfg.Log.WithName("phase")
		phaseApplyNetworkPoliciesLog   = phaseLog.WithName("applyNetworkPolicies")
		phaseSimulateReconciliationLog = phaseLog.WithName("simulateReconciliation")
		phaseUninstallLog              = phaseLog.WithName("uninstall")
	)

	return &ReferenceAddonReconciler{
		cfg:         cfg,
		paramGetter: getter,
		orderedPhases: []phase.Phase{
			NewPhaseUninstall(
				signaler,
				NewUninstallerImpl(
					client,
					NewCSVListerImpl(client),
					WithLog{Log: phaseUninstallLog.WithName("uninstaller")},
				),
				WithLog{Log: phaseUninstallLog},
				WithAddonNamespace(cfg.AddonNamespace),
				WithOperatorName(cfg.OperatorName),
			),
			NewPhaseSimulateReconciliation(
				WithLog{Log: phaseSimulateReconciliationLog},
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

	paramGetter ParameterGetter

	orderedPhases []phase.Phase
}

func (r *ReferenceAddonReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	params, err := r.paramGetter.GetParameters(ctx)
	if err != nil {
		// Log error and continue reconcilliation so subsequent phases
		// can fail if required parameters are missing.
		r.cfg.Log.Error(err, "unable to sync addon parameters")
	}

	phaseReq := phase.Request{
		Object: req.NamespacedName,
		Params: params,
	}

	for _, p := range r.orderedPhases {
		if res := p.Execute(ctx, phaseReq); res.Error() != nil {
			return ctrl.Result{}, res.Error()
		} else if !res.IsSuccess() {
			return ctrl.Result{Requeue: true}, nil
		} else if res.IsBlocking() {
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *ReferenceAddonReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&refapisv1alpha1.ReferenceAddon{}).
		Watches(
			&source.Kind{Type: &opsv1alpha1.ClusterServiceVersion{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(hasNamePrefix(r.cfg.OperatorName)),
		).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(hasName(r.cfg.OperatorName)),
		).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(hasName(r.cfg.AddonParameterSecretname)),
		).
		Watches(
			&source.Kind{Type: &netv1.NetworkPolicy{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(hasName(generateIngressPolicyName(r.cfg.OperatorName))),
		).
		Complete(r)
}

func hasNamePrefix(pfx string) predicate.Funcs {
	return predicate.NewPredicateFuncs(
		func(obj client.Object) bool {
			return strings.HasPrefix(obj.GetName(), pfx)
		},
	)
}

func hasName(name string) predicate.Funcs {
	return predicate.NewPredicateFuncs(
		func(obj client.Object) bool {
			return obj.GetName() == name
		},
	)
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
	if c.Log == nil {
		c.Log = logr.Discard()
	}
}

type ReferenceAddonReconcilerOption interface {
	ConfigureReferenceAddonReconciler(*ReferenceAddonReconcilerConfig)
}
