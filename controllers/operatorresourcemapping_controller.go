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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	devopsv1alpha1 "github.ibm.com/turbonomic/orm/api/v1alpha1"
	"github.ibm.com/turbonomic/orm/controllers/mappers"
	"github.ibm.com/turbonomic/orm/registry"
)

var (
	ocLog = ctrl.Log.WithName("orm controller")
)

// OperatorResourceMappingReconciler reconciles a OperatorResourceMapping object
type OperatorResourceMappingReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	registry        *registry.ResourceMappingRegistry
	ownershipMapper *mappers.OwnershipMapper
}

//+kubebuilder:rbac:groups=devops.turbonomic.io,resources=operatorresourcemappings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=devops.turbonomic.io,resources=operatorresourcemappings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=devops.turbonomic.io,resources=operatorresourcemappings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OperatorResourceMapping object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *OperatorResourceMappingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	orm := &devopsv1alpha1.OperatorResourceMapping{}
	err := r.Get(context.TODO(), req.NamespacedName, orm)

	if errors.IsNotFound(err) {
		r.cleanupORM(req.NamespacedName)
		return ctrl.Result{}, nil
	}

	if err != nil {
		ocLog.Error(err, "reconcile getting "+req.String())
		return ctrl.Result{}, err
	}

	ocLog.Info("reconciling orm", "object", req.NamespacedName)

	oldStatus := orm.Status.DeepCopy()
	orm.Status = devopsv1alpha1.OperatorResourceMappingStatus{}

	err = r.parseORM(orm)
	if err != nil {
		ocLog.Error(err, "registering sources of operator "+req.String()+" ... skipping")

		orm.Status.State = devopsv1alpha1.ORMTypeError
		orm.Status.Reason = devopsv1alpha1.ORMStatusReasonOwnerError
		orm.Status.Message = err.Error()
		r.checkAndUpdateStatus(oldStatus, orm)
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *OperatorResourceMappingReconciler) checkAndUpdateStatus(oldStatus *devopsv1alpha1.OperatorResourceMappingStatus, orm *devopsv1alpha1.OperatorResourceMapping) {
	var err error
	if !reflect.DeepEqual(orm.Status, *oldStatus) {
		err = r.Status().Update(context.TODO(), orm)
	}
	if err != nil {
		ocLog.Error(err, "failed to update orm status "+orm.Namespace+"/"+orm.Name)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *OperatorResourceMappingReconciler) SetupWithManagerAndRegistry(mgr ctrl.Manager, registry *registry.ResourceMappingRegistry) error {

	r.registry = registry
	r.ownershipMapper = mappers.NewOwnershipMapper(r.registry)
	if r.ownershipMapper == nil {
		return errors.NewServiceUnavailable("Failed to initialize advice mapper")
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&devopsv1alpha1.OperatorResourceMapping{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				if e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() {
					return true
				}

				return !reflect.DeepEqual(e.ObjectOld.GetAnnotations(), e.ObjectNew.GetAnnotations())
			}}).
		Complete(r)
}

func (r *OperatorResourceMappingReconciler) cleanupORM(key types.NamespacedName) {
	r.registry.CleanupRegistryForORM(key)
}

func (r *OperatorResourceMappingReconciler) parseORM(orm *devopsv1alpha1.OperatorResourceMapping) error {

	_, _, err := r.registry.ValidateAndRegisterORM(orm)
	if err != nil {
		return err
	}

	return r.ownershipMapper.RegisterOwnerFromORM(orm.Spec.Owner.GroupVersionKind(),
		types.NamespacedName{
			Namespace: orm.Spec.Owner.Namespace,
			Name:      orm.Spec.Owner.Name,
		},
		orm)
}
