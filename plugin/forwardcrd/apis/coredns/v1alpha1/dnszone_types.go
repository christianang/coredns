package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DNSZoneSpec struct {
	ZoneName  string `json:"zoneName,omitempty"`
	ForwardTo string `json:"forwardTo,omitempty"`
}

type DNSZoneStatus struct {
}

// +kubebuilder:object:root=true

// +kubebuilder:printcolumn:name="Zone Name",type=string,JSONPath=`.spec.zoneName`
// +kubebuilder:printcolumn:name="Forward To",type=string,JSONPath=`.spec.forwardTo`
type DNSZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DNSZoneSpec   `json:"spec,omitempty"`
	Status DNSZoneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type DNSZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DNSZone `json:"items"`
}
