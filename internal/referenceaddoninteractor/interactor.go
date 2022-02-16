package referenceaddoninteractor

import (
	"context"
	"fmt"
	"sync"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"github.com/openshift/reference-addon/internal/addoninstancesdk"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	referenceAddonInteractorSingleton    *ReferenceAddonInteractor
	referenceAddonInteractorSingletonMux = &sync.Mutex{}
)

type ReferenceAddonInteractor struct {
	Client dynamic.Interface
}

// ensure that ReferenceAddonInteractor implements AddonInstanceInteractorClient interface
var _ addoninstancesdk.AddonInstanceInteractorClient = (*ReferenceAddonInteractor)(nil)

func InitializeReferenceAddonInteractorSingleton(kubeConfigAbsolutePath string) (ReferenceAddonInteractor, error) {
	if referenceAddonInteractorSingleton == nil {
		referenceAddonInteractorSingletonMux.Lock()
		defer referenceAddonInteractorSingletonMux.Unlock()
		if referenceAddonInteractorSingleton == nil {
			config, err := clientcmd.BuildConfigFromFlags("", kubeConfigAbsolutePath)
			if err != nil {
				return ReferenceAddonInteractor{}, fmt.Errorf("failed to parse the REST config from kubeconfig/in-cluster config: %w", err)
			}

			client, err := dynamic.NewForConfig(config)
			if err != nil {
				return ReferenceAddonInteractor{}, fmt.Errorf("failed to setup a client from the REST config: %w", err)
			}

			referenceAddonInteractorSingleton = &ReferenceAddonInteractor{
				Client: client,
			}
		}
	}
	return *referenceAddonInteractorSingleton, nil
}

func GetReferenceAddonInteractorSingleton() (ReferenceAddonInteractor, error) {
	if referenceAddonInteractorSingleton == nil {
		return ReferenceAddonInteractor{}, fmt.Errorf("'Interactor' not found to be initialised. Initialize it by calling `InitializeReferenceAddonInteractorSingleton(...)`")
	}
	return *referenceAddonInteractorSingleton, nil
}

func (r ReferenceAddonInteractor) GetAddonInstance(ctx context.Context, addonInstanceName, addonInstanceNamespace string) (*addonsv1alpha1.AddonInstance, error) {
	res := schema.GroupVersionResource{Group: "addons.managed.openshift.io", Version: "v1alpha1", Resource: "addoninstances"}
	unstructuredAddonInstance, err := r.Client.Resource(res).Namespace(addonInstanceNamespace).Get(ctx, addonInstanceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get the `addoninstances` resource %s/%s: %w", addonInstanceNamespace, addonInstanceName, err)
	}

	var addonInstance addonsv1alpha1.AddonInstance
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredAddonInstance.Object, &addonInstance); err != nil {
		return nil, fmt.Errorf("failed to parse the fetched `addoninstances` resource: %w", err)
	}

	return &addonInstance, nil
}

func (r ReferenceAddonInteractor) UpsertHeartbeatToAddonInstance(ctx context.Context, addonInstance *addonsv1alpha1.AddonInstance, condition metav1.Condition) error {
	currentTime := metav1.Now()
	if condition.LastTransitionTime.IsZero() {
		condition.LastTransitionTime = currentTime
	}
	// TODO: confirm that it's not worth tracking the ObservedGeneration at per-condition basis
	meta.SetStatusCondition(&(*addonInstance).Status.Conditions, condition)
	addonInstance.Status.ObservedGeneration = (*addonInstance).Generation
	addonInstance.Status.LastHeartbeatTime = metav1.Now()

	return r.UpdateAddonInstanceStatus(ctx, addonInstance)
}

func (r ReferenceAddonInteractor) UpdateAddonInstanceStatus(ctx context.Context, addonInstance *addonsv1alpha1.AddonInstance) error {
	unstructuredAddonInstance, err := runtime.DefaultUnstructuredConverter.ToUnstructured(addonInstance)
	if err != nil {
		return fmt.Errorf("failed to convert the addoninstance to unstructured form: %w", err)
	}

	res := schema.GroupVersionResource{Group: "addons.managed.openshift.io", Version: "v1alpha1", Resource: "addoninstances"}
	if _, err := r.Client.Resource(res).Namespace(addonInstance.Namespace).UpdateStatus(ctx, &unstructured.Unstructured{Object: unstructuredAddonInstance}, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update the status of the addoninstance resource: %w", err)
	}
	return nil
}
