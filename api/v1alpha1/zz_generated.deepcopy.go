//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Advice) DeepCopyInto(out *Advice) {
	*out = *in
	out.Owner = in.Owner
	out.Target = in.Target
	if in.Value != nil {
		in, out := &in.Value, &out.Value
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	if in.LastTransitionTime != nil {
		in, out := &in.LastTransitionTime, &out.LastTransitionTime
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Advice.
func (in *Advice) DeepCopy() *Advice {
	if in == nil {
		return nil
	}
	out := new(Advice)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdviceMapping) DeepCopyInto(out *AdviceMapping) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdviceMapping.
func (in *AdviceMapping) DeepCopy() *AdviceMapping {
	if in == nil {
		return nil
	}
	out := new(AdviceMapping)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AdviceMapping) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdviceMappingItem) DeepCopyInto(out *AdviceMappingItem) {
	*out = *in
	out.TargetResourcePath = in.TargetResourcePath
	out.AdvisorResourcePath = in.AdvisorResourcePath
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdviceMappingItem.
func (in *AdviceMappingItem) DeepCopy() *AdviceMappingItem {
	if in == nil {
		return nil
	}
	out := new(AdviceMappingItem)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdviceMappingList) DeepCopyInto(out *AdviceMappingList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AdviceMapping, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdviceMappingList.
func (in *AdviceMappingList) DeepCopy() *AdviceMappingList {
	if in == nil {
		return nil
	}
	out := new(AdviceMappingList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AdviceMappingList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdviceMappingSpec) DeepCopyInto(out *AdviceMappingSpec) {
	*out = *in
	if in.Mappings != nil {
		in, out := &in.Mappings, &out.Mappings
		*out = make([]AdviceMappingItem, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdviceMappingSpec.
func (in *AdviceMappingSpec) DeepCopy() *AdviceMappingSpec {
	if in == nil {
		return nil
	}
	out := new(AdviceMappingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdviceMappingStatus) DeepCopyInto(out *AdviceMappingStatus) {
	*out = *in
	if in.Advices != nil {
		in, out := &in.Advices, &out.Advices
		*out = make([]Advice, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdviceMappingStatus.
func (in *AdviceMappingStatus) DeepCopy() *AdviceMappingStatus {
	if in == nil {
		return nil
	}
	out := new(AdviceMappingStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MappingPatterns) DeepCopyInto(out *MappingPatterns) {
	*out = *in
	if in.Patterns != nil {
		in, out := &in.Patterns, &out.Patterns
		*out = make([]Pattern, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make(map[string][]string, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
	if in.Selectors != nil {
		in, out := &in.Selectors, &out.Selectors
		*out = make(map[string]v1.LabelSelector, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MappingPatterns.
func (in *MappingPatterns) DeepCopy() *MappingPatterns {
	if in == nil {
		return nil
	}
	out := new(MappingPatterns)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ObjectLocator) DeepCopyInto(out *ObjectLocator) {
	*out = *in
	out.ObjectReference = in.ObjectReference
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = new(string)
		**out = **in
	}
	in.LabelSelector.DeepCopyInto(&out.LabelSelector)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ObjectLocator.
func (in *ObjectLocator) DeepCopy() *ObjectLocator {
	if in == nil {
		return nil
	}
	out := new(ObjectLocator)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatorResourceMapping) DeepCopyInto(out *OperatorResourceMapping) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatorResourceMapping.
func (in *OperatorResourceMapping) DeepCopy() *OperatorResourceMapping {
	if in == nil {
		return nil
	}
	out := new(OperatorResourceMapping)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OperatorResourceMapping) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatorResourceMappingList) DeepCopyInto(out *OperatorResourceMappingList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OperatorResourceMapping, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatorResourceMappingList.
func (in *OperatorResourceMappingList) DeepCopy() *OperatorResourceMappingList {
	if in == nil {
		return nil
	}
	out := new(OperatorResourceMappingList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OperatorResourceMappingList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatorResourceMappingSpec) DeepCopyInto(out *OperatorResourceMappingSpec) {
	*out = *in
	in.Owner.DeepCopyInto(&out.Owner)
	in.Mappings.DeepCopyInto(&out.Mappings)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatorResourceMappingSpec.
func (in *OperatorResourceMappingSpec) DeepCopy() *OperatorResourceMappingSpec {
	if in == nil {
		return nil
	}
	out := new(OperatorResourceMappingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatorResourceMappingStatus) DeepCopyInto(out *OperatorResourceMappingStatus) {
	*out = *in
	if in.LastTransitionTime != nil {
		in, out := &in.LastTransitionTime, &out.LastTransitionTime
		*out = (*in).DeepCopy()
	}
	out.Owner = in.Owner
	if in.OwnerMappingValues != nil {
		in, out := &in.OwnerMappingValues, &out.OwnerMappingValues
		*out = make([]OwnerMappingValue, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatorResourceMappingStatus.
func (in *OperatorResourceMappingStatus) DeepCopy() *OperatorResourceMappingStatus {
	if in == nil {
		return nil
	}
	out := new(OperatorResourceMappingStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OwnedResourcePath) DeepCopyInto(out *OwnedResourcePath) {
	*out = *in
	in.ObjectLocator.DeepCopyInto(&out.ObjectLocator)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OwnedResourcePath.
func (in *OwnedResourcePath) DeepCopy() *OwnedResourcePath {
	if in == nil {
		return nil
	}
	out := new(OwnedResourcePath)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OwnerMappingValue) DeepCopyInto(out *OwnerMappingValue) {
	*out = *in
	if in.Value != nil {
		in, out := &in.Value, &out.Value
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	if in.OwnedResourcePath != nil {
		in, out := &in.OwnedResourcePath, &out.OwnedResourcePath
		*out = new(OwnedResourcePath)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OwnerMappingValue.
func (in *OwnerMappingValue) DeepCopy() *OwnerMappingValue {
	if in == nil {
		return nil
	}
	out := new(OwnerMappingValue)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Pattern) DeepCopyInto(out *Pattern) {
	*out = *in
	in.OwnedResourcePath.DeepCopyInto(&out.OwnedResourcePath)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Pattern.
func (in *Pattern) DeepCopy() *Pattern {
	if in == nil {
		return nil
	}
	out := new(Pattern)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourcePath) DeepCopyInto(out *ResourcePath) {
	*out = *in
	out.ObjectReference = in.ObjectReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourcePath.
func (in *ResourcePath) DeepCopy() *ResourcePath {
	if in == nil {
		return nil
	}
	out := new(ResourcePath)
	in.DeepCopyInto(out)
	return out
}
