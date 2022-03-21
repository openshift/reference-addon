package main

import (
	"context"

	addonsv1alpha1apis "github.com/openshift/addon-operator/apis"
	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// just putting this here instead of main.go's init() to clearly depict the changes it would take to incorporate the addonssdkclient
func init() {
	_ = addonsv1alpha1apis.AddToScheme(scheme)
}

type addonSDKClient struct {
	client client.Client
}

func NewAddonSDKClient(client client.Client) addonSDKClient {
	return addonSDKClient{client: client}
}

func (adosdkClient addonSDKClient) GetAddonInstance(ctx context.Context, namespacedName types.NamespacedName, output *addonsv1alpha1.AddonInstance) error {
	return adosdkClient.client.Get(ctx, namespacedName, output)
}

func (adosdkClient addonSDKClient) UpdateAddonInstanceStatus(ctx context.Context, target *addonsv1alpha1.AddonInstance) error {
	return adosdkClient.client.Status().Update(ctx, target)
}
