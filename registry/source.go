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
	"errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// operandPath as key, sourcePath as value
type PatternMap map[string]string

// namespacedname of ORM as key, in case 1 source maps to more than 1 ORM
type SourceORMEntry map[types.NamespacedName]PatternMap

type sourceObjectRef struct {
	schema.GroupVersionResource
	types.NamespacedName
}

// object groupversionkind + namespacedname as key
type SourceRegistry struct {
	// groupversionkind of source resource as key
	registry map[sourceObjectRef]SourceORMEntry
}

var (
	srLog = ctrl.Log.WithName("source regisry")
)

func (sr *SourceRegistry) RegisterSource(op string, sobj corev1.ObjectReference, sp string, orm types.NamespacedName) (bool, error) {

	if sr.registry == nil {
		sr.registry = make(map[sourceObjectRef]SourceORMEntry)
	}

	gvr := r.findGVRfromGVK(sobj.GroupVersionKind())
	if gvr == nil {
		return false, errors.New("Source resource " + sobj.GroupVersionKind().String() + "is not installed")
	}

	var exists bool
	sref := sourceObjectRef{
		GroupVersionResource: *gvr,
		NamespacedName: types.NamespacedName{
			Namespace: sobj.Namespace,
			Name:      sobj.Name,
		},
	}

	if sref.Namespace == "" {
		sref.Namespace = orm.Namespace
	}

	var ormEntry SourceORMEntry
	if ormEntry, exists = sr.registry[sref]; !exists {
		ormEntry = make(map[types.NamespacedName]PatternMap)
	}

	pm, ok := ormEntry[orm]
	if !ok {
		pm = make(map[string]string)
	}

	pm[op] = sp
	ormEntry[orm] = pm
	sr.registry[sref] = ormEntry

	return exists, nil
}

func (sr *SourceRegistry) RetriveORMEntryForResource(gvk schema.GroupVersionKind, key types.NamespacedName) SourceORMEntry {
	if sr.registry == nil {
		return nil
	}

	gvr := r.findGVRfromGVK(gvk)
	if gvr == nil {
		return nil
	}

	sref := sourceObjectRef{
		GroupVersionResource: *gvr,
		NamespacedName:       key,
	}

	return sr.registry[sref]
}
