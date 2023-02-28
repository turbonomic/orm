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

package controllers

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/kubernetes"
	"github.com/turbonomic/orm/registry"
)

const predefinedOwnedResourceName = ".owned.name"
const predefinedParameterPlaceHolder = ".."

func RegisterORM(reg *registry.ORMRegistry, orm *v1alpha1.OperatorResourceMapping) error {
	var err error

	if orm == nil {
		return nil
	}

	if orm.Spec.Mappings.Patterns == nil || len(orm.Spec.Mappings.Patterns) == 0 {
		return nil
	}

	var srcObj *unstructured.Unstructured
	for _, p := range orm.Spec.Mappings.Patterns {
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
				ocLog.Error(err, "listing resource", "source", p.OwnedResourcePath)
			}
		}

		reg.CleanupRegistryForORM(types.NamespacedName{
			Namespace: orm.Namespace,
			Name:      orm.Name,
		})

		for _, srcObj := range srcObjs {
			p.OwnedResourcePath.Namespace = srcObj.GetNamespace()
			p.OwnedResourcePath.Name = srcObj.GetName()

			patterns := populatePatterns(orm.Spec.Mappings.Parameters, p)

			for _, pattern := range patterns {
				err = reg.RegsiterMapping(pattern.OwnerPath, pattern.OwnedResourcePath.Path,
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

func populatePatterns(parameters map[string][]string, pattern v1alpha1.Pattern) []v1alpha1.Pattern {
	var allpatterns []v1alpha1.Pattern

	pattern.OwnerPath = strings.ReplaceAll(pattern.OwnerPath, "{{"+predefinedOwnedResourceName+"}}", pattern.OwnedResourcePath.Name)
	pattern.OwnedResourcePath.Path = strings.ReplaceAll(pattern.OwnedResourcePath.Path, "{{"+predefinedOwnedResourceName+"}}", pattern.OwnedResourcePath.Name)

	allpatterns = append(allpatterns, pattern)

	if parameters == nil {
		parameters = make(map[string][]string)
	}
	parameters[predefinedParameterPlaceHolder] = []string{predefinedParameterPlaceHolder}
	var prevpatterns []v1alpha1.Pattern
	for name, values := range parameters {
		prevpatterns = allpatterns
		allpatterns = []v1alpha1.Pattern{}

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
