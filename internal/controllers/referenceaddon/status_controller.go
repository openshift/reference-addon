package referenceaddon

import (
	"context"
	"fmt"
	"time"

	av1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	addoninstance "github.com/openshift/addon-operator/pkg/client"
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

func NewStatusControllerReconciler(client client.Client) *StatusControllerReconciler {
	return &StatusControllerReconciler{
		client:              client,
		addonInstanceClient: addoninstance.NewAddonInstanceClient(client),
	}
}

func (r *StatusControllerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&av1alpha1.AddonInstance{}).
		Complete(r)
}

func (s *StatusControllerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.Log.WithName("reference addon - addon instance")
	key := client.ObjectKey{
		Namespace: req.Namespace,
		Name:      req.Name,
	}

	log.Info("find addon instance: ", req.Namespace, req.Name)
	addonInstance := &av1alpha1.AddonInstance{}
	if err := s.client.Get(ctx, key, addonInstance); err != nil {
		return ctrl.Result{}, fmt.Errorf("getting AddonInstance '%s/%s': %w", req.Namespace, req.Name, err)
	}

	log.Info("found addon instance: ", addonInstance)
	err := s.addonInstanceClient.SendPulse(ctx, *addonInstance, nil)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("successfully reconciled AddonInstance")

	return ctrl.Result{RequeueAfter: defaultRetryAfterTime}, nil
}
