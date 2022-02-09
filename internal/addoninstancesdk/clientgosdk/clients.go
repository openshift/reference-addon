package clientgosdk

import (
	"context"
	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// our tenants need to build a struct to implement this (here in reference addon, that struct is in `internal/referenceaddoninteractor/`)
// The tenant can choose to implement the following methods in whatever way they want depending on the client-go/controller-runtime they possess
type AddonInstanceInteractorClient interface {
	GetAddonInstance(ctx context.Context, addonInstanceName string, addonInstanceNamespace string) (*addonsv1alpha1.AddonInstance, error)
	// TODO: refine the following method
	AddonInstanceSpecChangeWatcher() (receiver chan bool, err error)
	UpdateAddonInstanceStatus(ctx context.Context, addonInstance *addonsv1alpha1.AddonInstance, condition metav1.Condition) error
}

// if the tenant wants a super-custom way to report heartbeats or just like DIY stuff, they can implement their own `AddonInstanceReporterClient` and utilise it
type AddonInstanceStatusReporterClient interface {
	Start(ctx context.Context) error
	SendHeartbeat(ctx context.Context, condition metav1.Condition) error
}