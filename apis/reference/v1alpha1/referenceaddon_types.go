package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReferenceAddonSpec defines the desired state of ReferenceAddon.
type ReferenceAddonSpec struct {
}

// ReferenceAddonStatus defines the observed state of ReferenceAddon
type ReferenceAddonStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

type ReferenceAddonCondition string

func (c ReferenceAddonCondition) String() string {
	return string(c)
}

const (
	ReferenceAddonConditionAvailable ReferenceAddonCondition = "Available"
)

type ReferenceAddonAvailableReason string

func (r ReferenceAddonAvailableReason) String() string {
	return string(r)
}

func (r ReferenceAddonAvailableReason) Status() metav1.ConditionStatus {
	switch r {
	case ReferenceAddonAvailableReasonReady:
		return "True"
	case ReferenceAddonAvailableReasonPending:
		return "False"
	default:
		return "Unknown"
	}
}

const (
	ReferenceAddonAvailableReasonReady        ReferenceAddonAvailableReason = "Ready"
	ReferenceAddonAvailableReasonPending      ReferenceAddonAvailableReason = "Pending"
	ReferenceAddonAvailableReasonUninstalling ReferenceAddonAvailableReason = "Uninstalling"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ReferenceAddon struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReferenceAddonSpec   `json:"spec,omitempty"`
	Status ReferenceAddonStatus `json:"status,omitempty"`
}

func (a *ReferenceAddon) HasConditionAvailable() bool {
	condT := ReferenceAddonConditionAvailable.String()

	return meta.FindStatusCondition(a.Status.Conditions, condT) != nil
}

// ReferenceAddonList contains a list of ReferenceAddons
// +kubebuilder:object:root=true
type ReferenceAddonList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReferenceAddon `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ReferenceAddon{}, &ReferenceAddonList{})
}
