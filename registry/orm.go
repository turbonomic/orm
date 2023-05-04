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

package registry

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/turbonomic/orm/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	devopsv1alpha1 "github.com/turbonomic/orm/api/v1alpha1"
	ormutils "github.com/turbonomic/orm/utils"
)

var (
	messagePlaceHolder                 = "owner path located"
	errorMessageMultipleSourceForOwner = "allow 1 and only 1 input from owned.name, owned.selector, owner.labelSelector"
	errorMessageUnknownSelector        = "unknown selector"
)

func (or *ResourceMappingRegistry) SeekTopOwnersResourcePathsForOwnedResourcePath(owned devopsv1alpha1.ResourcePath) []devopsv1alpha1.ResourcePath {
	owners := []devopsv1alpha1.ResourcePath{owned}

	more := true
	for more {
		more = false
		old := owners
		owners = []devopsv1alpha1.ResourcePath{}
		for _, rp := range old {
			orme := or.retrieveORMEntryForOwned(rp.ObjectReference)
			for _, oe := range orme {
				for o, m := range oe {
					if m[rp.Path] == "" {
						continue
					}
					more = true
					owners = append(owners, devopsv1alpha1.ResourcePath{
						ObjectReference: o,
						Path:            m[rp.Path],
					})
				}
			}
			if !more {
				owners = append(owners, rp)
			}
		}
	}

	return owners

}

const predefinedOwnedResourceName = ".owned.name"
const predefinedParameterPlaceHolder = ".."

func (or *ResourceMappingRegistry) validateORMOwner(orm *devopsv1alpha1.OperatorResourceMapping) (*unstructured.Unstructured, error) {
	var err error
	if orm == nil {
		return nil, nil
	}

	// get owner
	var obj *unstructured.Unstructured
	if orm.Spec.Owner.Name != "" {
		objk := types.NamespacedName{
			Namespace: orm.Spec.Owner.Namespace,
			Name:      orm.Spec.Owner.Name,
		}
		if objk.Namespace == "" {
			objk.Namespace = orm.Namespace
		}
		obj, err = kubernetes.Toolbox.GetResourceWithGVK(orm.Spec.Owner.GroupVersionKind(), objk)
		if err != nil {
			rLog.Error(err, "failed to find owner", "owner", orm.Spec.Owner)
			return nil, err
		}
	} else {
		if orm.Spec.Owner.Namespace == "" {
			orm.Spec.Owner.Namespace = orm.Namespace
		}
		objs, err := kubernetes.Toolbox.GetResourceListWithGVKWithSelector(orm.Spec.Owner.GroupVersionKind(),
			types.NamespacedName{Namespace: orm.Spec.Owner.Namespace, Name: orm.Spec.Owner.Name}, &orm.Spec.Owner.LabelSelector)
		if err != nil || len(objs) == 0 {
			rLog.Error(err, "failed to find owner", "owner", orm.Spec.Owner)
			return nil, err
		}
		obj = &objs[0]
	}

	orm.Spec.Owner.Name = obj.GetName()
	orm.Spec.Owner.Namespace = obj.GetNamespace()

	return obj, nil
}

func (or *ResourceMappingRegistry) SetORMStatusForOwner(owner *unstructured.Unstructured, orm *devopsv1alpha1.OperatorResourceMapping) {
	var err error

	if orm != nil && !or.staticCheckORMSpec(orm) {
		err = kubernetes.Toolbox.OrmClient.Status().Update(context.TODO(), orm)
		if err != nil {
			rLog.Error(err, "retry status")
		}
		return
	}

	objref := corev1.ObjectReference{}
	objref.Name = owner.GetName()
	objref.Namespace = owner.GetNamespace()
	objref.SetGroupVersionKind(owner.GetObjectKind().GroupVersionKind())

	ormEntry := or.retrieveORMEntryForOwner(objref)
	// orm and owner are 1-1 mapping, ormEntry should be 1 only
	for ormk := range ormEntry {

		if orm == nil {
			orm = &devopsv1alpha1.OperatorResourceMapping{}
			err = kubernetes.Toolbox.OrmClient.Get(context.TODO(), ormk, orm)
			if err != nil {
				rLog.Error(err, "retrieving ", "orm", ormk)
			}
		}
		or.setORMStatus(owner, orm)
		err = kubernetes.Toolbox.OrmClient.Status().Update(context.TODO(), orm)
		if err != nil {
			rLog.Error(err, "retry status")
		}
	}
}

func (or *ResourceMappingRegistry) staticCheckPattern(p *devopsv1alpha1.Pattern, selectors map[string]metav1.LabelSelector, parameters map[string][]string) error {
	count := 0
	if p.OwnedResourcePath.Name != "" {
		count++
	}
	if p.OwnedResourcePath.Selector != nil {
		if selectors == nil {
			return errors.New(errorMessageUnknownSelector)
		}

		if _, ok := selectors[*p.OwnedResourcePath.Selector]; !ok {
			return errors.New(errorMessageUnknownSelector)
		}
		count++
	}

	if !reflect.DeepEqual(p.OwnedResourcePath.LabelSelector, metav1.LabelSelector{}) {
		count++
	}

	if count != 1 {
		return errors.New(errorMessageMultipleSourceForOwner)
	}

	return nil
}

func (or *ResourceMappingRegistry) staticCheckORMSpec(orm *devopsv1alpha1.OperatorResourceMapping) bool {
	var passed = true

	errMappingValues := []devopsv1alpha1.OwnerMappingValue{}
	// only allow 1 way to choose target
	for _, p := range orm.Spec.Mappings.Patterns {
		if err := or.staticCheckPattern(&p, orm.Spec.Mappings.Selectors, orm.Spec.Mappings.Parameters); err != nil {
			mv := devopsv1alpha1.OwnerMappingValue{
				OwnerPath:         p.OwnerPath,
				OwnedResourcePath: &p.OwnedResourcePath,
				Reason:            string(devopsv1alpha1.ORMStatusReasonOwnedResourceError),
				Message:           err.Error(),
			}
			errMappingValues = append(errMappingValues, mv)
			passed = false
		}
	}

	if !passed {
		orm.Status.OwnerMappingValues = errMappingValues
		orm.Status.State = devopsv1alpha1.ORMTypeError
	}

	return passed
}

func (or *ResourceMappingRegistry) setORMStatus(owner *unstructured.Unstructured, orm *devopsv1alpha1.OperatorResourceMapping) {

	// set owner info and clean previous error status at status top level
	orm.Status = devopsv1alpha1.OperatorResourceMappingStatus{}

	orm.Status.Owner.APIVersion = owner.GetAPIVersion()
	orm.Status.Owner.Kind = owner.GetKind()
	orm.Status.Owner.Namespace = owner.GetNamespace()
	orm.Status.Owner.Name = owner.GetName()
	orm.Status.Message = ""
	orm.Status.Reason = ""
	orm.Status.State = devopsv1alpha1.ORMTypeOK

	orm.Status.OwnerMappingValues = nil

	ownerRef := corev1.ObjectReference{
		Namespace: owner.GetNamespace(),
		Name:      owner.GetName(),
	}
	ownerRef.SetGroupVersionKind(owner.GetObjectKind().GroupVersionKind())

	oe := or.retrieveObjectEntryForOwnerAndORM(ownerRef, types.NamespacedName{
		Namespace: orm.GetNamespace(),
		Name:      orm.GetName(),
	})

	for o, mappings := range *oe {
		for op, sp := range mappings {
			owned := devopsv1alpha1.OwnedResourcePath{
				Path: sp,
			}
			owned.ObjectReference = o
			mapitem := PrepareMappingForObject(owner, op, &owned)
			orm.Status.OwnerMappingValues = append(orm.Status.OwnerMappingValues, *mapitem)
		}
	}

	or.validateOwnedResources(owner, orm)

	orm.Status.LastTransitionTime = &metav1.Time{
		Time: time.Now(),
	}
}

func (or *ResourceMappingRegistry) ValidateAndRegisterORM(orm *devopsv1alpha1.OperatorResourceMapping) (*devopsv1alpha1.OperatorResourceMapping, *unstructured.Unstructured, error) {
	var err error

	if orm == nil {
		return nil, nil, nil
	}

	var owner *unstructured.Unstructured
	owner, err = or.validateORMOwner(orm)
	if err != nil {
		return orm, owner, err
	}

	if orm.Spec.Mappings.Patterns == nil || len(orm.Spec.Mappings.Patterns) == 0 {
		return orm, owner, nil
	}

	allpatterns := []devopsv1alpha1.Pattern{}
	for _, p := range orm.Spec.Mappings.Patterns {

		if or.staticCheckPattern(&p, orm.Spec.Mappings.Selectors, orm.Spec.Mappings.Parameters) != nil {
			continue
		}

		k := types.NamespacedName{Namespace: p.OwnedResourcePath.Namespace, Name: p.OwnedResourcePath.Name}
		if k.Namespace == "" {
			k.Namespace = orm.Namespace
		}

		// TODO: avoid to retrieve same source repeatedly
		if k.Name != "" {
			allpatterns = append(allpatterns, *p.DeepCopy())
		} else {
			var srcObjs []unstructured.Unstructured
			selector := p.OwnedResourcePath.LabelSelector
			if p.OwnedResourcePath.Selector != nil && *p.OwnedResourcePath.Selector != "" {
				selector = orm.Spec.Mappings.Selectors[*p.OwnedResourcePath.Selector]
			}
			srcObjs, err = kubernetes.Toolbox.GetResourceListWithGVKWithSelector(p.OwnedResourcePath.GroupVersionKind(), k, &selector)
			if err != nil {
				rLog.Error(err, "listing resource", "source", p.OwnedResourcePath)
				continue
			}

			if len(srcObjs) == 0 {
				// add a fake pattern to generate error message, use name to carry label selector info
				fakep := *p.DeepCopy()
				fakep.OwnedResourcePath.ObjectReference.Name = p.OwnedResourcePath.ObjectLocator.LabelSelector.String()
				allpatterns = append(allpatterns, fakep)
				continue
			}
			for _, obj := range srcObjs {
				newp := *p.DeepCopy()
				newp.OwnedResourcePath.ObjectReference.Name = obj.GetName()
				newp.OwnedResourcePath.ObjectReference.Namespace = obj.GetNamespace()
				allpatterns = append(allpatterns, newp)
			}
		}
	}

	or.CleanupRegistryForORM(types.NamespacedName{
		Namespace: orm.Namespace,
		Name:      orm.Name,
	})

	for _, p := range allpatterns {

		patterns := populatePatterns(orm.Spec.Mappings.Parameters, p)
		for _, pattern := range patterns {
			err = or.registerOwnershipMapping(pattern.OwnerPath, pattern.OwnedResourcePath.Path,
				types.NamespacedName{Name: orm.Name, Namespace: orm.Namespace},
				orm.Spec.Owner.ObjectReference,
				p.OwnedResourcePath.ObjectReference)
			if err != nil {
				return orm, owner, err
			}
		}

	}

	return orm, owner, nil
}

func (or *ResourceMappingRegistry) CleanupRegistryForORM(orm types.NamespacedName) {

	if or.ownerRegistry != nil {
		cleanupORMInRegistry(or.ownerRegistry, orm)
	}
	if or.ownedRegistry != nil {
		cleanupORMInRegistry(or.ownedRegistry, orm)
	}
}

func (or *ResourceMappingRegistry) retrieveORMEntryForOwner(owner corev1.ObjectReference) ResourceMappingEntry {
	return retrieveResourceMappingEntryForObjectFromRegistry(or.ownerRegistry, owner)
}

func (or *ResourceMappingRegistry) retrieveORMEntryForOwned(owned corev1.ObjectReference) ResourceMappingEntry {
	return retrieveResourceMappingEntryForObjectFromRegistry(or.ownedRegistry, owned)
}

func (or *ResourceMappingRegistry) retrieveObjectEntryForOwnerAndORM(owner corev1.ObjectReference, orm types.NamespacedName) *ObjectEntry {
	return retrieveObjectEntryForObjectAndORMFromRegistry(or.ownerRegistry, owner, orm)
}

func populatePatterns(parameters map[string][]string, pattern devopsv1alpha1.Pattern) []devopsv1alpha1.Pattern {
	var allpatterns []devopsv1alpha1.Pattern

	pattern.OwnerPath = strings.ReplaceAll(pattern.OwnerPath, "{{"+predefinedOwnedResourceName+"}}", pattern.OwnedResourcePath.Name)
	pattern.OwnedResourcePath.Path = strings.ReplaceAll(pattern.OwnedResourcePath.Path, "{{"+predefinedOwnedResourceName+"}}", pattern.OwnedResourcePath.Name)

	allpatterns = append(allpatterns, pattern)

	if parameters == nil {
		parameters = make(map[string][]string)
	}
	parameters[predefinedParameterPlaceHolder] = []string{predefinedParameterPlaceHolder}
	var prevpatterns []devopsv1alpha1.Pattern
	for name, values := range parameters {
		prevpatterns = allpatterns
		allpatterns = []devopsv1alpha1.Pattern{}

		for _, p := range prevpatterns {
			for _, v := range values {
				newp := p.DeepCopy()
				newp.OwnerPath = strings.ReplaceAll(p.OwnerPath, "{{"+name+"}}", v)
				newp.OwnedResourcePath.Path = strings.ReplaceAll(p.OwnedResourcePath.Path, "{{"+name+"}}", v)
				allpatterns = append(allpatterns, *newp)
			}
		}
	}
	return allpatterns
}

func (or *ResourceMappingRegistry) validateOwnedResources(owner *unstructured.Unstructured, orm *devopsv1alpha1.OperatorResourceMapping) {
	var err error

	ownerRef := corev1.ObjectReference{
		Namespace: owner.GetNamespace(),
		Name:      owner.GetName(),
	}
	ownerRef.SetGroupVersionKind(owner.GetObjectKind().GroupVersionKind())

	oe := or.retrieveObjectEntryForOwnerAndORM(ownerRef, types.NamespacedName{
		Namespace: orm.GetNamespace(),
		Name:      orm.GetName(),
	})

	if oe == nil {
		rLog.Error(errors.New("failed to locate owner in registry"), "owner ref", ownerRef, "orm", orm)
		return
	}

	for resource, mappings := range *oe {
		resobj := &unstructured.Unstructured{}
		resobj.SetGroupVersionKind(resource.GroupVersionKind())

		resobj, err = kubernetes.Toolbox.GetResourceWithGVK(resource.GroupVersionKind(), types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name})
		if err != nil {
			for op := range mappings {
				for n, m := range orm.Status.OwnerMappingValues {
					if op == m.OwnerPath {
						orm.Status.OwnerMappingValues[n].Message = "Failed to locate owned resource: " + resource.String()
						orm.Status.OwnerMappingValues[n].Reason = string(devopsv1alpha1.ORMStatusReasonOwnedResourceError)
					}
				}
			}
			continue
		}

		for op, sp := range mappings {
			testValue := ormutils.PrepareRawExtensionFromUnstructured(resobj, sp)
			for n, m := range orm.Status.OwnerMappingValues {
				if op == m.OwnerPath {
					if testValue == nil && orm.Status.OwnerMappingValues[n].Message == messagePlaceHolder {
						orm.Status.OwnerMappingValues[n].Message = "Failed to locate mapping path " + sp + " in owned resource"
						orm.Status.OwnerMappingValues[n].Reason = string(devopsv1alpha1.ORMStatusReasonOwnedResourceError)
					} else if testValue != nil && orm.Status.OwnerMappingValues[n].Message == messagePlaceHolder {
						orm.Status.OwnerMappingValues[n].Message = ""
						orm.Status.OwnerMappingValues[n].Reason = ""
					} else if testValue != nil && orm.Status.OwnerMappingValues[n].Reason == string(devopsv1alpha1.ORMStatusReasonOwnedResourceError) {
						orm.Status.OwnerMappingValues[n].Message = ""
						orm.Status.OwnerMappingValues[n].Reason = ""
					}
				}
			}
		}
	}

}

func PrepareMappingForObject(obj *unstructured.Unstructured, objPath string, owned *devopsv1alpha1.OwnedResourcePath) *devopsv1alpha1.OwnerMappingValue {
	mapitem := devopsv1alpha1.OwnerMappingValue{}
	mapitem.OwnerPath = objPath

	mapitem.Value = ormutils.PrepareRawExtensionFromUnstructured(obj, objPath)
	if mapitem.Value == nil {
		mapitem.Message = "Failed to locate ownerPath in owner"
		mapitem.Reason = string(devopsv1alpha1.ORMStatusReasonOwnerError)
		return &mapitem
	}

	mapitem.Message = messagePlaceHolder
	mapitem.OwnedResourcePath = owned

	return &mapitem
}
