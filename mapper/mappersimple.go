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

package mapper

import (
	"context"
	"reflect"

	"github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/kubernetes"
	"github.com/turbonomic/orm/registry"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	messagePlaceHolder = "locating source path"

	msLog = ctrl.Log.WithName("mapper simple")

	mp *SimpleMapper
)

// OperatorResourceMappingReconciler reconciles a OperatorResourceMapping object
type SimpleMapper struct {
	registry.ORMRegistry

	watchingGVK map[schema.GroupVersionKind]bool
}

func (m *SimpleMapper) CleanupORM(key types.NamespacedName) {
	m.CleanupRegistryForORM(key)
}

func (m *SimpleMapper) MapORM(orm *v1alpha1.OperatorResourceMapping) error {

	var err error
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
			msLog.Error(err, "failed to find owner", "owner", orm.Spec.Owner)
			return err
		}
	} else {
		objs, err := kubernetes.Toolbox.GetResourceListWithGVKWithSelector(orm.Spec.Owner.GroupVersionKind(),
			types.NamespacedName{Namespace: orm.Spec.Owner.Namespace, Name: orm.Spec.Owner.Name}, &orm.Spec.Owner.LabelSelector)
		if err != nil || len(objs) == 0 {
			msLog.Error(err, "failed to find owner", "owner", orm.Spec.Owner)
			return err
		}
		obj = &objs[0]
	}

	err = RegisterORM(&m.ORMRegistry, orm)
	if err != nil {
		return err
	}

	if _, ok := m.watchingGVK[obj.GroupVersionKind()]; !ok {
		kubernetes.Toolbox.WatchResourceWithGVK(obj.GroupVersionKind(), cache.ResourceEventHandlerFuncs{
			AddFunc: func(new interface{}) {
				ownerobj := new.(*unstructured.Unstructured)
				m.mapForOwner(ownerobj)
			},
			UpdateFunc: func(old, new interface{}) {
				ownerobj := new.(*unstructured.Unstructured)
				m.mapForOwner(ownerobj)
			}})
		m.watchingGVK[obj.GroupVersionKind()] = true
	}

	m.setORMStatus(obj, orm)

	return err
}

func (m *SimpleMapper) mapForOwner(owner *unstructured.Unstructured) {
	var err error

	var orgStatus v1alpha1.OperatorResourceMappingStatus

	objref := corev1.ObjectReference{}
	objref.Name = owner.GetName()
	objref.Namespace = owner.GetNamespace()
	objref.SetGroupVersionKind(owner.GetObjectKind().GroupVersionKind())

	// set orm status if the object is owner
	orm := &v1alpha1.OperatorResourceMapping{}

	ormEntry := m.ORMRegistry.RetriveORMEntryForOwner(objref)

	for ormk := range ormEntry {

		err = kubernetes.Toolbox.OrmClient.Get(context.TODO(), ormk, orm)
		if err != nil {
			msLog.Error(err, "retrieving ", "orm", ormk)
		}
		orm.Status.DeepCopyInto(&orgStatus)
		m.setORMStatus(owner, orm)

		if !reflect.DeepEqual(orgStatus, orm.Status) {
			m.updateORMStatus(orm)
		}

	}
}

func (m *SimpleMapper) validateOwnedResources(owner *unstructured.Unstructured, orm *v1alpha1.OperatorResourceMapping) {
	var err error

	ownerRef := corev1.ObjectReference{
		Namespace: owner.GetNamespace(),
		Name:      owner.GetName(),
	}
	ownerRef.SetGroupVersionKind(owner.GetObjectKind().GroupVersionKind())

	oe := m.ORMRegistry.RetriveObjectEntryForOwnerAndORM(ownerRef, types.NamespacedName{
		Namespace: orm.GetNamespace(),
		Name:      orm.GetName(),
	})

	for resource, mappings := range *oe {
		resobj := &unstructured.Unstructured{}
		resobj.SetGroupVersionKind(resource.GroupVersionKind())

		resobj, err = kubernetes.Toolbox.GetResourceWithGVK(resource.GroupVersionKind(), types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name})
		if err != nil {
			for op := range mappings {
				for n, m := range orm.Status.OwnerMappingValues {
					if op == m.OwnerPath {
						orm.Status.OwnerMappingValues[n].Message = "Failed to locate owned resource: " + resource.String()
						orm.Status.OwnerMappingValues[n].Reason = string(v1alpha1.ORMStatusReasonOwnedResourceError)
					}
				}
			}
			continue
		}

		for op, sp := range mappings {
			mapitem := PrepareMappingForObject(resobj, sp)
			for n, m := range orm.Status.OwnerMappingValues {
				if op == m.OwnerPath {
					if mapitem == nil && orm.Status.OwnerMappingValues[n].Message == messagePlaceHolder {
						orm.Status.OwnerMappingValues[n].Message = "Failed to locate mapping path " + sp + " in owned resource"
						orm.Status.OwnerMappingValues[n].Reason = string(v1alpha1.ORMStatusReasonOwnedResourceError)
					} else if mapitem != nil && orm.Status.OwnerMappingValues[n].Message == messagePlaceHolder {
						orm.Status.OwnerMappingValues[n].Message = ""
						orm.Status.OwnerMappingValues[n].Reason = ""

					} else if mapitem != nil && orm.Status.OwnerMappingValues[n].Reason == string(v1alpha1.ORMStatusReasonOwnedResourceError) {
						orm.Status.OwnerMappingValues[n].Message = ""
						orm.Status.OwnerMappingValues[n].Reason = ""
					}
				}
			}
		}
	}

}

func (m *SimpleMapper) setORMStatus(owner *unstructured.Unstructured, orm *v1alpha1.OperatorResourceMapping) {
	existingMappings := orm.Status.OwnerMappingValues
	orm.Status.OwnerMappingValues = nil

	ownerRef := corev1.ObjectReference{
		Namespace: owner.GetNamespace(),
		Name:      owner.GetName(),
	}
	ownerRef.SetGroupVersionKind(owner.GetObjectKind().GroupVersionKind())

	oe := m.ORMRegistry.RetriveObjectEntryForOwnerAndORM(ownerRef, types.NamespacedName{
		Namespace: orm.GetNamespace(),
		Name:      orm.GetName(),
	})

	allmappings := make(map[string]bool)
	for _, mappings := range *oe {
		for k := range mappings {
			allmappings[k] = true
		}
	}

	// add old mappings first
	for _, mapping := range existingMappings {
		if _, ok := allmappings[mapping.OwnerPath]; ok {
			delete(allmappings, mapping.OwnerPath)
		} else {
			continue
		}

		mapitem := PrepareMappingForObject(owner, mapping.OwnerPath)
		if mapitem != nil {
			mapitem.Message = messagePlaceHolder
			orm.Status.OwnerMappingValues = append(orm.Status.OwnerMappingValues, *mapitem)
		} else {
			mapitem = &v1alpha1.OwnerMappingValue{
				OwnerPath: mapping.OwnerPath,
				Message:   "Failed to locate ownerPath in owner",
				Reason:    string(v1alpha1.ORMStatusReasonOwnerError),
			}
			orm.Status.OwnerMappingValues = append(orm.Status.OwnerMappingValues, *mapitem)
		}

	}

	if len(allmappings) != 0 {
		for ownerPath := range allmappings {
			mapitem := PrepareMappingForObject(owner, ownerPath)
			if mapitem != nil {
				mapitem.Message = messagePlaceHolder
				orm.Status.OwnerMappingValues = append(orm.Status.OwnerMappingValues, *mapitem)
			} else {
				mapitem = &v1alpha1.OwnerMappingValue{
					OwnerPath: ownerPath,
					Message:   "Failed to locate ownerPath in owner",
					Reason:    string(v1alpha1.ORMStatusReasonOwnerError),
				}
				orm.Status.OwnerMappingValues = append(orm.Status.OwnerMappingValues, *mapitem)
			}
		}
	}

	m.validateOwnedResources(owner, orm)
}

func (m *SimpleMapper) updateORMStatus(orm *v1alpha1.OperatorResourceMapping) {

	var err error
	err = kubernetes.Toolbox.OrmClient.Status().Update(context.TODO(), orm)
	if err != nil {
		msLog.Error(err, "retry status")
	}

}

func (m *SimpleMapper) Start(context.Context) error {
	return nil
}

func (m *SimpleMapper) SetupWithManager(mgr manager.Manager) error {
	return mgr.Add(m)
}

func GetSimpleMapper(config *rest.Config, scheme *runtime.Scheme) (Mapper, error) {
	var err error

	if mp == nil {
		mp = &SimpleMapper{}
		if mp.watchingGVK == nil {
			mp.watchingGVK = make(map[schema.GroupVersionKind]bool)
		}
	}

	return mp, err
}
