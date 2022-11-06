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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Schema struct {
	dynamic.Interface
	config    *rest.Config
	gvkGVRMap map[schema.GroupVersionKind]*schema.GroupVersionResource
	ctx       context.Context
}

var (
	ormSchema         Schema
	resourcePredicate = discovery.SupportsAllVerbs{Verbs: []string{"create", "update", "delete", "list", "watch"}}

	sLog = ctrl.Log.WithName("schema")
)

func InitORMSchema(config *rest.Config) error {
	var err error

	ormSchema.config = config

	if config == nil {
		return errors.New("Null Config for discovery")
	}

	ormSchema.ctx = context.TODO()
	ormSchema.Interface, err = dynamic.NewForConfig(config)

	return err
}

func (s *Schema) getOperandbyGVK(gvk schema.GroupVersionKind, req types.NamespacedName) (*unstructured.Unstructured, error) {

	var err error
	gvr := s.findGVRfromGVK(gvk)
	if gvr == nil {
		return nil, errors.New("Operator " + gvk.String() + "is not installed")
	}

	obj := &unstructured.Unstructured{}
	obj, err = s.Resource(*gvr).Namespace(req.Namespace).Get(s.ctx, req.Name, metav1.GetOptions{})

	return obj, err
}

func (s *Schema) updateOperand(gvk schema.GroupVersionKind, obj *unstructured.Unstructured) error {
	var err error

	gvr := s.findGVRfromGVK(gvk)
	if gvr == nil {
		return errors.New("Operator " + gvk.String() + "is not installed")
	}

	_, err = s.Resource(*gvr).Namespace(obj.GetNamespace()).Update(s.ctx, obj, metav1.UpdateOptions{})

	return err

}

func (s *Schema) findGVRfromGVK(gvk schema.GroupVersionKind) *schema.GroupVersionResource {
	if s.config == nil {
		return nil
	}

	if s.gvkGVRMap == nil || s.gvkGVRMap[gvk] == nil {
		s.discoverSchemaMappings()
	}

	return s.gvkGVRMap[gvk]
}

func (s *Schema) discoverSchemaMappings() {
	resources, err := discovery.NewDiscoveryClientForConfigOrDie(s.config).ServerPreferredResources()
	// do not return if there is error
	// some api server aggregation may cause this problem, but can still get return some resources.
	if err != nil {
	}

	filteredResources := discovery.FilteredBy(resourcePredicate, resources)

	if s.gvkGVRMap == nil {
		s.gvkGVRMap = make(map[schema.GroupVersionKind]*schema.GroupVersionResource)
	}

	for _, rl := range filteredResources {
		s.buildGVKGVRMap(rl)
	}
}

func (s *Schema) buildGVKGVRMap(rl *metav1.APIResourceList) {
	for _, res := range rl.APIResources {
		gv, err := schema.ParseGroupVersion(rl.GroupVersion)
		if err != nil {
			continue
		}

		gvk := schema.GroupVersionKind{
			Kind:    res.Kind,
			Group:   gv.Group,
			Version: gv.Version,
		}
		gvr := &schema.GroupVersionResource{
			Group:    gv.Group,
			Version:  gv.Version,
			Resource: res.Name,
		}

		s.gvkGVRMap[gvk] = gvr
	}
}
