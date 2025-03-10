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

package mappers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	devopsv1alpha1 "github.ibm.com/turbonomic/orm/api/v1alpha1"
	"github.ibm.com/turbonomic/orm/kubernetes"
	"github.ibm.com/turbonomic/orm/registry"
)

var ()

// OperatorResourceMappingReconciler reconciles a OperatorResourceMapping object
type OwnershipMapper struct {
	reg *registry.ResourceMappingRegistry

	watchingGVK map[schema.GroupVersionKind]bool
}

func (m *OwnershipMapper) RegisterOwnerFromORM(gvk schema.GroupVersionKind, key types.NamespacedName, orm *devopsv1alpha1.OperatorResourceMapping) error {
	obj, err := kubernetes.Toolbox.GetResourceWithGVK(gvk, key)
	if err != nil {
		return err
	}
	m.reg.SetORMStatusForOwner(obj, orm)

	if _, ok := m.watchingGVK[gvk]; !ok {
		kubernetes.Toolbox.WatchResourceWithGVK(gvk, cache.ResourceEventHandlerFuncs{
			AddFunc: func(new interface{}) {
				obj := new.(*unstructured.Unstructured)
				m.reg.SetORMStatusForOwner(obj, nil)
			},
			UpdateFunc: func(old, new interface{}) {
				obj := new.(*unstructured.Unstructured)
				m.reg.SetORMStatusForOwner(obj, nil)
			}})
		m.watchingGVK[gvk] = true
	}

	return nil
}

func NewOwnershipMapper(reg *registry.ResourceMappingRegistry) *OwnershipMapper {

	om := &OwnershipMapper{
		reg: reg,
	}
	if om.watchingGVK == nil {
		om.watchingGVK = make(map[schema.GroupVersionKind]bool)
	}

	return om
}
