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
	"strings"

	"github.com/turbonomic/orm/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	devopsv1alpha1 "github.com/turbonomic/orm/api/v1alpha1"
)

func (or *ResourceMappingRegistry) SeekTopOwnersResourcePathsForOwnedResourcePath(owned devopsv1alpha1.ResourcePath) []devopsv1alpha1.ResourcePath {
	owners := []devopsv1alpha1.ResourcePath{owned}

	more := true
	for more {
		more = false
		old := owners
		owners = []devopsv1alpha1.ResourcePath{}
		for _, rp := range old {
			orme := or.RetrieveORMEntryForOwned(rp.ObjectReference)
			for _, oe := range orme {
				for o, m := range oe {
					if m[rp.Path] == "" {
						continue
					}
					more = true
					owners = append(owners, devopsv1alpha1.ResourcePath{
						ObjectReference: o,
						Path:            m[rp.Path],
					})
				}
			}
			if !more {
				owners = append(owners, rp)
			}
		}
	}

	return owners

}

const predefinedOwnedResourceName = ".owned.name"
const predefinedParameterPlaceHolder = ".."

func (or *ResourceMappingRegistry) RegisterORM(orm *devopsv1alpha1.OperatorResourceMapping) error {
	var err error

	if orm == nil {
		return nil
	}

	if orm.Spec.Mappings.Patterns == nil || len(orm.Spec.Mappings.Patterns) == 0 {
		return nil
	}

	srcmap := make(map[string][]types.NamespacedName)
	for _, p := range orm.Spec.Mappings.Patterns {

		var srckeys []types.NamespacedName

		k := types.NamespacedName{Namespace: p.OwnedResourcePath.Namespace, Name: p.OwnedResourcePath.Name}
		if k.Namespace == "" {
			k.Namespace = orm.Namespace
		}

		// TODO: avoid to retrieve same source repeatedly
		if k.Name != "" {
			srckeys = append(srckeys, k)
		} else {
			var srcObjs []unstructured.Unstructured
			srcObjs, err = kubernetes.Toolbox.GetResourceListWithGVKWithSelector(p.OwnedResourcePath.GroupVersionKind(), k, &p.OwnedResourcePath.LabelSelector)
			if err != nil {
				rLog.Error(err, "listing resource", "source", p.OwnedResourcePath)
			}
			for _, obj := range srcObjs {
				srckeys = append(srckeys, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()})
			}
		}
		srcmap[p.OwnedResourcePath.Path] = srckeys
	}

	or.CleanupRegistryForORM(types.NamespacedName{
		Namespace: orm.Namespace,
		Name:      orm.Name,
	})

	for _, p := range orm.Spec.Mappings.Patterns {
		for _, k := range srcmap[p.OwnedResourcePath.Path] {
			p.OwnedResourcePath.Namespace = k.Namespace
			p.OwnedResourcePath.Name = k.Name

			patterns := populatePatterns(orm.Spec.Mappings.Parameters, p)

			for _, pattern := range patterns {
				err = or.registerOwnershipMapping(pattern.OwnerPath, pattern.OwnedResourcePath.Path,
					types.NamespacedName{Name: orm.Name, Namespace: orm.Namespace},
					orm.Spec.Owner.ObjectReference,
					p.OwnedResourcePath.ObjectReference)
				if err != nil {
					return err
				}
			}
		}

	}

	return nil
}

func (or *ResourceMappingRegistry) CleanupRegistryForORM(orm types.NamespacedName) {

	if or.ownerRegistry != nil {
		cleanupORMInRegistry(or.ownerRegistry, orm)
	}
	if or.ownedRegistry != nil {
		cleanupORMInRegistry(or.ownedRegistry, orm)
	}
}

func (or *ResourceMappingRegistry) RetrieveORMEntryForOwner(owner corev1.ObjectReference) ResourceMappingEntry {
	return retrieveResourceMappingEntryForObjectFromRegistry(or.ownerRegistry, owner)
}

func (or *ResourceMappingRegistry) RetrieveORMEntryForOwned(owned corev1.ObjectReference) ResourceMappingEntry {
	return retrieveResourceMappingEntryForObjectFromRegistry(or.ownedRegistry, owned)
}

func (or *ResourceMappingRegistry) RetrieveObjectEntryForOwnerAndORM(owner corev1.ObjectReference, orm types.NamespacedName) *ObjectEntry {
	return retrieveObjectEntryForObjectAndORMFromRegistry(or.ownerRegistry, owner, orm)
}

func populatePatterns(parameters map[string][]string, pattern devopsv1alpha1.Pattern) []devopsv1alpha1.Pattern {
	var allpatterns []devopsv1alpha1.Pattern

	pattern.OwnerPath = strings.ReplaceAll(pattern.OwnerPath, "{{"+predefinedOwnedResourceName+"}}", pattern.OwnedResourcePath.Name)
	pattern.OwnedResourcePath.Path = strings.ReplaceAll(pattern.OwnedResourcePath.Path, "{{"+predefinedOwnedResourceName+"}}", pattern.OwnedResourcePath.Name)

	allpatterns = append(allpatterns, pattern)

	if parameters == nil {
		parameters = make(map[string][]string)
	}
	parameters[predefinedParameterPlaceHolder] = []string{predefinedParameterPlaceHolder}
	var prevpatterns []devopsv1alpha1.Pattern
	for name, values := range parameters {
		prevpatterns = allpatterns
		allpatterns = []devopsv1alpha1.Pattern{}

		for _, p := range prevpatterns {
			for _, v := range values {
				newp := p.DeepCopy()
				newp.OwnerPath = strings.ReplaceAll(p.OwnerPath, "{{"+name+"}}", v)
				newp.OwnedResourcePath.Path = strings.ReplaceAll(p.OwnedResourcePath.Path, "{{"+name+"}}", v)
				allpatterns = append(allpatterns, *newp)
			}
		}
	}
	return allpatterns
}
