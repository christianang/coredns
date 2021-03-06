package v1alpha1

import "k8s.io/apimachinery/pkg/runtime/schema"

// +kubebuilder:object:generate=true
// +groupName=coredns.io

var (
	GroupVersion         = schema.GroupVersion{Group: "coredns.io", Version: "v1alpha1"}
	GroupVersionResource = schema.GroupVersionResource{Group: "coredns.io", Version: "v1alpha1", Resource: "dnszones"}
)
