package addoninstancesdk

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
