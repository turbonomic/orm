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

package enforcer

import (
	"errors"
	"strings"

	"github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/registry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Enforcer struct {
	reg *registry.Registry
}

var (
	eLog = ctrl.Log.WithName("enforcer")

	enforcer *Enforcer
)

func (e *Enforcer) CreateUpdateOperandRegistryEntry(orm *v1alpha1.OperatorResourceMapping) error {

	req := types.NamespacedName{}
	req.Namespace = orm.Spec.Operand.Namespace
	if req.Namespace == "" {
		req.Namespace = orm.Namespace
	}

	req.Name = orm.Spec.Operand.Name
	if req.Name == "" {
		req.Name = orm.Name
	}

	obj, err := e.reg.GetResourceWithGVK(orm.Spec.Operand.GroupVersionKind(), req)
	if err != nil {
		return err
	}

	if orm.Spec.EnforcementMode != v1alpha1.EnforcementModeNone {
		err = e.doMappings(orm, obj)
		if err != nil {
			return err
		}
	}

	err = e.reg.UpdateResourceWithGVK(orm.Spec.Operand.GroupVersionKind(), obj)

	return err
}

func (e *Enforcer) doMappings(orm *v1alpha1.OperatorResourceMapping, obj *unstructured.Unstructured) error {
	var err error

	if orm.Status.Mappings == nil {
		return nil
	}

	for n, m := range orm.Status.Mappings {
		fields := strings.Split(m.OperandPath, ".")
		value, err := runtime.DefaultUnstructuredConverter.ToUnstructured(m.Value)
		if err != nil {
			e.updateMappingStatus(orm, n, err)
			return err
		}

		err = unstructured.SetNestedField(obj.Object, value, fields...)
		if err != nil {
			e.updateMappingStatus(orm, n, err)
			return err
		}

		e.updateMappingStatus(orm, n, err)
	}

	return err
}

func (e *Enforcer) updateMappingStatus(orm *v1alpha1.OperatorResourceMapping, n int, err error) {
	if err == nil {
		orm.Status.Mappings[n].Mapped = corev1.ConditionTrue
		orm.Status.Mappings[n].Reason = ""
		orm.Status.Mappings[n].Message = ""
	} else {
		orm.Status.Mappings[n].Mapped = corev1.ConditionFalse
		orm.Status.Mappings[n].Reason = string(v1alpha1.ORMStatusReasonOperandError)
		orm.Status.Mappings[n].Message = err.Error()
	}
}

func GetEnforcer(r *registry.Registry) (*Enforcer, error) {
	if r == nil {
		return nil, errors.New("Null registry for enforcer")
	}

	if enforcer == nil {
		enforcer = &Enforcer{}
	}

	enforcer.reg = r

	return enforcer, nil
}
