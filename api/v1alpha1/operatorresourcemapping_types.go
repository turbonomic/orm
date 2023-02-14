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

// Reference to the source object by name or by label
type ObjectLocator struct {
	// if namespace is empty, use the operand namespace as default;
	// if apiversion and kind are empty, se apps/Deployment as default
	corev1.ObjectReference `json:",inline"`

	// if ObjectReferene.name is provided use the name, otherwise, use this label selector to find target resource(s)
	// if more than 1 resoures matching the selector, all of them are included
	metav1.LabelSelector `json:",inline"`
}

type OwnedResourcePath struct {
	Path          string `json:"path"` // Path to the field inside the source resource
	ObjectLocator `json:",inline"`
}

type Pattern struct {
	// path to the location in operand, also serves as key of this pattern
	OwnerPath string `json:"ownerPath"`

	// indicates which value should be mapped
	OwnedResourcePath OwnedResourcePath `json:"owned"`
}

type MappingPatterns struct {
	Patterns   []Pattern           `json:"patterns,omitempty"`
	Parameters map[string][]string `json:"parameters,omitempty"`
}

type EnforcementMode string

const (
	EnforcementModeNone   EnforcementMode = "none"
	EnforcementModeOnce   EnforcementMode = "once"
	EnforcementModeAlways EnforcementMode = "always"
)

var EnforcementModeDefault = EnforcementModeOnce

// OperatorResourceMappingSpec defines the desired state of OperatorResourceMapping
type OperatorResourceMappingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Operand is the target to make actual changes
	// if name and namespace are not provided, use same one as orm cr
	Owner           ObjectLocator   `json:"owner"`
	EnforcementMode EnforcementMode `json:"enforcement,omitempty"`
	Mappings        MappingPatterns `json:"mappings,omitempty"`
}

type Mapping struct {
	OwnerPath string                `json:"ownerPath"`
	Value     *runtime.RawExtension `json:"value"`

	// Status of the condition, one of True, False, Unknown.
	Mapped corev1.ConditionStatus `json:"mapped"`
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
	State ORMStatusType `json:"state,omitempty"`
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`

	// +optional
	MappedPatterns []Mapping `json:"mappedPatterns,omitempty"`
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
