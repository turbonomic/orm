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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	devopsv1alpha1 "github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/controllers/mappers"
	"github.com/turbonomic/orm/kubernetes"
	"github.com/turbonomic/orm/registry"
	"github.com/turbonomic/orm/util"
)

var (
	acLog = ctrl.Log.WithName("am controller")
)

// AdviceMappingReconciler reconciles a AdviceMapping object
type AdviceMappingReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Registry *registry.ResourceMappingRegistry

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
		return ctrl.Result{}, nil
	}

	if err != nil {
		acLog.Error(err, "reconcile getting "+req.String())
		return ctrl.Result{}, err
	}

	acLog.Info("reconciling advice mapping", "object", req.NamespacedName)

	err = r.compareAndEnforce(am)
	if err != nil {
		acLog.Error(err, "simple enforcement")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AdviceMappingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error

	r.adviceMapper, err = mappers.NewAdviceMapper(r.Registry)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&devopsv1alpha1.AdviceMapping{}).
		Complete(r)
}

func (e *AdviceMappingReconciler) compareAndEnforce(am *devopsv1alpha1.AdviceMapping) error {

	if am.Spec.Owner.Namespace == "" {
		am.Spec.Owner.Namespace = am.Namespace
	}

	original, err := kubernetes.Toolbox.GetResourceWithObjectReference(am.Spec.Owner)
	if err != nil {
		return err
	}

	updated := false
	for _, m := range am.Status.Advices {
		obj := original
		if m.Owner.Name != "" {
			obj, err = kubernetes.Toolbox.GetResourceWithObjectReference(m.Owner.ObjectReference)
		}
		if err != nil {
			acLog.Error(err, "finding true owner", "owner", m.Owner)
			continue
		}

		fields := strings.Split(m.Owner.Path, ".")
		lastField := fields[len(fields)-1]
		valueInOwner, found, err := util.NestedField(obj, lastField, m.Owner.Path)
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
			if found && !reflect.DeepEqual(v, valueInOwner) {
				util.SetNestedField(obj.Object, v, m.Owner.Path)

				if obj != original {
					err = kubernetes.Toolbox.UpdateResourceWithGVK(obj.GroupVersionKind(), obj)
					if err != nil {
						acLog.Error(err, "updating true owner", "owner", m.Owner)
						continue
					}
				} else {
					updated = true
				}
			}
		}
	}

	if updated {
		err = kubernetes.Toolbox.UpdateResourceWithGVK(original.GroupVersionKind(), original)
	}

	return err
}
