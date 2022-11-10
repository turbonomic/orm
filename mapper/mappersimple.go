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

func (m *SimpleMapper) buildAllPatterns(orm *v1alpha1.OperatorResourceMapping) []v1alpha1.Pattern {
	allpatterns := orm.Spec.Mappings.Patterns
	if orm.Spec.Mappings.Components != nil && len(orm.Spec.Mappings.Components) > 0 {
		var prevpatterns []v1alpha1.Pattern
		for name, list := range orm.Spec.Mappings.Components {
			prevpatterns = allpatterns
			allpatterns = []v1alpha1.Pattern{}
			for _, p := range prevpatterns {
				if strings.Index(p.OperandPath, "{{"+name+"}}") == -1 {
					allpatterns = append(allpatterns, p)
				} else {
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

func (m *SimpleMapper) CreateUpdateSourceRegistryEntries(orm *v1alpha1.OperatorResourceMapping) error {

	var err error

	if orm == nil {
		return nil
	}

	m.reg.CleanupRegistryForORM(types.NamespacedName{
		Namespace: orm.Namespace,
		Name:      orm.Name,
	})

	if orm.Spec.Mappings.Patterns == nil || len(orm.Spec.Mappings.Patterns) == 0 {
		return nil
	}

	allpatterns := m.buildAllPatterns(orm)

	var srcObj *unstructured.Unstructured
	for _, p := range allpatterns {
		k := types.NamespacedName{Namespace: p.Source.Namespace, Name: p.Source.Name}
		if k.Namespace == "" {
			k.Namespace = orm.Namespace
		}

		var srcObjs []unstructured.Unstructured
		if k.Name != "" {
			srcObj, err = m.reg.GetResourceWithGVK(p.Source.GroupVersionKind(), k)
			if err != nil {
				msLog.Error(err, "creating entry for ", "source", p.Source)
				return err
			}
			srcObjs = append(srcObjs, *srcObj)
		} else {
			srcObjs, err = m.reg.GetResourceListWithGVKWithSelector(p.Source.GroupVersionKind(), k, &p.Source.LabelSelector)
			if err != nil {
				msLog.Error(err, "listing resource", "source", p.Source)
			}
		}

		var exists bool

		for _, srcObj := range srcObjs {
			objref := p.Source.ObjectReference.DeepCopy()
			objref.Namespace = srcObj.GetNamespace()
			objref.Name = srcObj.GetName()
			exists, err = m.reg.RegisterSource(p.OperandPath, *objref,
				p.Source.Path,
				types.NamespacedName{Name: orm.Name, Namespace: orm.Namespace})
			if err != nil {
				return err
			}

			pm := make(map[string]string)
			pm[p.OperandPath] = p.Source.Path
			m.mapOnceForOneORM(&srcObj, orm, pm, true)
		}
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

		m.mapOnceForOneORM(obj, orm, pm, false)

		if !reflect.DeepEqual(orgStatus, orm.Status) {
			m.updateORMStatus(orm)
		}

	}

}

func (m *SimpleMapper) mapOnceForOneORM(obj *unstructured.Unstructured, orm *v1alpha1.OperatorResourceMapping, pm registry.PatternMap, force bool) {

	var mgr string

	if !force {
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
	}

	for op, sp := range pm {

		mapitem := v1alpha1.Mapping{}
		mapitem.OperandPath = op

		fields := strings.Split(sp, ".")
		lastField := fields[len(fields)-1]
		valueInObj, found, err := util.NestedField(obj, lastField, sp)

		valueMap := make(map[string]interface{})
		valueMap[lastField] = valueInObj

		if err != nil {
			msLog.Error(err, "parsing src", "fields", fields, "actual", obj.Object["metadata"])
			return
		}
		if !found {
			continue
		}

		valueObj := &unstructured.Unstructured{
			Object: valueMap,
		}
		mapitem.Value = &runtime.RawExtension{
			Object: valueObj,
		}

		exists := false
		for n, mp := range orm.Status.Mappings {
			if mp.OperandPath == mapitem.OperandPath {
				mapitem.DeepCopyInto(&(orm.Status.Mappings[n]))
				exists = true
				break
			}
		}

		if !exists {
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
