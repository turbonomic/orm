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

	"github.com/turbonomic/orm/api/v1alpha1"
)

func BuildAllPatterns(orm *v1alpha1.OperatorResourceMapping) []v1alpha1.Pattern {
	allpatterns := orm.Spec.Mappings.Patterns
	if orm.Spec.Mappings.Lists != nil && len(orm.Spec.Mappings.Lists) > 0 {
		var prevpatterns []v1alpha1.Pattern
		for name, speclist := range orm.Spec.Mappings.Lists {
			prevpatterns = allpatterns
			allpatterns = []v1alpha1.Pattern{}
			var loc int
			for _, p := range prevpatterns {
				loc = strings.Index(p.OperandPath, "{{"+name+"}}")
				if loc == -1 {
					allpatterns = append(allpatterns, p)
				} else {
					list := speclist
					if len(list) == 1 && list[0] == "*" {
						tmpp := p.DeepCopy()
						tmpp.OperandPath = strings.ReplaceAll(p.OperandPath, "{{"+name+"}}", list[0])
						tmpp.Source.Path = strings.ReplaceAll(p.Source.Path, "{{"+name+"}}", list[0])
						//TODO: replace list with the names got from source, need to retreive all source objs
					}
					for _, c := range list {
						newp := p.DeepCopy()
						newp.OperandPath = strings.ReplaceAll(p.OperandPath, "{{"+name+"}}", c)
						newp.Source.Path = strings.ReplaceAll(p.Source.Path, "{{"+name+"}}", c)
						allpatterns = append(allpatterns, *newp)
					}
				}
			}
		}
	}
	return allpatterns
}
