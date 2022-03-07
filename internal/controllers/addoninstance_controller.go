package controllers

import (
	"context"

	"github.com/go-logr/logr"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/openshift/reference-addon/internal/addonsdk"
)

type AddonInstanceReconciler struct {
	client.Client
	StatusReporter  *addonsdk.StatusReporter
	Log             logr.Logger
	TargetNamespace string
}

func (r *AddonInstanceReconciler) Reconcile(
	ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	if req.NamespacedName.Name != "addon-instance" {
		return ctrl.Result{}, nil
	}

	addonInstance := &addonsv1alpha1.AddonInstance{}
	if err := r.Get(ctx, req.NamespacedName, addonInstance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	r.Log.Info("reporting addon-instance spec change to status-reporter")
	if err := r.StatusReporter.ReportAddonInstanceSpecChange(ctx, *addonInstance); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *AddonInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	addonInstanceConfigurationChangePredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// ignore updates to .status in which case metadata.generation does not change
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&addonsv1alpha1.AddonInstance{}, builder.WithPredicates(addonInstanceConfigurationChangePredicate)).
		Complete(r)
}
