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
	"reflect"
	"strings"

	"github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/registry"
	"github.com/turbonomic/orm/util"
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

	var srcObj *unstructured.Unstructured
	for _, p := range orm.Spec.Patterns {
		var exists bool
		exists, err = m.reg.RegisterSource(p.OperandPath, p.Source.ObjectReference, p.Source.Path,
			types.NamespacedName{Name: orm.Name, Namespace: orm.Namespace})
		if err != nil {
			return err
		}

		k := types.NamespacedName{Namespace: p.Source.Namespace, Name: p.Source.Name}
		if k.Namespace == "" {
			k.Namespace = orm.Namespace
		}
		srcObj, err = m.reg.GetResourceWithGVK(p.Source.GroupVersionKind(), k)

		if err != nil {
			msLog.Error(err, "creating entry for ", "source", p.Source)
			return err
		}
		pm := make(map[string]string)
		pm[p.OperandPath] = p.Source.Path
		m.mapOnceForOneORM(srcObj, orm, pm)

		if !exists {
			m.reg.WatchResourceWithGVK(p.Source.GroupVersionKind(), cache.ResourceEventHandlerFuncs{
				UpdateFunc: func(old, new interface{}) {
					obj := new.(*unstructured.Unstructured)

					m.mapOnce(obj)
				}})
		}

	}

	return nil
}

func (m *SimpleMapper) mapOnce(obj *unstructured.Unstructured) {
	var err error

	var orgStatus v1alpha1.OperatorResourceMappingStatus

	k := types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}

	re := m.reg.RetriveORMEntryForResource(obj.GroupVersionKind(), k)
	if re == nil {
		return
	}

	for ormk, pm := range re {

		orm := &v1alpha1.OperatorResourceMapping{}
		err = m.reg.OrmClient.Get(context.TODO(), ormk, orm)
		if err != nil {
			msLog.Error(err, "watching")
		}
		orm.Status.DeepCopyInto(&orgStatus)

		m.mapOnceForOneORM(obj, orm, pm)

		if !reflect.DeepEqual(orgStatus, orm.Status) {
			m.updateORMStatus(orm)
		}

	}

}

// assuming 1 source only serves 1 orm for poc
func (m *SimpleMapper) mapOnceForOneORM(obj *unstructured.Unstructured, orm *v1alpha1.OperatorResourceMapping, pm registry.PatternMap) {

	var mgr string

	mfs := obj.GetManagedFields()
	if len(mfs) == 0 {
		msLog.Info("no managed fields", "gvk", obj.GroupVersionKind())
	}

	mgr = mfs[0].Manager
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

	for op, sp := range pm {

		mapitem := v1alpha1.Mapping{}
		mapitem.OperandPath = op

		fields := strings.Split(sp, ".")
		lastField := fields[len(fields)-1]
		valueInObj, found, err := util.NestedField(obj, lastField, sp)

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
	}

	return
}

func (m *SimpleMapper) updateORMStatus(orm *v1alpha1.OperatorResourceMapping) {
	k := types.NamespacedName{
		Namespace: orm.GetNamespace(),
		Name:      orm.GetName(),
	}

	err := m.reg.OrmClient.Status().Update(context.TODO(), orm)
	if err != nil {
		st := orm.Status.DeepCopy()
		err = m.reg.OrmClient.Get(context.TODO(), k, orm)
		st.DeepCopyInto(&orm.Status)
		err = m.reg.OrmClient.Status().Update(context.TODO(), orm)
		if err != nil {
			msLog.Error(err, "retry status")
		}
	}

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
