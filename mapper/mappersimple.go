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
	"context"
	"errors"
	"strings"

	"github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/registry"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
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

func (m *SimpleMapper) CreateUpdateSourceRegistryEntries(orm *v1alpha1.OperatorResourceMapping) error {

	var err error

	if orm == nil {
		return nil
	}

	if orm.Spec.Patterns == nil || len(orm.Spec.Patterns) == 0 {
		return nil
	}

	for _, p := range orm.Spec.Patterns {
		var exists bool
		exists, err = m.reg.RegisterSource(p.OperandPath, p.Source.ObjectReference, p.Source.Path,
			types.NamespacedName{Name: orm.Name, Namespace: orm.Namespace})
		if err != nil {
			return err
		}

		if !exists {
			m.reg.WatchResourceWithGVK(p.Source.GroupVersionKind(), cache.ResourceEventHandlerFuncs{
				UpdateFunc: func(old, new interface{}) {
					obj := new.(*unstructured.Unstructured)

					k := types.NamespacedName{
						Namespace: obj.GetNamespace(),
						Name:      obj.GetName(),
					}

					se := m.reg.RetriveSource(obj.GroupVersionKind(), k)
					if se == nil {
						return
					}

					orm := &v1alpha1.OperatorResourceMapping{}
					err = m.reg.OrmClient.Get(context.TODO(), se.ORM, orm)
					if err != nil {
						msLog.Error(err, "watching")
					}

					mfs := obj.GetManagedFields()
					if len(mfs) == 0 {
						msLog.Info("no managed fields", "gvk", obj.GroupVersionKind(), "key", k)
					}

					mgr := mfs[0].Manager
					t := mfs[0].Time
					for _, mf := range obj.GetManagedFields() {
						if t.Before(mf.Time) {
							mgr = mf.Manager
							t = mf.Time
						}
					}

					allowedmgrs := orm.Spec.Operand.OtherManagers
					if allowedmgrs == nil || len(allowedmgrs) == 0 {
						allowedmgrs = v1alpha1.DefaultOtherManagers
					}

					found := false
					for _, om := range allowedmgrs {
						if mgr == om {
							found = true
							break
						}
					}
					if !found {
						return
					}

					mapitem := v1alpha1.Mapping{}
					mapitem.OperandPath = se.OperandPath

					fields := strings.Split(se.SourcePath, ".")
					lastField := fields[len(fields)-1]
					valueInObj, found, err := unstructured.NestedFieldCopy(obj.Object, fields...)

					valueMap := make(map[string]interface{})
					valueMap[lastField] = valueInObj
					if err != nil || !found {
						msLog.Error(err, "parsing src", "fields", fields, "actual", obj.Object["metadata"])
						return
					}

					valueObj := &unstructured.Unstructured{
						Object: valueMap,
					}
					mapitem.Value = &runtime.RawExtension{
						Object: valueObj,
					}

					added := false
					for n, mp := range orm.Status.Mappings {
						if mp.OperandPath == mapitem.OperandPath {
							mapitem.DeepCopyInto(&(orm.Status.Mappings[n]))
							added = true
							break
						}
					}

					if !added {
						orm.Status.Mappings = append(orm.Status.Mappings, mapitem)
					}

					err = m.reg.OrmClient.Status().Update(context.TODO(), orm)
					if err != nil {
						st := orm.Status.DeepCopy()
						err = m.reg.OrmClient.Get(context.TODO(), k, orm)
						st.DeepCopyInto(&orm.Status)
						err = m.reg.OrmClient.Status().Update(context.TODO(), orm)
						if err != nil {
							msLog.Error(err, "retry status")
						}
					}
				}})
		}
	}

	return nil
}

func (m *SimpleMapper) mapOnce() error {
	return nil
}

func GetSimpleMapper(r *registry.Registry) (*SimpleMapper, error) {
	if r == nil {
		return nil, errors.New("Null registry for enforcer")
	}

	if mp == nil {
		mp = &SimpleMapper{}
	}

	mp.reg = r

	return mp, nil
}
