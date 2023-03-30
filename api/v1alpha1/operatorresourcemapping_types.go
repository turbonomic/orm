/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Reference to object by name or by label
type ObjectLocator struct {
	// if namespace is empty, use the owner's namespace as default;
	corev1.ObjectReference `json:",inline"`

	// if ObjectReference.name is provided use the name, otherwise, use this label selector to find target resource(s)
	metav1.LabelSelector `json:",inline"`
}

type OwnedResourcePath struct {
	// path to the field inside the owned resource
	Path string `json:"path"`

	// identify owned resources by object reference of label
	// if more than 1 resources matching the selector, all of them are included
	ObjectLocator `json:",inline"`
}

type Pattern struct {
	// path to the location in owner resource, also serves as key of this pattern
	OwnerPath string `json:"ownerPath"`

	// indicates which value should be mapped
	OwnedResourcePath OwnedResourcePath `json:"owned"`
}

type MappingPatterns struct {
	// patterns for orm controller to generate mappings
	Patterns []Pattern `json:"patterns,omitempty"`

	// parameters defined here can be used in owner and owned resource paths
	// user can also use .owner.name to refer owner's name without defining it
	Parameters map[string][]string `json:"parameters,omitempty"`
}

// OperatorResourceMappingSpec defines the desired state of OperatorResourceMapping
type OperatorResourceMappingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Owner is the resource owns the deployed resources
	// if name and namespace are not provided, use same one as orm cr
	Owner ObjectLocator `json:"owner"`

	// mapping patters defined for the owner
	Mappings MappingPatterns `json:"mappings,omitempty"`
}

type OwnerMappingValue struct {
	// path in owner resource
	OwnerPath string `json:"ownerPath"`
	// value of the path in owner resource
	Value *runtime.RawExtension `json:"value,omitempty"`
	// owned resource and the path
	OwnedResourcePath *OwnedResourcePath `json:"owned,omitempty"`

	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

type ORMStatusType string

const (
	ORMTypeOK    ORMStatusType = "ok"
	ORMTypeError ORMStatusType = "error"
)

type ORMStatusReason string

const (
	ORMStatusReasonOwnerError         ORMStatusReason = "OwnerError"
	ORMStatusReasonOwnedResourceError ORMStatusReason = "OwnedResourceError"
)

// OperatorResourceMappingStatus defines the observed state of OperatorResourceMapping
type OperatorResourceMappingStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	// state of ORM resource
	State ORMStatusType `json:"state,omitempty"`
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`

	// owner object reference
	Owner corev1.ObjectReference `json:"owner,omitempty"`

	// +optional
	// all mappings generated from the patterns defined in spec and their values
	OwnerMappingValues []OwnerMappingValue `json:"ownerValues,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=operatorresourcemappings,scope=Namespaced
//+kubebuilder:resource:path=operatorresourcemappings,shortName=orm;orms

// OperatorResourceMapping is the Schema for the operatorresourcemappings API
type OperatorResourceMapping struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OperatorResourceMappingSpec   `json:"spec,omitempty"`
	Status OperatorResourceMappingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OperatorResourceMappingList contains a list of OperatorResourceMapping
type OperatorResourceMappingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OperatorResourceMapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OperatorResourceMapping{}, &OperatorResourceMappingList{})
}
