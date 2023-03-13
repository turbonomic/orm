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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	autoscalingv2 "k8s.io/api/autoscaling/v2"

	devopsv1alpha1 "github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/kubernetes"
)

type HorizontalPodAutoScalerGeneratorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	chLog = ctrl.Log.WithName("hpa-target converter")
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
func (r *HorizontalPodAutoScalerGeneratorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}

	err := r.Get(context.TODO(), req.NamespacedName, hpa)

	if errors.IsNotFound(err) {
		am := &devopsv1alpha1.AdviceMapping{}
		am.Name = req.Name
		am.Namespace = req.Namespace
		r.Delete(context.TODO(), am)
		return ctrl.Result{}, nil
	}

	if err != nil {
		chLog.Error(err, "reconcile getting "+req.String())
		return ctrl.Result{}, err
	}

	chLog.Info("generating am for hpa", "object", req.NamespacedName)

	r.checkAndGenerateAdviceMapping(hpa)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (c *HorizontalPodAutoScalerGeneratorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c.Client = mgr.GetClient()
	c.Scheme = mgr.GetScheme()

	gvk := schema.GroupVersionKind{
		Group:   "autoscaling",
		Version: "v2",
		Kind:    "HorizontalPodAutoscaler",
	}

	if kubernetes.Toolbox.FindGVRfromGVK(gvk) == nil {
		return nil
	}

	autoscalingv2.AddToScheme(c.Scheme)
	chLog.Info("Registering generator", "for", gvk)

	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalingv2.HorizontalPodAutoscaler{}).
		Complete(c)
}

func (r *HorizontalPodAutoScalerGeneratorReconciler) checkAndGenerateAdviceMapping(hpa *autoscalingv2.HorizontalPodAutoscaler) {
	var err error
	am := &devopsv1alpha1.AdviceMapping{}

	key := types.NamespacedName{Namespace: hpa.Namespace, Name: hpa.Name}
	err = r.Get(context.TODO(), key, am)
	if err != nil {
		if !errors.IsNotFound(err) {
			chLog.Error(err, "getting advice mapping", "key", key)
			return
		}
		am.Namespace = hpa.GetNamespace()
		am.Name = hpa.GetName()
		r.updateAdviceMapping(am, hpa)
		err = r.Create(context.TODO(), am)
	} else {
		m := am.Spec.Mappings
		r.updateAdviceMapping(am, hpa)
		if !reflect.DeepEqual(m, am.Spec.Mappings) {
			err = r.Update(context.TODO(), am)
		}
	}

	if err != nil {
		chLog.Error(err, "creating/updating am ", "key", key)
	}
}

const (
	resourceReplicasPath   = ".spec.replicas"
	hpaDesiredReplicasPath = ".status.desiredReplicas"
)

func (r *HorizontalPodAutoScalerGeneratorReconciler) updateAdviceMapping(am *devopsv1alpha1.AdviceMapping, hpa *autoscalingv2.HorizontalPodAutoscaler) {
	if hpa == nil || am == nil {
		return
	}

	targetReference := corev1.ObjectReference{
		APIVersion: hpa.Spec.ScaleTargetRef.APIVersion,
		Kind:       hpa.Spec.ScaleTargetRef.Kind,
		Namespace:  hpa.Namespace,
		Name:       hpa.Spec.ScaleTargetRef.Name,
	}
	advisorReference := corev1.ObjectReference{
		APIVersion: hpa.APIVersion,
		Kind:       hpa.Kind,
		Namespace:  hpa.Namespace,
		Name:       hpa.Name,
	}

	replicas := devopsv1alpha1.AdviceMappingItem{
		TargetResourcePath: devopsv1alpha1.ResourcePath{
			ObjectReference: targetReference,
			Path:            resourceReplicasPath,
		},
		AdvisorResourcePath: devopsv1alpha1.ResourcePath{
			ObjectReference: advisorReference,
			Path:            hpaDesiredReplicasPath,
		},
	}

	found := false
	for _, m := range am.Spec.Mappings {
		if reflect.DeepEqual(m, replicas) {
			found = true
		}
	}
	if !found {
		am.Spec.Mappings = append(am.Spec.Mappings, replicas)
	}
}
