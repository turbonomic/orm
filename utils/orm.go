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

package util

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	devopsv1alpha1 "github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/kubernetes"
	"github.com/turbonomic/orm/registry"
)

var (
	uoLog = ctrl.Log.WithName("util unstructured")
)

func SeekTopOwnersResourcePathsForTarget(reg *registry.ResourceMappingRegistry, target devopsv1alpha1.ResourcePath) []devopsv1alpha1.ResourcePath {
	owners := []devopsv1alpha1.ResourcePath{target}

	more := true
	for more {
		more = false
		old := owners
		owners = []devopsv1alpha1.ResourcePath{}
		for _, rp := range old {
			orme := reg.RetrieveORMEntryForOwned(rp.ObjectReference)
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

func RegisterAM(reg *registry.ResourceMappingRegistry, am *devopsv1alpha1.AdviceMapping) error {
	var err error

	if am == nil || reg == nil {
		return nil
	}

	if am.Spec.Mappings == nil || len(am.Spec.Mappings) == 0 {
		return nil
	}

	for _, m := range am.Spec.Mappings {
		reg.RegisterAdviceMapping(m.TargetResourcePath.Path, m.AdvisorResourcePath.Path,
			types.NamespacedName{
				Namespace: am.Namespace,
				Name:      am.Name,
			},
			m.TargetResourcePath.ObjectReference,
			m.AdvisorResourcePath.ObjectReference,
		)
	}

	return err
}

func RegisterORM(reg *registry.ResourceMappingRegistry, orm *devopsv1alpha1.OperatorResourceMapping) error {
	var err error

	if orm == nil || reg == nil {
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
				uoLog.Error(err, "listing resource", "source", p.OwnedResourcePath)
			}
			for _, obj := range srcObjs {
				srckeys = append(srckeys, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()})
			}
		}
		srcmap[p.OwnedResourcePath.Path] = srckeys
	}

	reg.CleanupRegistryForORM(types.NamespacedName{
		Namespace: orm.Namespace,
		Name:      orm.Name,
	})

	for _, p := range orm.Spec.Mappings.Patterns {
		for _, k := range srcmap[p.OwnedResourcePath.Path] {
			p.OwnedResourcePath.Namespace = k.Namespace
			p.OwnedResourcePath.Name = k.Name

			patterns := populatePatterns(orm.Spec.Mappings.Parameters, p)

			for _, pattern := range patterns {
				err = reg.RegisterOwnershipMapping(pattern.OwnerPath, pattern.OwnedResourcePath.Path,
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
