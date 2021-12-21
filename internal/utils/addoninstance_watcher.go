package utils

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type AddonInstanceReconciler struct {
	client.Client
	scheme                                  *runtime.Scheme
	log                                     logr.Logger
	targetNamespace                         string
	handleAddonInstanceConfigurationChanges func(newAddonInstanceSpec addonsv1alpha1.AddonInstanceSpec) error
}

func SetupAddonInstanceConfigurationChangeWatcher(mgr ctrl.Manager, addonReconciler ReconcilerWithHeartbeat) error {
	r := &AddonInstanceReconciler{
		Client:                                  mgr.GetClient(),
		log:                                     ctrl.Log.WithName("controllers").WithName("AddonInstanceWatcher-" + addonReconciler.GetAddonName()),
		scheme:                                  mgr.GetScheme(),
		targetNamespace:                         addonReconciler.GetAddonTargetNamespace(),
		handleAddonInstanceConfigurationChanges: addonReconciler.HandleAddonInstanceConfigurationChanges,
	}

	addonInstanceConfigurationChangePredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// AddonInstance CR to be watched should be only <target-namnespace>/addon-instance
			// ignore updates to .status in which case metadata.generation does not change
			return e.ObjectNew.GetName() == "addon-instance" && e.ObjectNew.GetNamespace() == r.targetNamespace && e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&addonsv1alpha1.AddonInstance{}, builder.WithPredicates(addonInstanceConfigurationChangePredicate)).
		Complete(r); err != nil {
		r.log.Error(err, fmt.Sprintf("unable to setup AddonInstance watcher for the addon '%s'", addonReconciler.GetAddonName()))
		return err
	}
	return nil
}

func (r *AddonInstanceReconciler) Reconcile(
	ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	addonInstance := &addonsv1alpha1.AddonInstance{}
	if err := r.Get(ctx, req.NamespacedName, addonInstance); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if err := r.handleAddonInstanceConfigurationChanges(addonInstance.Spec); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
