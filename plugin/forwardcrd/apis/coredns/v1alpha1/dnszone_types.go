package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DNSZoneSpec represents the spec of a DNSZone
type DNSZoneSpec struct {
	ZoneName  string `json:"zoneName,omitempty"`
	ForwardTo string `json:"forwardTo,omitempty"`
}

// DNSZoneStatus represents the status of a DNSZone
type DNSZoneStatus struct {
}

// +kubebuilder:object:root=true

// +kubebuilder:printcolumn:name="Zone Name",type=string,JSONPath=`.spec.zoneName`
// +kubebuilder:printcolumn:name="Forward To",type=string,JSONPath=`.spec.forwardTo`

// DNSZone represents a zone that should have its DNS requests forwarded to an
// upstream DNS server within CoreDNS
type DNSZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DNSZoneSpec   `json:"spec,omitempty"`
	Status DNSZoneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DNSZoneList represents a list of DNSZones
type DNSZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DNSZone `json:"items"`
}
