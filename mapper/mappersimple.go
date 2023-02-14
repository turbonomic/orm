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
	"github.com/turbonomic/orm/kubernetes"
	"github.com/turbonomic/orm/registry"
	"github.com/turbonomic/orm/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	registry.ORMRegistry

	watchingGVK map[schema.GroupVersionKind]bool
}

func (m *SimpleMapper) CleanupORM(key types.NamespacedName) {
	m.CleanupRegistryForORM(key)
}

func (m *SimpleMapper) MapORM(orm *v1alpha1.OperatorResourceMapping) error {

	objs, err := RegisterORM(orm, &m.ORMRegistry)

	if err != nil {
		return err
	}

	if objs != nil && len(objs) > 0 {
		for objref := range objs {
			obj, err := kubernetes.Toolbox.GetResourceWithGVK(objref.GroupVersionKind(),
				types.NamespacedName{Namespace: objref.Namespace, Name: objref.Name})
			if err != nil {
				msLog.Error(err, "creating entry for ", "objref", objref)
				return err
			}
			ormkey := types.NamespacedName{
				Namespace: orm.Namespace,
				Name:      orm.Name,
			}
			oe := m.RetriveObjectEntryForResourceAndORM(objref, ormkey)
			if oe == nil {
				err = errors.New("Failed to find object entry for " + objref.String() + " for orm " + ormkey.String())
				return err
			}
			m.mapOnceForOneORM(obj, orm, *oe, true)

			if _, ok := m.watchingGVK[objref.GroupVersionKind()]; !ok {
				m.watchingGVK[objref.GroupVersionKind()] = true

				kubernetes.Toolbox.WatchResourceWithGVK(objref.GroupVersionKind(), cache.ResourceEventHandlerFuncs{
					UpdateFunc: func(old, new interface{}) {
						obj := new.(*unstructured.Unstructured)

						m.mapOnce(obj)
					}})
			}
		}
	}

	return err
}

func (m *SimpleMapper) mapOnce(obj *unstructured.Unstructured) {
	var err error

	var orgStatus v1alpha1.OperatorResourceMappingStatus

	objref := corev1.ObjectReference{}
	objref.Name = obj.GetName()
	objref.Namespace = obj.GetNamespace()
	objref.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())

	re := m.ORMRegistry.RetriveORMEntryForResource(objref)
	if re == nil {
		return
	}

	for ormk, oe := range re {

		orm := &v1alpha1.OperatorResourceMapping{}
		err = kubernetes.Toolbox.OrmClient.Get(context.TODO(), ormk, orm)
		if err != nil {
			msLog.Error(err, "watching")
		}
		orm.Status.DeepCopyInto(&orgStatus)

		m.mapOnceForOneORM(obj, orm, oe, false)

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

func (m *SimpleMapper) mapOnceForOneORM(obj *unstructured.Unstructured, orm *v1alpha1.OperatorResourceMapping, oe registry.ObjectEntry, force bool) {

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

	for op, sp := range oe.Mappings {

		mapitem := v1alpha1.Mapping{}
		mapitem.OwnerPath = op

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
			if mp.OwnerPath == mapitem.OwnerPath {
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
		if mp.watchingGVK == nil {
			mp.watchingGVK = make(map[schema.GroupVersionKind]bool)
		}
	}

	return mp, err
}
