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

package kubernetes

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
)

type resourceEntry struct {
	schema.GroupVersionResource
	Namespaced bool
}

type Schema struct {
	gvkGVRMap map[schema.GroupVersionKind]*resourceEntry
}

var (
	resourcePredicate = discovery.SupportsAllVerbs{Verbs: []string{"create", "update", "delete", "list", "watch"}}

	sLog = ctrl.Log.WithName("schema")
)

func (s *Schema) FindGVKForResource(resource string) (*schema.GroupVersionKind, bool) {
	var gvk *schema.GroupVersionKind
	var found bool

	if s.gvkGVRMap == nil {
		s.discoverSchemaMappings()
	}

	loc := strings.Index(resource, ".")
	if loc == -1 {
		return s.bestEffortForGvkForMissingResource(resource), false
	}

	res := resource[:loc]
	group := resource[loc+1:]

	found = false
	for k, gvr := range s.gvkGVRMap {
		if gvr.Resource == res && gvr.Group == group {
			found = true
			gvk = &k
			break
		}
	}

	if !found {
		gvk = s.bestEffortForGvkForMissingResource(resource)
	}

	return gvk, found
}

func (s *Schema) bestEffortForGvkForMissingResource(resource string) *schema.GroupVersionKind {
	var gvk schema.GroupVersionKind

	loc := strings.Index(resource, ".")
	if loc == -1 {
		gvk.Kind = resource
	} else {
		gvk.Kind = resource[:loc]
		gvk.Group = resource[loc+1:]
	}
	return &gvk
}

func (s *Schema) FindGVRfromGVK(gvk schema.GroupVersionKind) (*schema.GroupVersionResource, bool) {
	if s.gvkGVRMap == nil || s.gvkGVRMap[gvk] == nil {
		s.discoverSchemaMappings()
	}

	if s.gvkGVRMap[gvk] == nil {
		return nil, true
	}

	return &s.gvkGVRMap[gvk].GroupVersionResource, s.gvkGVRMap[gvk].Namespaced
}

func (s *Schema) discoverSchemaMappings() {
	resources, err := discovery.NewDiscoveryClientForConfigOrDie(Toolbox.cfg).ServerPreferredResources()
	// do not return if there is error
	// some api server aggregation may cause this problem, but can still get return some resources.
	if err != nil {
		sLog.Info("discovering schema mapping, ignoring error ", err)
	}

	filteredResources := discovery.FilteredBy(resourcePredicate, resources)

	if s.gvkGVRMap == nil {
		s.gvkGVRMap = make(map[schema.GroupVersionKind]*resourceEntry)
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
		entry := &resourceEntry{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: res.Name,
			},
			Namespaced: res.Namespaced,
		}

		s.gvkGVRMap[gvk] = entry
	}
}
