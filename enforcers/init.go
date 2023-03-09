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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Enforcer interface {
	SetupWithManager(client.Client, *runtime.Scheme, ctrl.Manager) error
}

var enforcers = []Enforcer{
	&SimpleEnforcer{},
}

func SetupAllEnforcersWithManager(mgr ctrl.Manager) error {
	var err error
	for _, e := range enforcers {

		if err = e.SetupWithManager(mgr.GetClient(), mgr.GetScheme(), mgr); err != nil {
			return err
		}
	}

	return nil
}
