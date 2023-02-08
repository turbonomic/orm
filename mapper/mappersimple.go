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
	"reflect"
	"strings"

	"github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/kubernetes"
	"github.com/turbonomic/orm/util"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	annoAllowedManager = "devops.turbonomic.io/allowed-managers"

	msLog = ctrl.Log.WithName("mapper simple")

	mp *SimpleMapper
)

// OperatorResourceMappingReconciler reconciles a OperatorResourceMapping object
type SimpleMapper struct {
	SourceRegistry
}

func (m *SimpleMapper) CleanupORM(key types.NamespacedName) {
	m.CleanupRegistryForORM(key)
}

func (m *SimpleMapper) MapORM(orm *v1alpha1.OperatorResourceMapping) error {

	var err error

	if orm == nil {
		return nil
	}

	m.CleanupRegistryForORM(types.NamespacedName{
		Namespace: orm.Namespace,
		Name:      orm.Name,
	})

	if orm.Spec.Mappings.Patterns == nil || len(orm.Spec.Mappings.Patterns) == 0 {
		return nil
	}

	allpatterns := BuildAllPatterns(orm)

	var srcObj *unstructured.Unstructured
	for _, p := range allpatterns {
		k := types.NamespacedName{Namespace: p.Source.Namespace, Name: p.Source.Name}
		if k.Namespace == "" {
			k.Namespace = orm.Namespace
		}

		// TODO: avoid to retrieve same source repeatedly
		var srcObjs []unstructured.Unstructured
		if k.Name != "" {
			srcObj, err = kubernetes.Toolbox.GetResourceWithGVK(p.Source.GroupVersionKind(), k)
			if err != nil {
				msLog.Error(err, "creating entry for ", "source", p.Source)
				return err
			}
			srcObjs = append(srcObjs, *srcObj)
		} else {
			srcObjs, err = kubernetes.Toolbox.GetResourceListWithGVKWithSelector(p.Source.GroupVersionKind(), k, &p.Source.LabelSelector)
			if err != nil {
				msLog.Error(err, "listing resource", "source", p.Source)
			}
		}

		var exists bool

		for _, srcObj := range srcObjs {
			objref := p.Source.ObjectReference.DeepCopy()
			objref.Namespace = srcObj.GetNamespace()
			objref.Name = srcObj.GetName()
			exists, err = m.RegisterSource(p.OperandPath, *objref,
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
			kubernetes.Toolbox.WatchResourceWithGVK(p.Source.GroupVersionKind(), cache.ResourceEventHandlerFuncs{
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

	re := m.RetriveORMEntryForResource(obj.GroupVersionKind(), k)
	if re == nil {
		return
	}

	for ormk, pm := range re {

		orm := &v1alpha1.OperatorResourceMapping{}
		err = kubernetes.Toolbox.OrmClient.Get(context.TODO(), ormk, orm)
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

func (m *SimpleMapper) isAllowedManager(mgr string, rules map[string]string) bool {
	if rules == nil || len(rules) == 0 {
		return true
	}

	mgrsstr := rules[annoAllowedManager]
	var allowedmgrs []string

	allowedmgrs = strings.Split(strings.TrimSpace(mgrsstr), ",")

	if len(allowedmgrs) > 0 && allowedmgrs[0] != "" {
		found := false
		for _, om := range allowedmgrs {
			if mgr == om {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (m *SimpleMapper) mapOnceForOneORM(obj *unstructured.Unstructured, orm *v1alpha1.OperatorResourceMapping, pm PatternMap, force bool) {

	if !force {
		mfs := obj.GetManagedFields()
		if len(mfs) == 0 {
			msLog.Info("no managed fields", "gvk", obj.GroupVersionKind())
		}

		mgr := mfs[0].Manager
		t := mfs[0].Time
		for _, mf := range obj.GetManagedFields() {
			if t.Before(mf.Time) {
				mgr = mf.Manager
				t = mf.Time
			}
		}

		if !m.isAllowedManager(mgr, obj.GetAnnotations()) {
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
		for n, mp := range orm.Status.MappedPatterns {
			if mp.OperandPath == mapitem.OperandPath {
				mapitem.DeepCopyInto(&(orm.Status.MappedPatterns[n]))
				exists = true
				break
			}
		}

		if !exists {
			orm.Status.MappedPatterns = append(orm.Status.MappedPatterns, mapitem)
		}
	}

	return
}

func (m *SimpleMapper) updateORMStatus(orm *v1alpha1.OperatorResourceMapping) {
	k := types.NamespacedName{
		Namespace: orm.GetNamespace(),
		Name:      orm.GetName(),
	}

	err := kubernetes.Toolbox.OrmClient.Status().Update(context.TODO(), orm)
	if err != nil {
		st := orm.Status.DeepCopy()
		err = kubernetes.Toolbox.OrmClient.Get(context.TODO(), k, orm)
		st.DeepCopyInto(&orm.Status)
		err = kubernetes.Toolbox.OrmClient.Status().Update(context.TODO(), orm)
		if err != nil {
			msLog.Error(err, "retry status")
		}
	}

}

func (m *SimpleMapper) Start(context.Context) error {
	return nil
}

func (m *SimpleMapper) SetupWithManager(mgr manager.Manager) error {
	return mgr.Add(m)
}

func GetSimpleMapper(config *rest.Config, scheme *runtime.Scheme) (Mapper, error) {
	var err error

	if mp == nil {
		mp = &SimpleMapper{}
	}

	return mp, err
}
