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

package enforcer

import (
	"github.com/turbonomic/orm/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type registryEntry struct {
	ORM             types.NamespacedName
	EnforcementMode v1alpha1.EnforcementMode
	Mappings        []v1alpha1.Mapping
}

type operandGVRRegistry map[types.NamespacedName]registryEntry
type OperandRegistry map[schema.GroupVersionResource]operandGVRRegistry

var (
	orLog = ctrl.Log.WithName("operand registry")
)