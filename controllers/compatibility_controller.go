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

	"github.com/turbonomic/orm/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type CompatibilityReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	ccLog = ctrl.Log.WithName("compatibility controller")
)

//+kubebuilder:rbac:groups=devops.turbonomic.io,resources=operatorresourcemappings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=devops.turbonomic.io,resources=operatorresourcemappings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=devops.turbonomic.io,resources=operatorresourcemappings/finalizers,verbs=update

//+kubebuilder:rbac:groups=turbonomic.com,resources=operatorresourcemappings,verbs=get;list;watch
//+kubebuilder:rbac:groups=turbonomic.com,resources=operatorresourcemappings/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OperatorResourceMapping object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *CompatibilityReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	orm := &v1alpha1.OperatorResourceMapping{}
	err := r.Get(context.TODO(), req.NamespacedName, orm)

	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}

	if err != nil {
		ocLog.Error(err, "reconcile getting "+req.String())
		return ctrl.Result{}, err
	}

	ocLog.Info("reconciling", "operand", orm.Spec.Operand, "mappings", orm.Spec.Mappings, "Status", orm.Status.MappedPatterns)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (c *CompatibilityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ormv1Obj := &unstructured.Unstructured{}
	ormv1Obj.SetAPIVersion("turbonomic.com/v1alpha1")
	ormv1Obj.SetKind("OperatorResourceMapping")

	return ctrl.NewControllerManagedBy(mgr).
		For(ormv1Obj).
		Complete(c)
}
