package controllers

import (
	"context"
	"math/rand"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/reference-addon/internal/utils"
)

type ReferenceAddonReconciler struct {
	client.Client
	Log                          logr.Logger
	Scheme                       *runtime.Scheme
	HeartbeatCommunicatorChannel chan metav1.Condition
}

func (r *ReferenceAddonReconciler) Reconcile(
	ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("addon", req.NamespacedName.String())

	successfulCondition := metav1.Condition{
		Type:    "addons.managed.openshift.io/Healthy",
		Status:  "True",
		Reason:  "AllComponentsUp",
		Message: "Everything under reference-addon is working perfectly fine",
	}
	failureCondition := metav1.Condition{
		Type:    "addons.managed.openshift.io/Healthy",
		Status:  "False",
		Reason:  "ComponentsDown",
		Message: "Components X, Y and Z are down!",
	}

	// approximately 5/10 times report a failure condition (probability ~ 50%) and remaining number of times report a successful condition
	if rand.Intn(10) >= 5 {
		utils.CommunicateHeartbeat(r.HeartbeatCommunicatorChannel, failureCondition, log)
	} else {
		utils.CommunicateHeartbeat(r.HeartbeatCommunicatorChannel, successfulCondition, log)
	}

	return ctrl.Result{}, nil
}

func (r *ReferenceAddonReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Pod{}). // just a placeholder for now
		Complete(r)
}
