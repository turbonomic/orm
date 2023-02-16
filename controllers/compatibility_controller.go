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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	devopsv1alpha1 "github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/kubernetes"
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

	ormv1Obj := &unstructured.Unstructured{}
	ormv1Obj.SetAPIVersion("turbonomic.com/v1alpha1")
	ormv1Obj.SetKind("OperatorResourceMapping")

	err := r.Get(context.TODO(), req.NamespacedName, ormv1Obj)

	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}

	if err != nil {
		ocLog.Error(err, "reconcile getting "+req.String())
		return ctrl.Result{}, err
	}

	ocLog.Info("reconciling ormv1", "object", req.NamespacedName)

	r.compatibilityCheck(ormv1Obj)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (c *CompatibilityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error

	ormv1Obj := &unstructured.Unstructured{}
	ormv1Obj.SetAPIVersion("turbonomic.com/v1alpha1")
	ormv1Obj.SetKind("OperatorResourceMapping")

	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(ormv1Obj).
		Complete(c)
}

func (c *CompatibilityReconciler) compatibilityCheck(ormv1Obj *unstructured.Unstructured) (ctrl.Result, error) {

	if ormv1Obj == nil {
		return ctrl.Result{}, nil
	}

	req := types.NamespacedName{
		Namespace: ormv1Obj.GetNamespace(),
		Name:      ormv1Obj.GetName(),
	}

	orm := &devopsv1alpha1.OperatorResourceMapping{}
	err := c.Get(context.TODO(), req, orm)

	if err != nil && !errors.IsNotFound(err) {
		ocLog.Error(err, "reconcile getting "+req.String())
		return ctrl.Result{}, err
	}

	if errors.IsNotFound(err) {
		err = c.createNewCompatibleORMv2(ormv1Obj)
	} else {
		err = c.updateCompatibleORMv2(ormv1Obj, orm)
	}

	if err != nil {
		ccLog.Error(err, "updating orm")
	}

	return ctrl.Result{}, err
}

func (c *CompatibilityReconciler) createNewCompatibleORMv2(ormv1Obj *unstructured.Unstructured) error {
	var err error
	var neworm *devopsv1alpha1.OperatorResourceMapping

	neworm, err = c.constructCompatibleORMv2(ormv1Obj)

	err = c.Create(context.TODO(), neworm)

	if err != nil {
		ccLog.Error(err, "creating new orm")
	}

	return err
}

func (c *CompatibilityReconciler) updateCompatibleORMv2(ormv1Obj *unstructured.Unstructured, orm *devopsv1alpha1.OperatorResourceMapping) error {
	var err error
	var neworm *devopsv1alpha1.OperatorResourceMapping

	neworm, err = c.constructCompatibleORMv2(ormv1Obj)
	if err != nil {
		return err
	}

	neworm.Spec.EnforcementMode = orm.Spec.EnforcementMode

	if !reflect.DeepEqual(neworm.Spec, orm.Spec) {
		neworm.Spec.DeepCopyInto(&orm.Spec)

		err = c.Update(context.TODO(), orm)
	}

	return err
}

var (
	mappingsPath  = []string{"spec", "resourceMappings"}
	templatePath  = []string{"resourceMappingTemplates"}
	parameterPath = []string{"srcResourceSpec", parameterKey}
	srcKindPath   = []string{"srcResourceSpec", "kind"}
	parameterKey  = "componentNames"
	parameterStr  = "{{.componentName}}"
	dstPathKey    = "destPath"
	srcPathKey    = "srcPath"
)

func (c *CompatibilityReconciler) constructCompatibleORMv2(ormv1Obj *unstructured.Unstructured) (*devopsv1alpha1.OperatorResourceMapping, error) {

	orm := &devopsv1alpha1.OperatorResourceMapping{}

	orm.Name = ormv1Obj.GetName()
	orm.Namespace = ormv1Obj.GetNamespace()

	mappings, found, err := unstructured.NestedSlice(ormv1Obj.Object, mappingsPath...)

	if err != nil {
		return nil, err
	}

	if !found || len(mappings) == 0 {
		return orm, nil
	}

	gvk, _ := kubernetes.Toolbox.FindGVKForResource(ormv1Obj.GetName())
	orm.Spec.Owner.APIVersion, orm.Spec.Owner.Kind = gvk.ToAPIVersionAndKind()
	orm.Spec.EnforcementMode = devopsv1alpha1.EnforcementModeNone

	var templates []interface{}

	for _, mapping := range mappings {
		var paramList []string
		paramList, found, err = unstructured.NestedStringSlice(mapping.(map[string]interface{}), parameterPath...)
		if err != nil {
			return orm, err
		}

		var srcKind string
		srcKind, found, err = unstructured.NestedString(mapping.(map[string]interface{}), srcKindPath...)

		if err != nil {
			return orm, err
		}

		for _, component := range paramList {
			templates, found, err = unstructured.NestedSlice(mapping.(map[string]interface{}), templatePath...)
			if err != nil {
				return orm, err
			}

			var opPathStr, srcPathStr string
			for _, template := range templates {
				opPathStr = template.(map[string]interface{})[dstPathKey].(string)
				opPathStr = strings.ReplaceAll(opPathStr, parameterStr, component)
				srcPathStr = template.(map[string]interface{})[srcPathKey].(string)
				srcPathStr = strings.ReplaceAll(srcPathStr, parameterStr, component)
				pattern := devopsv1alpha1.Pattern{
					OwnerPath: opPathStr,
					OwnedResourcePath: devopsv1alpha1.OwnedResourcePath{
						Path: srcPathStr,
						ObjectLocator: devopsv1alpha1.ObjectLocator{
							ObjectReference: corev1.ObjectReference{
								Kind:       srcKind,
								APIVersion: "apps/v1",
								Name:       component,
							},
						},
					},
				}
				orm.Spec.Mappings.Patterns = append(orm.Spec.Mappings.Patterns, pattern)
			}
		}
	}

	return orm, nil
}
