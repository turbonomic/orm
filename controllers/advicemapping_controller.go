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

package controllers

import (
	"context"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	devopsv1alpha1 "github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/controllers/mappers"
	"github.com/turbonomic/orm/kubernetes"
	"github.com/turbonomic/orm/registry"
	ormutils "github.com/turbonomic/orm/utils"
)

var (
	acLog = ctrl.Log.WithName("am controller")
)

// AdviceMappingReconciler reconciles a AdviceMapping object
type AdviceMappingReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	registry *registry.ResourceMappingRegistry

	adviceMapper *mappers.AdviceMapper
}

//+kubebuilder:rbac:groups=devops.turbonomic.io,resources=advicemappings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=devops.turbonomic.io,resources=advicemappings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=devops.turbonomic.io,resources=advicemappings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AdviceMapping object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *AdviceMappingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	am := &devopsv1alpha1.AdviceMapping{}
	err := r.Get(context.TODO(), req.NamespacedName, am)
	if errors.IsNotFound(err) {
		r.registry.CleanupRegistryForAM(req.NamespacedName)
		return ctrl.Result{}, nil
	}
	if err != nil {
		acLog.Error(err, "reconcile getting "+req.String())
		return ctrl.Result{}, err
	}

	acLog.Info("reconciling advice mapping", "object", req.NamespacedName)

	err = r.parseAM(am)
	if err != nil {
		acLog.Error(err, "parse advice mapping")
		return ctrl.Result{}, nil
	}

	r.adviceMapper.MapAdvisorMapping(am)

	// reload to pickup status
	err = r.Get(context.TODO(), req.NamespacedName, am)
	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	err = r.compareAndEnforce(am)
	if err != nil {
		acLog.Error(err, "simple enforcement")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AdviceMappingReconciler) SetupWithManagerAndRegistry(mgr ctrl.Manager, registry *registry.ResourceMappingRegistry) error {

	r.registry = registry
	r.adviceMapper = mappers.NewAdviceMapper(mgr.GetClient(), r.registry)

	if r.adviceMapper == nil {
		return errors.NewServiceUnavailable("Failed to initialize advice mapper")
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&devopsv1alpha1.AdviceMapping{}).
		Complete(r)
}

func (r *AdviceMappingReconciler) parseAM(am *devopsv1alpha1.AdviceMapping) error {
	var err error

	amkey := types.NamespacedName{
		Namespace: am.Namespace,
		Name:      am.Name,
	}

	r.registry.CleanupRegistryForAM(amkey)

	for _, m := range am.Spec.Mappings {
		r.registry.RegisterAdviceMapping(m.TargetResourcePath.Path, m.AdvisorResourcePath.Path, amkey, m.TargetResourcePath.ObjectReference, m.AdvisorResourcePath.ObjectReference)
		r.adviceMapper.RegisterForAdvisor(m.AdvisorResourcePath.GroupVersionKind(),
			types.NamespacedName{
				Namespace: m.AdvisorResourcePath.Namespace,
				Name:      m.AdvisorResourcePath.Name,
			})
	}
	return err
}

func (e *AdviceMappingReconciler) compareAndEnforce(am *devopsv1alpha1.AdviceMapping) error {
	var err error
	var obj *unstructured.Unstructured

	for _, m := range am.Status.Advices {
		if m.Owner.Namespace == "" {
			m.Owner.Namespace = am.Namespace
		}
		obj, err = kubernetes.Toolbox.GetResourceWithObjectReference(m.Owner.ObjectReference)
		if err != nil {
			acLog.Error(err, "finding true owner", "owner", m.Owner)
			continue
		}

		fields := strings.Split(m.Owner.Path, ".")
		lastField := fields[len(fields)-1]
		valueInOwner, found, err := ormutils.NestedField(obj, lastField, m.Owner.Path)
		if err != nil {
			acLog.Error(err, "finding path in owner", "path", m.Owner.Path)
			continue
		}

		value, err := runtime.DefaultUnstructuredConverter.ToUnstructured(m.Value)
		if err != nil {
			acLog.Error(err, "converting value", "value", m.Value)
			continue
		}

		for _, v := range value {
			if !found || !reflect.DeepEqual(v, valueInOwner) {
				ormutils.SetNestedField(obj.Object, v, m.Owner.Path)

				err = kubernetes.Toolbox.UpdateResourceWithGVK(obj.GroupVersionKind(), obj)
				if err != nil {
					acLog.Error(err, "updating true owner", "owner", m.Owner)
					continue
				}
			}
		}
	}

	return err
}
