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

type SourceRegistryEntry struct {
	OperandPath string
	SourcePath  string
	ORM         types.NamespacedName
}

type sourceGVRRegistry map[types.NamespacedName]SourceRegistryEntry
type SourceRegistry struct {
	registry map[schema.GroupVersionResource]sourceGVRRegistry
}

var (
	srLog = ctrl.Log.WithName("source regisry")
)

func (sr *SourceRegistry) RegisterSource(op string, sobj corev1.ObjectReference, sp string, orm types.NamespacedName) (bool, error) {

	if sr.registry == nil {
		sr.registry = make(map[schema.GroupVersionResource]sourceGVRRegistry)
	}

	gvr := r.findGVRfromGVK(sobj.GroupVersionKind())
	if gvr == nil {
		return false, errors.New("Source resource " + sobj.GroupVersionKind().String() + "is not installed")
	}

	var exists bool
	var gvrReg sourceGVRRegistry

	if gvrReg, exists = sr.registry[*gvr]; !exists {
		gvrReg = make(map[types.NamespacedName]SourceRegistryEntry)
	}

	skey := types.NamespacedName{
		Namespace: sobj.Namespace,
		Name:      sobj.Name,
	}
	if skey.Namespace == "" {
		skey.Namespace = orm.Namespace
	}

	entry, ok := gvrReg[skey]
	if !ok {
		entry = SourceRegistryEntry{}
	}

	entry.OperandPath = op
	entry.SourcePath = sp
	entry.ORM = orm

	gvrReg[skey] = entry

	sr.registry[*gvr] = gvrReg

	return exists, nil
}

func (sr *SourceRegistry) RetriveSource(gvk schema.GroupVersionKind, key types.NamespacedName) *SourceRegistryEntry {
	var exists bool

	if sr.registry == nil {
		return nil
	}

	gvr := r.findGVRfromGVK(gvk)
	if gvr == nil {
		return nil
	}

	var gvrReg sourceGVRRegistry
	if gvrReg, exists = sr.registry[*gvr]; !exists {
		return nil
	}

	if entry, ok := gvrReg[key]; ok {
		return &entry
	}
	return nil
}
