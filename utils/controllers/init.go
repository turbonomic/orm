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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type UtilityController interface {
	reconcile.Reconciler
	SetupWithManager(ctrl.Manager) error
}

var all = []UtilityController{
	&CompatibilityReconciler{},
	&HorizontalPodAutoScalerGeneratorReconciler{},
	&VerticalPodAutoScalerGeneratorReconciler{},
}

func SetupAllWithManager(mgr ctrl.Manager) error {
	var err error
	for _, c := range all {
		err = c.SetupWithManager(mgr)
		if err != nil {
			return err
		}
	}
	return err
}
