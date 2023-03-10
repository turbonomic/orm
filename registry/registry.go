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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// namespacedname of ORM as key, in case 1 source maps to more than 1 ORM

type Mappings map[string]string
type ObjectEntry map[corev1.ObjectReference]Mappings

// ResourceMappingEntry is defined for registry to search orm and all registered mappings
type ResourceMappingEntry map[types.NamespacedName]ObjectEntry

type ResourceMappingRegistry struct {
	// ownerRegistry is defined to find orm and mappings by Owner Object
	ownerRegistry map[corev1.ObjectReference]ResourceMappingEntry
	// advisorRegistry is defined to find am and mappings by advisor
	advisorRegistry map[corev1.ObjectReference]ResourceMappingEntry
}

func registerMappingToRegistry(registry map[corev1.ObjectReference]ResourceMappingEntry, ownerPath string, objectPath string, orm types.NamespacedName, resource corev1.ObjectReference, index corev1.ObjectReference) error {

	if resource.Namespace == "" {
		resource.Namespace = orm.Namespace
	}

	if index.Namespace == "" {
		index.Namespace = orm.Namespace
	}

	indexref := corev1.ObjectReference{
		Namespace: index.Namespace,
		Name:      index.Name,
	}
	indexref.SetGroupVersionKind(index.GroupVersionKind())

	var ResourceMappingEntry ResourceMappingEntry
	var exists bool
	if ResourceMappingEntry, exists = registry[indexref]; !exists {
		ResourceMappingEntry = make(map[types.NamespacedName]ObjectEntry)
	}

	var oe ObjectEntry
	var ok bool

	if oe, ok = ResourceMappingEntry[orm]; !ok {
		oe = make(map[corev1.ObjectReference]Mappings)
	}

	resref := corev1.ObjectReference{
		Namespace: resource.Namespace,
		Name:      resource.Name,
	}
	resref.SetGroupVersionKind(resource.GroupVersionKind())

	var m Mappings
	if m, ok = oe[resref]; !ok {
		m = make(map[string]string)
	}
	m[ownerPath] = objectPath
	oe[resref] = m
	ResourceMappingEntry[orm] = oe
	registry[indexref] = ResourceMappingEntry

	return nil
}

func (or *ResourceMappingRegistry) RegisterAdviceMapping(targetPath string, advicePath string, am types.NamespacedName, target corev1.ObjectReference, advisor corev1.ObjectReference) error {
	var err error

	if or.advisorRegistry == nil {
		or.advisorRegistry = make(map[corev1.ObjectReference]ResourceMappingEntry)
	}

	err = registerMappingToRegistry(or.advisorRegistry, targetPath, advicePath, am, target, advisor)

	return err
}

func (or *ResourceMappingRegistry) RegisterOwnershipMapping(ownerPath string, objectPath string, orm types.NamespacedName, owner corev1.ObjectReference, object corev1.ObjectReference) error {

	var err error

	if or.ownerRegistry == nil {
		or.ownerRegistry = make(map[corev1.ObjectReference]ResourceMappingEntry)
	}

	err = registerMappingToRegistry(or.ownerRegistry, ownerPath, objectPath, orm, object, owner)

	return err
}

func cleanupORMInRegistry(registry map[corev1.ObjectReference]ResourceMappingEntry, orm types.NamespacedName) {
	if registry == nil {
		return
	}

	for _, ResourceMappingEntry := range registry {
		delete(ResourceMappingEntry, orm)
	}
}

func (or *ResourceMappingRegistry) CleanupRegistryForORM(orm types.NamespacedName) {

	cleanupORMInRegistry(or.ownerRegistry, orm)
}

func retrieveResourceMappingEntryForObjectFromRegistry(registry map[corev1.ObjectReference]ResourceMappingEntry, objref corev1.ObjectReference) ResourceMappingEntry {
	if registry == nil {
		return nil
	}

	return registry[objref]
}

func retrieveObjectEntryForObjectAndORMFromRegistry(registry map[corev1.ObjectReference]ResourceMappingEntry, obj corev1.ObjectReference, orm types.NamespacedName) *ObjectEntry {
	orme := retrieveResourceMappingEntryForObjectFromRegistry(registry, obj)
	if orme == nil {
		return nil
	}

	oe, ok := orme[orm]
	if !ok {
		return nil
	}

	return &oe
}

func (or *ResourceMappingRegistry) RetrieveORMEntryForOwner(owner corev1.ObjectReference) ResourceMappingEntry {
	return retrieveResourceMappingEntryForObjectFromRegistry(or.ownerRegistry, owner)
}

func (or *ResourceMappingRegistry) RetrieveObjectEntryForOwnerAndORM(owner corev1.ObjectReference, orm types.NamespacedName) *ObjectEntry {
	return retrieveObjectEntryForObjectAndORMFromRegistry(or.ownerRegistry, owner, orm)
}
