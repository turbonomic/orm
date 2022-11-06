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
	"errors"

	"github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/registry"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	msLog = ctrl.Log.WithName("mapper simple")

	mp *SimpleMapper
)

// OperatorResourceMappingReconciler reconciles a OperatorResourceMapping object
type SimpleMapper struct {
	reg *registry.Registry
}

func (m *SimpleMapper) CreateUpdateSourceRegistryEntries(op *v1alpha1.OperatorResourceMapping) error {
	return nil
}

func GetMapper(r *registry.Registry) (*SimpleMapper, error) {
	if r == nil {
		return nil, errors.New("Null registry for enforcer")
	}

	if mp == nil {
		mp = &SimpleMapper{}
	}

	mp.reg = r

	return mp, nil
}
