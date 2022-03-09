package testutil

import (
	"context"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// `addonsdk.client` interface
type addonSdkClientMock struct {
	client.Client
}

func NewAddonSdkClientMock(client *Client) *addonSdkClientMock {
	return &addonSdkClientMock{
		Client: client,
	}
}

func (c *addonSdkClientMock) GetAddonInstance(ctx context.Context, namespacedName types.NamespacedName, output *addonsv1alpha1.AddonInstance) error {
	return c.Get(ctx, client.ObjectKey(namespacedName), output)
}

func (c *addonSdkClientMock) UpdateAddonInstanceStatus(ctx context.Context, target *addonsv1alpha1.AddonInstance) error {
	return c.Status().Update(ctx, target, &client.UpdateOptions{})
}
