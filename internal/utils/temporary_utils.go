// TO BE REMOVED AFTER INTEGRATING ADDON INSTANCE CLIENT SDK

package utils

import (
	"context"
	"fmt"
	"reflect"
	"time"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func SetAddonInstanceCondition(ctx context.Context, cacheBackedKubeClient client.Client, condition metav1.Condition, addonName string, targetNamespace string) error {
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
	GetLatestHeartbeat() metav1.Condition
	SetLatestHeartbeat(metav1.Condition)
}

func SetupHeartbeatReporter(r ReconcilerWithHeartbeat, mgr manager.Manager, handleAddonInstanceConfigurationChanges func(addonsv1alpha1.AddonInstanceSpec)) error {
	addonName := r.GetAddonName()
	defaultHealthyHeartbeatConditionToBeginWith := metav1.Condition{
		Type:    "addons.managed.openshift.io/Healthy",
		Status:  "False",
		Reason:  "NoHeartbeatReported",
		Message: fmt.Sprintf("Addon '%s' hasn't reported any heartbeat yet", addonName),
	}
	// initialized with a healthy heartbeat condition corresponding to the addon
	r.SetLatestHeartbeat(defaultHealthyHeartbeatConditionToBeginWith)

	heartbeatReporterFunction := func(ctx context.Context) error {
		currentAddonInstanceConfiguration, err := GetAddonInstanceConfiguration(ctx, mgr.GetClient(), addonName, r.GetAddonTargetNamespace())
		if err != nil {
			return fmt.Errorf("failed to get the AddonInstance configuration corresponding to the Addon '%s': %w", addonName, err)
		}

		// Heartbeat reporter section: report a heartbeat at an interval ('currentAddonInstanceConfiguration.HeartbeatUpdatePeriod' seconds)
		for {
			currentHeartbeatCondition := r.GetLatestHeartbeat()
			if err := SetAddonInstanceCondition(ctx, mgr.GetClient(), currentHeartbeatCondition, addonName, r.GetAddonTargetNamespace()); err != nil {
				mgr.GetLogger().Error(err, "error occurred while setting the condition", "HeartbeatCondition", fmt.Sprintf("%+v", currentHeartbeatCondition))
				// waiting for heartbeat update period for executing the next iteration
				<-time.After(currentAddonInstanceConfiguration.HeartbeatUpdatePeriod.Duration)
				continue
			}

			// checking latest addonInstance configuration and seeing if it differs with current AddonInstance configuration
			latestAddonInstanceConfiguration, err := GetAddonInstanceConfiguration(ctx, mgr.GetClient(), addonName, r.GetAddonTargetNamespace())
			if err != nil {
				return fmt.Errorf("failed to get the AddonInstance configuration corresponding to the Addon '%s': %w", addonName, err)
			}
			if !reflect.DeepEqual(currentAddonInstanceConfiguration, latestAddonInstanceConfiguration) {
				currentAddonInstanceConfiguration = latestAddonInstanceConfiguration
				handleAddonInstanceConfigurationChanges(currentAddonInstanceConfiguration)
			}

			// waiting for heartbeat update period for executing the next iteration
			<-time.After(currentAddonInstanceConfiguration.HeartbeatUpdatePeriod.Duration)
		}
	}

	// coupling the heartbeat reporter function with the manager
	if err := mgr.Add(manager.RunnableFunc(heartbeatReporterFunction)); err != nil {
		return err
	}
	return nil
}

func GetAddonInstanceConfiguration(ctx context.Context, cacheBackedKubeClient client.Client, addonName string, targetNamespace string) (addonsv1alpha1.AddonInstanceSpec, error) {
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
