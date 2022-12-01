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
	"context"

	"github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/kubernetes"
	"github.com/turbonomic/orm/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type SimpleEnforcer struct {
	reg *kubernetes.Toolbox
}

var (
	eLog = ctrl.Log.WithName("enforcer")

	enforcer *SimpleEnforcer
)

func (e *SimpleEnforcer) EnforceORM(orm *v1alpha1.OperatorResourceMapping) error {
	if orm == nil {
		return nil
	}

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
		err = e.enforceOnce(orm, obj)
		if err != nil {
			eLog.Error(err, "enforce out")
			return err
		}
	}

	err = e.reg.UpdateResourceWithGVK(orm.Spec.Operand.GroupVersionKind(), obj)

	return err
}

func (e *SimpleEnforcer) enforceOnce(orm *v1alpha1.OperatorResourceMapping, obj *unstructured.Unstructured) error {
	var err error

	if orm.Status.MappedPatterns == nil {
		return nil
	}

	for n, m := range orm.Status.MappedPatterns {
		value, err := runtime.DefaultUnstructuredConverter.ToUnstructured(m.Value)
		if err != nil {
			e.updateMappingStatus(orm, n, err)
			return err
		}

		var valueInObj interface{}
		for _, v := range value {
			valueInObj = v
		}

		err = util.SetNestedField(obj.Object, valueInObj, m.OperandPath)
		if err != nil {
			e.updateMappingStatus(orm, n, err)
			return err
		}

		e.updateMappingStatus(orm, n, err)
	}

	return err
}

func (e *SimpleEnforcer) updateMappingStatus(orm *v1alpha1.OperatorResourceMapping, n int, err error) {
	if err == nil {
		orm.Status.MappedPatterns[n].Mapped = corev1.ConditionTrue
		orm.Status.MappedPatterns[n].Reason = ""
		orm.Status.MappedPatterns[n].Message = ""
	} else {
		orm.Status.MappedPatterns[n].Mapped = corev1.ConditionFalse
		orm.Status.MappedPatterns[n].Reason = string(v1alpha1.ORMStatusReasonOperandError)
		orm.Status.MappedPatterns[n].Message = err.Error()
	}
}

func (m *SimpleEnforcer) CleanupORM(key types.NamespacedName) {
	return
}

func (m *SimpleEnforcer) Start(ctx context.Context) error {
	m.reg.Start(ctx)

	return nil
}

func (m *SimpleEnforcer) SetupWithManager(mgr manager.Manager) error {
	return mgr.Add(m)
}

func GetSimpleEnforcer(config *rest.Config, scheme *runtime.Scheme) (Enforcer, error) {
	var err error

	if enforcer == nil {
		enforcer = &SimpleEnforcer{}
	}

	enforcer.reg, err = kubernetes.GetToolbox(config, scheme)

	return enforcer, err
}
