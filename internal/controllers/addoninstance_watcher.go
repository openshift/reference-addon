package controllers

import (
	"context"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/openshift/reference-addon/internal/addonsdk"
)

type AddonInstanceWatcher struct {
	client.Client
	StatusReporter  addonsdk.StatusReporterClient
	Log             logr.Logger
	TargetNamespace string
}

func (r *AddonInstanceWatcher) Reconcile(
	ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	addonInstance := &addonsv1alpha1.AddonInstance{}
	if err := r.Get(ctx, req.NamespacedName, addonInstance); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	r.Log.Info("reporting addon-instance spec change to status-reporter")
	if err := r.StatusReporter.ReportAddonInstanceSpecChange(ctx, addonInstance); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *AddonInstanceWatcher) SetupWithManager(mgr ctrl.Manager) error {
	addonInstanceConfigurationChangePredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// AddonInstance CR to be watched should be only <target-namespace>/addon-instance
			// ignore updates to .status in which case metadata.generation does not change
			return e.ObjectNew.GetName() == "addon-instance" && e.ObjectNew.GetNamespace() == r.TargetNamespace && e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
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
