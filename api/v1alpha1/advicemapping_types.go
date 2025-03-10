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

type ResourcePath struct {
	corev1.ObjectReference `json:",inline"`
	Path                   string `json:"path"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type AdviceMappingItem struct {
	TargetResourcePath  ResourcePath `json:"target"`
	AdvisorResourcePath ResourcePath `json:"advisor"`
}

// AdviceMappingSpec defines the desired state of AdviceMapping
type AdviceMappingSpec struct {
	Mappings []AdviceMappingItem `json:"mappings"`
}

type Advice struct {
	// object reference and path in owner resource
	Owner ResourcePath `json:"owner"`
	// path in owner resource
	Target ResourcePath `json:"target"`
	// value of the path in advisor resource
	Value *runtime.RawExtension `json:"adviceValue,omitempty"`

	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

type AMStatusType string

const (
	AMTypeOK    AMStatusType = "ok"
	AMTypeError AMStatusType = "error"
)

// AdviceMappingStatus defines the observed state of AdviceMapping
type AdviceMappingStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	State AMStatusType `json:"state,omitempty"`
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`

	// +optional
	// all mappings generated from the patterns defined in spec and their values
	Advices []Advice `json:"advices,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=advicemappings,scope=Namespaced
//+kubebuilder:resource:path=advicemappings,shortName=am;ams

// AdviceMapping is the Schema for the advicemappings API
type AdviceMapping struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AdviceMappingSpec   `json:"spec,omitempty"`
	Status AdviceMappingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AdviceMappingList contains a list of AdviceMapping
type AdviceMappingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AdviceMapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AdviceMapping{}, &AdviceMappingList{})
}
