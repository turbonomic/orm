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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	vpa_types "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"

	devopsv1alpha1 "github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/kubernetes"
)

type VerticalPodAutoScalerGeneratorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	cvLog = ctrl.Log.WithName("vpa-target converter")
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
func (r *VerticalPodAutoScalerGeneratorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	vpav1Obj := &vpa_types.VerticalPodAutoscaler{}

	err := r.Get(context.TODO(), req.NamespacedName, vpav1Obj)

	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}

	if err != nil {
		cvLog.Error(err, "reconcile getting "+req.String())
		return ctrl.Result{}, err
	}

	cvLog.Info("generating am for vpa", "object", req.NamespacedName)

	r.checkAndGenerateAdviceMapping(vpav1Obj)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (c *VerticalPodAutoScalerGeneratorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c.Client = mgr.GetClient()
	c.Scheme = mgr.GetScheme()

	gvk := schema.GroupVersionKind{
		Group:   "autoscaling.k8s.io",
		Version: "v1",
		Kind:    "VerticalPodAutoscaler",
	}

	if kubernetes.Toolbox.FindGVRfromGVK(gvk) == nil {
		return nil
	}

	cvLog.Info("Registering generator", "for", gvk)
	vpa_types.AddToScheme(c.Scheme)

	return ctrl.NewControllerManagedBy(mgr).
		For(&vpa_types.VerticalPodAutoscaler{}).
		Complete(c)
}

func (r *VerticalPodAutoScalerGeneratorReconciler) checkAndGenerateAdviceMapping(vpa *vpa_types.VerticalPodAutoscaler) {
	var err error
	am := &devopsv1alpha1.AdviceMapping{}

	key := types.NamespacedName{Namespace: vpa.Namespace, Name: vpa.Name}
	err = r.Get(context.TODO(), key, am)
	if err != nil {
		if !errors.IsNotFound(err) {
			cvLog.Error(err, "getting advice mapping", "key", key)
			return
		}
		am.Namespace = vpa.GetNamespace()
		am.Name = vpa.GetName()
		r.updateAdviceMapping(am, vpa)
		err = r.Create(context.TODO(), am)
	} else {
		m := am.Spec.Mappings
		r.updateAdviceMapping(am, vpa)
		if !reflect.DeepEqual(m, am.Spec.Mappings) {
			err = r.Update(context.TODO(), am)
		}
	}

	if err != nil {
		cvLog.Error(err, "creating/updating am ", "key", key)
	}
}

const (
	nameKeyInPath               = "{{name}}"
	resourceRequestPathTemplate = ".spec.template.spec.containers[?(@.name==\"" + nameKeyInPath + "\")].resources.requests"
	resourceLimitPathTemplate   = ".spec.template.spec.containers[?(@.name==\"" + nameKeyInPath + "\")].resources.limits"
	vpaLowerBoundPathTemplate   = ".status.recommendation.containerRecommendations[?(@.containerName==\"" + nameKeyInPath + "\")].lowerBound"
	vpaUpperBoundPathTemplate   = ".status.recommendation.containerRecommendations[?(@.containerName==\"" + nameKeyInPath + "\")].upperBound"
)

func (r *VerticalPodAutoScalerGeneratorReconciler) updateAdviceMapping(am *devopsv1alpha1.AdviceMapping, vpa *vpa_types.VerticalPodAutoscaler) {
	if vpa == nil || am == nil {
		return
	}

	targetReference := corev1.ObjectReference{
		APIVersion: vpa.Spec.TargetRef.APIVersion,
		Kind:       vpa.Spec.TargetRef.Kind,
		Namespace:  vpa.Namespace,
		Name:       vpa.Spec.TargetRef.Name,
	}
	advisorReference := corev1.ObjectReference{
		APIVersion: vpa.APIVersion,
		Kind:       vpa.Kind,
		Namespace:  vpa.Namespace,
		Name:       vpa.Name,
	}

	for _, container := range vpa.Status.Recommendation.ContainerRecommendations {
		req := devopsv1alpha1.AdviceMappingItem{
			TargetResourcePath: devopsv1alpha1.ResourcePath{
				ObjectReference: targetReference,
				Path:            strings.ReplaceAll(resourceRequestPathTemplate, nameKeyInPath, container.ContainerName),
			},
			AdvisorResourcePath: devopsv1alpha1.ResourcePath{
				ObjectReference: advisorReference,
				Path:            strings.ReplaceAll(vpaLowerBoundPathTemplate, nameKeyInPath, container.ContainerName),
			},
		}

		limit := devopsv1alpha1.AdviceMappingItem{
			TargetResourcePath: devopsv1alpha1.ResourcePath{
				ObjectReference: targetReference,
				Path:            strings.ReplaceAll(resourceLimitPathTemplate, nameKeyInPath, container.ContainerName),
			},
			AdvisorResourcePath: devopsv1alpha1.ResourcePath{
				ObjectReference: advisorReference,
				Path:            strings.ReplaceAll(vpaUpperBoundPathTemplate, nameKeyInPath, container.ContainerName),
			},
		}
		reqFound := false
		limitFound := false
		for _, m := range am.Spec.Mappings {
			if reflect.DeepEqual(m, req) {
				reqFound = true
			}
			if reflect.DeepEqual(m, limit) {
				limitFound = true
			}
		}
		if !reqFound {
			am.Spec.Mappings = append(am.Spec.Mappings, req)
		}
		if !limitFound {
			am.Spec.Mappings = append(am.Spec.Mappings, limit)
		}
	}
}
