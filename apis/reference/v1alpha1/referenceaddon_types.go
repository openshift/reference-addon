package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReferenceAddonSpec defines the desired state of ReferenceAddon.
type ReferenceAddonSpec struct {
	ReportSuccessfulStatus bool `json:"reportSuccessfulStatus,omitempty"`
}

// ReferenceAddonStatus defines the observed state of ReferenceAddon
type ReferenceAddonStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ReferenceAddon struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReferenceAddonSpec   `json:"spec,omitempty"`
	Status ReferenceAddonStatus `json:"status,omitempty"`
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
