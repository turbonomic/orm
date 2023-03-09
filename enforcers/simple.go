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

package enforcers

import (
	"context"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	devopsv1alpha1 "github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/kubernetes"
	"github.com/turbonomic/orm/util"
)

var (
	seLog = ctrl.Log.WithName("simple enforcer")
)

type SimpleEnforcer struct {
	client.Client
	Scheme *runtime.Scheme
}

func (e *SimpleEnforcer) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	am := &devopsv1alpha1.AdviceMapping{}
	err := e.Get(context.TODO(), req.NamespacedName, am)

	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}

	if err != nil {
		seLog.Error(err, "reconcile getting "+req.String())
		return ctrl.Result{}, err
	}

	seLog.Info("reconciling advice mapping", "object", req.NamespacedName)

	err = e.compareAndEnforce(am)
	if err != nil {
		seLog.Error(err, "simple enforcement")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (e *SimpleEnforcer) SetupWithManager(client client.Client, scheme *runtime.Scheme, mgr ctrl.Manager) error {
	e.Client = client
	e.Scheme = scheme

	return ctrl.NewControllerManagedBy(mgr).
		For(&devopsv1alpha1.AdviceMapping{}).
		Complete(e)
}

func (e *SimpleEnforcer) compareAndEnforce(am *devopsv1alpha1.AdviceMapping) error {

	if am.Spec.Owner.Namespace == "" {
		am.Spec.Owner.Namespace = am.Namespace
	}

	owner, err := kubernetes.Toolbox.GetResourceWithObjectReference(am.Spec.Owner)
	if err != nil {
		return err
	}

	updated := false
	for _, m := range am.Status.Advices {
		fields := strings.Split(m.OwnerPath, ".")
		lastField := fields[len(fields)-1]
		valueInOwner, found, err := util.NestedField(owner, lastField, m.OwnerPath)
		if err != nil {
			seLog.Error(err, "finding path in owner", "path", m.OwnerPath)
			continue
		}

		value, err := runtime.DefaultUnstructuredConverter.ToUnstructured(m.Value)
		if err != nil {
			seLog.Error(err, "converting value", "value", m.Value)
			continue
		}

		for _, v := range value {

			if found && !reflect.DeepEqual(v, valueInOwner) {
				util.SetNestedField(owner.Object, v, m.OwnerPath)
				updated = true
			}
		}
	}

	if updated {
		err = kubernetes.Toolbox.UpdateResourceWithGVK(owner.GroupVersionKind(), owner)
	}

	return err
}
