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
type ObjectEntry struct {
	corev1.ObjectReference
	// ormPath as key, objectPath as value
	Mappings map[string]string
}

type ORMEntry map[types.NamespacedName]ObjectEntry

type ORMRegistry struct {
	// groupversionkind of source resource as key
	registry map[corev1.ObjectReference]ORMEntry
}

var (
	rLog = ctrl.Log.WithName("source regisry")
)

func (or *ORMRegistry) RegsiterMapping(operandPath string, objectPath string, orm types.NamespacedName, operand corev1.ObjectReference, object corev1.ObjectReference) error {

	if or.registry == nil {
		or.registry = make(map[corev1.ObjectReference]ORMEntry)
	}

	oref := corev1.ObjectReference{
		Namespace: object.Namespace,
		Name:      object.Name,
	}
	oref.SetGroupVersionKind(object.GroupVersionKind())

	if oref.Namespace == "" {
		oref.Namespace = orm.Namespace
	}

	var ormEntry ORMEntry
	var exists bool
	if ormEntry, exists = or.registry[oref]; !exists {
		ormEntry = make(map[types.NamespacedName]ObjectEntry)
	}

	var oe ObjectEntry
	var ok bool
	oe, ok = ormEntry[orm]
	if !ok {
		oe.Mappings = make(map[string]string)
	}

	oe.SetGroupVersionKind(operand.GroupVersionKind())
	oe.Namespace = operand.Namespace
	oe.Name = operand.Name

	oe.Mappings[operandPath] = objectPath
	ormEntry[orm] = oe
	or.registry[oref] = ormEntry

	return nil
}

func (or *ORMRegistry) CleanupRegistryForORM(orm types.NamespacedName) {
	if or.registry == nil {
		return
	}

	for _, ormEntry := range or.registry {
		if _, exists := ormEntry[orm]; exists {
			delete(ormEntry, orm)
		}
	}

	return

}

func (or *ORMRegistry) RetriveObjectEntryForResourceAndORM(obj corev1.ObjectReference, orm types.NamespacedName) *ObjectEntry {
	orme := or.RetriveORMEntryForResource(obj)
	if orme == nil {
		return nil
	}

	oe, ok := orme[orm]
	if !ok {
		return nil
	}

	return &oe
}

func (or *ORMRegistry) RetriveORMEntryForResource(objref corev1.ObjectReference) ORMEntry {
	if or.registry == nil {
		return nil
	}

	return or.registry[objref]
}
