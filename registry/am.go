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
	devopsv1alpha1 "github.ibm.com/turbonomic/orm/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (or *ResourceMappingRegistry) CleanupRegistryForAM(am types.NamespacedName) {
	if or.advisorRegistry != nil {
		cleanupORMInRegistry(or.advisorRegistry, am)
	}
}

func (or *ResourceMappingRegistry) RegisterAM(am *devopsv1alpha1.AdviceMapping) error {
	var err error

	if am == nil {
		return nil
	}

	if am.Spec.Mappings == nil || len(am.Spec.Mappings) == 0 {
		return nil
	}

	amkey := types.NamespacedName{
		Namespace: am.Namespace,
		Name:      am.Name,
	}

	or.CleanupRegistryForAM(amkey)

	for _, m := range am.Spec.Mappings {
		or.registerAdviceMappingItem(m.AdvisorResourcePath.Path, m.TargetResourcePath.Path,
			amkey,
			m.TargetResourcePath.ObjectReference,
			m.AdvisorResourcePath.ObjectReference,
		)
	}

	return err
}

func (or *ResourceMappingRegistry) RetrieveAMEntryForAdvisor(advisor corev1.ObjectReference) ResourceMappingEntryType {
	return retrieveResourceMappingEntryForObjectFromRegistry(or.advisorRegistry, advisor)
}
