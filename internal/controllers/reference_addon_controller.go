package controllers

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	refapisv1alpha1 "github.com/openshift/reference-addon/apis/reference/v1alpha1"
)

type ReferenceAddonReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *ReferenceAddonReconciler) Reconcile(
	ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("addon", req.NamespacedName.String())

	// dummy code to indicate reconciliation of the reference-addon object
	if strings.HasPrefix(req.NamespacedName.Name, "redhat-") {
		log.Info("reconciling for a reference addon object prefixed by redhat- ")
	} else if strings.HasPrefix(req.NamespacedName.Name, "reference-addon") {
		log.Info("reconciling for a reference addon object named reference-addon")
	} else {
		log.Info("reconciling for a reference addon object not prefixed by redhat- or named reference-addon")
	}

	return ctrl.Result{}, nil
}

func (r *ReferenceAddonReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&refapisv1alpha1.ReferenceAddon{}).
		Complete(r)
}
