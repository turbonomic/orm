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
	ctrl "sigs.k8s.io/controller-runtime"
)

// namespacedname of ORM as key, in case 1 source maps to more than 1 ORM

type Mappings map[string]string
type ObjectEntry map[corev1.ObjectReference]Mappings

// ORMEntry is defined for registry to search orm and all registered mappings
type ORMEntry map[types.NamespacedName]ObjectEntry

type ORMRegistry struct {
	// registry is defined to find orm and mappings by Owned Object
	registry map[corev1.ObjectReference]ORMEntry
	// ownerRegistry is defined to find orm and mappings by Owner Object
	ownerRegistry map[corev1.ObjectReference]ORMEntry
}

var (
	rLog = ctrl.Log.WithName("source regisry")
)

func registerMappingToRegistry(registry map[corev1.ObjectReference]ORMEntry, operandPath string, objectPath string, orm types.NamespacedName, resource corev1.ObjectReference, index corev1.ObjectReference) error {

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

	var ormEntry ORMEntry
	var exists bool
	if ormEntry, exists = registry[indexref]; !exists {
		ormEntry = make(map[types.NamespacedName]ObjectEntry)
	}

	var oe ObjectEntry
	var ok bool

	if oe, ok = ormEntry[orm]; !ok {
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
	m[operandPath] = objectPath
	oe[resref] = m
	ormEntry[orm] = oe
	registry[indexref] = ormEntry

	return nil
}

func (or *ORMRegistry) RegisterMapping(operandPath string, objectPath string, orm types.NamespacedName, operand corev1.ObjectReference, object corev1.ObjectReference) error {

	var err error

	if or.registry == nil {
		or.registry = make(map[corev1.ObjectReference]ORMEntry)
	}
	err = registerMappingToRegistry(or.registry, operandPath, objectPath, orm, operand, object)
	if err != nil {
		return err
	}

	if or.ownerRegistry == nil {
		or.ownerRegistry = make(map[corev1.ObjectReference]ORMEntry)
	}

	err = registerMappingToRegistry(or.ownerRegistry, operandPath, objectPath, orm, object, operand)

	return err
}

func cleanupORMInRegistry(registry map[corev1.ObjectReference]ORMEntry, orm types.NamespacedName) {
	if registry == nil {
		return
	}

	for _, ormEntry := range registry {
		if _, exists := ormEntry[orm]; exists {
			delete(ormEntry, orm)
		}
	}
}

func (or *ORMRegistry) CleanupRegistryForORM(orm types.NamespacedName) {
	cleanupORMInRegistry(or.registry, orm)

	cleanupORMInRegistry(or.ownerRegistry, orm)

	return

}

func retriveORMEntryForObjectFromRegistry(registry map[corev1.ObjectReference]ORMEntry, objref corev1.ObjectReference) ORMEntry {
	if registry == nil {
		return nil
	}

	return registry[objref]
}

func retriveObjectEntryForObjectAndORMFromRegistry(registry map[corev1.ObjectReference]ORMEntry, obj corev1.ObjectReference, orm types.NamespacedName) *ObjectEntry {
	orme := retriveORMEntryForObjectFromRegistry(registry, obj)
	if orme == nil {
		return nil
	}

	oe, ok := orme[orm]
	if !ok {
		return nil
	}

	return &oe
}

func (or *ORMRegistry) RetriveORMEntryForOwner(ownerref corev1.ObjectReference) ORMEntry {
	return retriveORMEntryForObjectFromRegistry(or.ownerRegistry, ownerref)
}

func (or *ORMRegistry) RetriveObjectEntryForOwnerAndORM(ownerref corev1.ObjectReference, orm types.NamespacedName) *ObjectEntry {
	return retriveObjectEntryForObjectAndORMFromRegistry(or.ownerRegistry, ownerref, orm)
}

func (or *ORMRegistry) RetriveORMEntryForResource(objref corev1.ObjectReference) ORMEntry {
	return retriveORMEntryForObjectFromRegistry(or.registry, objref)
}

func (or *ORMRegistry) RetriveObjectEntryForResourceAndORM(objref corev1.ObjectReference, orm types.NamespacedName) *ObjectEntry {
	return retriveObjectEntryForObjectAndORMFromRegistry(or.registry, objref, orm)
}
