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

package mapper

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/kubernetes"
	"github.com/turbonomic/orm/registry"
	"github.com/turbonomic/orm/util"
)

func RegisterORM(reg *registry.ORMRegistry, orm *v1alpha1.OperatorResourceMapping) error {
	var err error

	if orm == nil {
		return nil
	}

	reg.CleanupRegistryForORM(types.NamespacedName{
		Namespace: orm.Namespace,
		Name:      orm.Name,
	})

	if orm.Spec.Mappings.Patterns == nil || len(orm.Spec.Mappings.Patterns) == 0 {
		return nil
	}

	allpatterns := buildAllPatterns(orm)

	var srcObj *unstructured.Unstructured
	for _, p := range allpatterns {
		k := types.NamespacedName{Namespace: p.OwnedResourcePath.Namespace, Name: p.OwnedResourcePath.Name}
		if k.Namespace == "" {
			k.Namespace = orm.Namespace
		}

		// TODO: avoid to retrieve same source repeatedly
		var srcObjs []unstructured.Unstructured
		if k.Name != "" {
			srcObj = &unstructured.Unstructured{}
			srcObj.SetGroupVersionKind(p.OwnedResourcePath.GroupVersionKind())
			srcObj.SetNamespace(k.Namespace)
			srcObj.SetName(k.Name)
			srcObjs = append(srcObjs, *srcObj)
		} else {
			srcObjs, err = kubernetes.Toolbox.GetResourceListWithGVKWithSelector(p.OwnedResourcePath.GroupVersionKind(), k, &p.OwnedResourcePath.LabelSelector)
			if err != nil {
				msLog.Error(err, "listing resource", "source", p.OwnedResourcePath)
			}
		}

		for _, srcObj := range srcObjs {
			objref := p.OwnedResourcePath.ObjectReference.DeepCopy()
			objref.Namespace = srcObj.GetNamespace()
			objref.Name = srcObj.GetName()
			err = reg.RegsiterMapping(p.OwnerPath, p.OwnedResourcePath.Path,
				types.NamespacedName{Name: orm.Name, Namespace: orm.Namespace},
				orm.Spec.Owner.ObjectReference,
				*objref)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func buildAllPatterns(orm *v1alpha1.OperatorResourceMapping) []v1alpha1.Pattern {
	allpatterns := orm.Spec.Mappings.Patterns
	if orm.Spec.Mappings.Parameters != nil && len(orm.Spec.Mappings.Parameters) > 0 {
		var prevpatterns []v1alpha1.Pattern
		for name, speclist := range orm.Spec.Mappings.Parameters {
			prevpatterns = allpatterns
			allpatterns = []v1alpha1.Pattern{}
			var loc int
			for _, p := range prevpatterns {
				loc = strings.Index(p.OwnerPath, "{{"+name+"}}")
				if loc == -1 {
					allpatterns = append(allpatterns, p)
				} else {
					list := speclist
					if len(list) == 1 && list[0] == "*" {
						tmpp := p.DeepCopy()
						tmpp.OwnerPath = strings.ReplaceAll(p.OwnerPath, "{{"+name+"}}", list[0])
						tmpp.OwnedResourcePath.Path = strings.ReplaceAll(p.OwnedResourcePath.Path, "{{"+name+"}}", list[0])
						//TODO: replace list with the names got from source, need to retreive all source objs
					}
					for _, c := range list {
						newp := p.DeepCopy()
						newp.OwnerPath = strings.ReplaceAll(p.OwnerPath, "{{"+name+"}}", c)
						newp.OwnedResourcePath.Path = strings.ReplaceAll(p.OwnedResourcePath.Path, "{{"+name+"}}", c)
						allpatterns = append(allpatterns, *newp)
					}
				}
			}
		}
	}
	return allpatterns
}

func PrepareMappingForObject(obj *unstructured.Unstructured, objPath string) *v1alpha1.Mapping {
	mapitem := v1alpha1.Mapping{}
	mapitem.OwnerPath = objPath

	fields := strings.Split(objPath, ".")
	lastField := fields[len(fields)-1]
	valueInObj, found, err := util.NestedField(obj, lastField, objPath)

	valueMap := make(map[string]interface{})
	valueMap[lastField] = valueInObj

	if err != nil {
		msLog.Error(err, "parsing src", "fields", fields, "actual", obj.Object["metadata"])
		return nil
	}
	if !found {
		return nil
	}

	valueObj := &unstructured.Unstructured{
		Object: valueMap,
	}
	mapitem.Value = &runtime.RawExtension{
		Object: valueObj,
	}

	return &mapitem
}
