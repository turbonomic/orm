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

	"github.com/turbonomic/orm/kubernetes"
	"github.com/turbonomic/orm/registry"
)

type AdviceMapper struct {
	reg *registry.ResourceMappingRegistry

	watchingGVK map[schema.GroupVersionKind]bool
}

func NewAdviceMapper(reg *registry.ResourceMappingRegistry) (*AdviceMapper, error) {
	mp := &AdviceMapper{
		reg: reg,
	}

	return mp, nil
}

func (m *AdviceMapper) RegisterForAdvisor(gvk schema.GroupVersionKind, key types.NamespacedName) error {

	if _, ok := m.watchingGVK[gvk]; !ok {
		kubernetes.Toolbox.WatchResourceWithGVK(gvk, cache.ResourceEventHandlerFuncs{
			AddFunc: func(new interface{}) {
				obj := new.(*unstructured.Unstructured)
				m.mapForAdvisor(obj)
			},
			UpdateFunc: func(old, new interface{}) {
				obj := new.(*unstructured.Unstructured)
				m.mapForAdvisor(obj)
			}})
		m.watchingGVK[gvk] = true
	}

	return nil
}

func (m *AdviceMapper) mapForAdvisor(obj *unstructured.Unstructured) {

}
