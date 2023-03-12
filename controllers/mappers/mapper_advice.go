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

package mappers

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	devopsv1alpha1 "github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/kubernetes"
	"github.com/turbonomic/orm/registry"
	ormutils "github.com/turbonomic/orm/utils"
)

var (
	maLog = ctrl.Log.WithName("Mapper Advice")
)

type AdviceMapper struct {
	reg         *registry.ResourceMappingRegistry
	watchingGVK map[schema.GroupVersionKind]bool

	client.Client
}

func NewAdviceMapper(client client.Client, reg *registry.ResourceMappingRegistry) *AdviceMapper {
	mp := &AdviceMapper{
		Client: client,
		reg:    reg,
	}

	mp.watchingGVK = make(map[schema.GroupVersionKind]bool)

	return mp
}

func (m *AdviceMapper) checkAndReconcileAdvisor(obj *unstructured.Unstructured) {
	objref := corev1.ObjectReference{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
	objref.SetGroupVersionKind(obj.GroupVersionKind())

	if len(m.reg.RetrieveAMEntryForAdvisor(objref)) > 0 {
		maLog.Info("reconciling advisor", "objref", objref)
		m.mapForAdvisor(obj)
	}
}

func (m *AdviceMapper) RegisterForAdvisor(gvk schema.GroupVersionKind, key types.NamespacedName) error {

	if _, ok := m.watchingGVK[gvk]; !ok {
		kubernetes.Toolbox.WatchResourceWithGVK(gvk, cache.ResourceEventHandlerFuncs{
			AddFunc: func(new interface{}) {
				m.checkAndReconcileAdvisor(new.(*unstructured.Unstructured))
			},
			UpdateFunc: func(old, new interface{}) {
				m.checkAndReconcileAdvisor(new.(*unstructured.Unstructured))
			}})
		m.watchingGVK[gvk] = true
	}

	return nil
}

func (m *AdviceMapper) mapForAdvisor(obj *unstructured.Unstructured) {

	// retrieve all AMs associated with this advisor obj
	objref := corev1.ObjectReference{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
	objref.SetGroupVersionKind(obj.GroupVersionKind())
	allams := m.reg.RetrieveAMEntryForAdvisor(objref)

	for amkey, entry := range allams {
		m.mapAdvisorMappingForAdvisor(obj, amkey, entry)
	}

}

func (m *AdviceMapper) MapAdvisorMapping(am *devopsv1alpha1.AdviceMapping) {

	var err error
	var obj *unstructured.Unstructured

	objs := []*unstructured.Unstructured{}
	objrefs := make(map[corev1.ObjectReference]bool)

	for _, mappings := range am.Spec.Mappings {
		if mappings.AdvisorResourcePath.ObjectReference.Namespace == "" {
			mappings.AdvisorResourcePath.ObjectReference.Namespace = am.Namespace
		}
		// avoid to load same object multiple times
		if !objrefs[mappings.AdvisorResourcePath.ObjectReference] {
			objrefs[mappings.AdvisorResourcePath.ObjectReference] = true
			obj, err = kubernetes.Toolbox.GetResourceWithObjectReference(mappings.AdvisorResourcePath.ObjectReference)
			if err != nil {
				maLog.Error(err, "failed to locate target", "objref", mappings.AdvisorResourcePath.ObjectReference)
				continue
			}
			objs = append(objs, obj)
		}
	}

	for _, obj = range objs {
		m.mapForAdvisor(obj)
	}

}

func (m *AdviceMapper) mapAdvisorMappingForAdvisor(advisor *unstructured.Unstructured, amkey types.NamespacedName, entry registry.ObjectEntry) {
	// for each AM, find out the targets objref and path to be advised
	//   for each target objref and path, find out true owner from ORM
	//   update AM status with the advisor value and target and owner
	// 	 check if status is changed and update AM status

	advices := []devopsv1alpha1.Advice{}
	for target, mappings := range entry {
		for advisorPath, targetPath := range mappings {

			owners := ormutils.SeekTopOwnersResourcePathsForTarget(m.reg, devopsv1alpha1.ResourcePath{
				ObjectReference: target,
				Path:            targetPath,
			})

			for _, owner := range owners {
				advice := devopsv1alpha1.Advice{
					Target: devopsv1alpha1.ResourcePath{
						ObjectReference: target,
						Path:            targetPath,
					},
					Value: ormutils.PrepareRawExtensionFromUnstructured(advisor, advisorPath),
					Owner: owner,
				}
				advices = append(advices, advice)
			}
		}
	}

	m.updateAdvisorMappingStatusForAdvisors(amkey, advices)
}

func (m *AdviceMapper) updateAdvisorMappingStatusForAdvisors(amkey types.NamespacedName, advices []devopsv1alpha1.Advice) {
	var err error

	am := &devopsv1alpha1.AdviceMapping{}
	err = m.Get(context.TODO(), amkey, am)
	if err != nil {
		maLog.Error(err, "finding advice mapping", "key", amkey)
		return
	}

	oldadvices := am.Status.Advices
	am.Status.Advices = []devopsv1alpha1.Advice{}
	added := make(map[devopsv1alpha1.ResourcePath]bool)
	for _, oldad := range oldadvices {
		for _, newad := range advices {
			if reflect.DeepEqual(oldad.Target, newad.Target) {
				am.Status.Advices = append(am.Status.Advices, newad)
				added[newad.Target] = true
				break
			}
		}
	}

	for _, newad := range advices {
		if !added[newad.Target] {
			am.Status.Advices = append(am.Status.Advices, newad)
		}
	}

	if !reflect.DeepEqual(oldadvices, am.Status.Advices) {
		err = m.Status().Update(context.TODO(), am)
		if err != nil {
			maLog.Error(err, "updating advisor mapping status", "advice mapping version", am.ObjectMeta.ResourceVersion)
		}
	}

}
