// TO BE REMOVED AFTER INTEGRATING ADDON INSTANCE CLIENT SDK

package addoninstancesdk

import (
	"context"
	"fmt"
	"sync"
	"time"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	addonsv1apis "github.com/openshift/addon-operator/apis"
)

var (
	latestHeartbeat metav1.Condition
	mu              sync.Mutex
)

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
		var addonInstanceConfigurationMutex sync.Mutex

		currentAddonInstanceConfiguration, err := getAddonInstanceConfiguration(ctx, mgr.GetClient(), addonName, r.GetAddonTargetNamespace())
		if err != nil {
			return fmt.Errorf("failed to get the AddonInstance configuration corresponding to the Addon '%s': %w", addonName, err)
		}

		stop := make(chan error)
		defer close(stop)

		// Heartbeat reporter section: report a heartbeat at an interval ('currentAddonInstanceConfiguration.HeartbeatUpdatePeriod' seconds)
		// 1. Each iteration spins up a go-routine, where that go-routine (concurrently) sends the heartbeat and syncs the 'currentAddonInstanceConfiguration' with the latest one.
		// 2. Spinning up go-routines here because that would end up ensuring that the for-loop runs almost exactly at a periodic rate of 'current heartbeatUpdatePeriod' seconds,
		// instead of running at a rate of (current heartbeatUpdatePeriod + time to send latest heartbeat + time to sync `currentAddonInstanceConfiguration` with latest one) seconds.
		for {
			go func(stop chan error) {
				if err := SendLatestHeartbeat(ctx, r); err != nil {
					mgr.GetLogger().Error(err, "error occurred while sending the latest heartbeat")
				}

				addonInstanceConfigurationMutex.Lock()
				defer addonInstanceConfigurationMutex.Unlock()

				latestAddonInstanceConfiguration, err := getAddonInstanceConfiguration(ctx, mgr.GetClient(), addonName, r.GetAddonTargetNamespace())
				if err != nil {
					// TODO(ykukreja): report error instead and stop the heartbeat reporter loop? (instead of `return`, write `stop <- err`)
					mgr.GetLogger().Error(err, fmt.Sprintf("failed to get the AddonInstance configuration corresponding to the Addon '%s'", addonName))
					return
				}
				currentAddonInstanceConfiguration = latestAddonInstanceConfiguration
			}(stop)

			select {
			// stop this heartbeat reporter loop if any of the above go-routines endup raising an error/exit
			case err := <-stop:
				mgr.GetLogger().Error(err, "failed to report certain heartbeats")
				return err
			default:
				// waiting for heartbeat update period for executing the next iteration
				<-time.After(currentAddonInstanceConfiguration.HeartbeatUpdatePeriod.Duration)
			}
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
