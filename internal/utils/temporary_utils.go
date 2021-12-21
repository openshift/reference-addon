// TO BE REMOVED AFTER INTEGRATING ADDON INSTANCE CLIENT SDK

package utils

import (
	"context"
	"fmt"
	"sync"
	"time"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	addonsv1apis "github.com/openshift/addon-operator/apis"
)

var (
	latestHeartbeat metav1.Condition
	mu              sync.Mutex
)

func setAddonInstanceCondition(ctx context.Context, cacheBackedKubeClient client.Client, condition metav1.Condition, addonName string, targetNamespace string) error {
	addonInstance, err := getAddonInstanceByAddon(ctx, cacheBackedKubeClient, addonName, targetNamespace)
	if err != nil {
		return fmt.Errorf("failed to fetch AddonInstance by the addon '%s': %w", addonName, err)
	}
	if err := upsertAddonInstanceCondition(ctx, cacheBackedKubeClient, &addonInstance, condition); err != nil {
		return fmt.Errorf("failed to update the conditions of the AddonInstance resource in the namespace %s: %w", addonInstance.Namespace, err)
	}
	return nil
}

type ReconcilerWithHeartbeat interface {
	reconcile.Reconciler
	GetAddonName() string            // so as to give the addon developer the freedom to return addon's name the way they want: through a direct hardcoded string or through env var populated by downwards API
	GetAddonTargetNamespace() string // so as to give the addon developer the freedom to return targetNamespace the way they want: through a direct hardcoded string or through env var populated by downwards API
	HandleAddonInstanceConfigurationChanges(newAddonInstanceSpec addonsv1alpha1.AddonInstanceSpec) error
	GetClient() client.Client
}

func SetAndSendHeartbeat(r ReconcilerWithHeartbeat, heartbeat metav1.Condition) error {
	// locking here so as to ensure the atomicity (hence, the happens-before relationship) of the operation: set `latestHeartbeat` + send the just-set `latestHeartbeat`
	mu.Lock()
	defer mu.Unlock()
	latestHeartbeat = heartbeat
	if err := setAddonInstanceCondition(context.TODO(), r.GetClient(), latestHeartbeat, r.GetAddonName(), r.GetAddonTargetNamespace()); err != nil {
		return fmt.Errorf("failed to send the heartbeat '%+v': %w", latestHeartbeat, err)
	}
	return nil
}

func SendLatestHeartbeat(ctx context.Context, r ReconcilerWithHeartbeat) error {
	mu.Lock()
	defer mu.Unlock()
	if err := setAddonInstanceCondition(ctx, r.GetClient(), latestHeartbeat, r.GetAddonName(), r.GetAddonTargetNamespace()); err != nil {
		return fmt.Errorf("failed to send the heartbeat '%+v': %w", latestHeartbeat, err)
	}
	return nil
}

func SetupHeartbeatReporter(r ReconcilerWithHeartbeat, mgr manager.Manager) error {
	_ = addonsv1apis.AddToScheme(mgr.GetScheme())

	addonName := r.GetAddonName()
	mu.Lock()
	if (latestHeartbeat == metav1.Condition{}) {
		latestHeartbeat = metav1.Condition{
			Type:    "addons.managed.openshift.io/Healthy",
			Status:  "False",
			Reason:  "NoHeartbeatReported",
			Message: fmt.Sprintf("Addon '%s' hasn't reported any heartbeat yet", addonName),
		}
	}
	mu.Unlock()

	heartbeatReporterFunction := func(ctx context.Context) error {
		currentAddonInstanceConfiguration, err := getAddonInstanceConfiguration(ctx, mgr.GetClient(), addonName, r.GetAddonTargetNamespace())
		if err != nil {
			return fmt.Errorf("failed to get the AddonInstance configuration corresponding to the Addon '%s': %w", addonName, err)
		}

		// Heartbeat reporter section: report a heartbeat at an interval ('currentAddonInstanceConfiguration.HeartbeatUpdatePeriod' seconds)
		for {
			if err := SendLatestHeartbeat(ctx, r); err != nil {
				mgr.GetLogger().Error(err, "error occurred while sending the latest heartbeat")
			}

			latestAddonInstanceConfiguration, err := getAddonInstanceConfiguration(ctx, mgr.GetClient(), addonName, r.GetAddonTargetNamespace())
			if err != nil {
				// TODO(ykukreja): report error instead and stop the heartbeat reporter loop? (instead of `return`, write `stop <- err`)
				mgr.GetLogger().Error(err, fmt.Sprintf("failed to get the AddonInstance configuration corresponding to the Addon '%s'", addonName))
			} else {
				currentAddonInstanceConfiguration = latestAddonInstanceConfiguration
			}

			// waiting for heartbeat update period for executing the next iteration
			<-time.After(currentAddonInstanceConfiguration.HeartbeatUpdatePeriod.Duration)
		}
	}

	addonInstanceConfigurationChangeWatcher := func(ctx context.Context) error {
		return SetupAddonInstanceConfigurationChangeWatcher(mgr, r)
	}

	// coupling the heartbeat reporter function with the manager
	if err := mgr.Add(manager.RunnableFunc(heartbeatReporterFunction)); err != nil {
		return err
	}

	// coupling the AddonInstance configuration change watcher with the manager
	if err := mgr.Add(manager.RunnableFunc(addonInstanceConfigurationChangeWatcher)); err != nil {
		return err
	}
	return nil
}

func getAddonInstanceConfiguration(ctx context.Context, cacheBackedKubeClient client.Client, addonName string, targetNamespace string) (addonsv1alpha1.AddonInstanceSpec, error) {
	addonInstance, err := getAddonInstanceByAddon(ctx, cacheBackedKubeClient, addonName, targetNamespace)
	if err != nil {
		return addonsv1alpha1.AddonInstanceSpec{}, fmt.Errorf("failed to fetch AddonInstance by the addon '%s': %w", addonName, err)
	}
	return addonInstance.Spec, nil
}

func getAddonInstanceByAddon(ctx context.Context, cacheBackedKubeClient client.Client, addonName string, targetNamespace string) (addonsv1alpha1.AddonInstance, error) {
	if targetNamespace == "" {
		return addonsv1alpha1.AddonInstance{}, fmt.Errorf("failed to fetch the target namespace of the addon. ADDON_TARGET_NAMESPACE env variable not found")
	}
	addonInstance := &addonsv1alpha1.AddonInstance{}
	if err := cacheBackedKubeClient.Get(ctx, types.NamespacedName{Name: addonsv1alpha1.DefaultAddonInstanceName, Namespace: targetNamespace}, addonInstance); err != nil {
		return addonsv1alpha1.AddonInstance{}, fmt.Errorf("failed to fetch the AddonInstance resource in the namespace %s: %w", targetNamespace, err)
	}
	return *addonInstance, nil
}

func upsertAddonInstanceCondition(ctx context.Context, cacheBackedKubeClient client.Client, addonInstance *addonsv1alpha1.AddonInstance, condition metav1.Condition) error {
	currentTime := metav1.Now()
	if condition.LastTransitionTime.IsZero() {
		condition.LastTransitionTime = currentTime
	}
	// TODO: confirm that it's not worth tracking the ObservedGeneration at per-condition basis
	meta.SetStatusCondition(&(*addonInstance).Status.Conditions, condition)
	addonInstance.Status.ObservedGeneration = (*addonInstance).Generation
	addonInstance.Status.LastHeartbeatTime = metav1.Now()
	return cacheBackedKubeClient.Status().Update(ctx, addonInstance)
}
