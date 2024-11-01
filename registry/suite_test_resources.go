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

package registry

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	devopsv1alpha1 "github.ibm.com/turbonomic/orm/api/v1alpha1"
)

var (
	_t_ownerpath = ".spec.template.spec.containers[?(@.name==\"" + _t_ownerref.Name + "\")].resources"
	_t_ownedpath = ".spec.template.spec.containers[?(@.name==\"" + _t_ownedref.Name + "\")].resources"

	_t_namespace = "default"

	_t_ormkey = types.NamespacedName{
		Namespace: _t_namespace,
		Name:      "orm",
	}
	_t_ownerref = corev1.ObjectReference{
		Namespace:  _t_namespace,
		Name:       "owner",
		Kind:       "Deployment",
		APIVersion: "apps/v1",
	}
	_t_ownedref = corev1.ObjectReference{
		Namespace:  _t_namespace,
		Name:       "owned",
		Kind:       "Deployment",
		APIVersion: "apps/v1",
	}

	_t_image = "test-image"

	_t_res1 = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu": *resource.NewMilliQuantity(1000, resource.DecimalSI),
		},
	}
	_t_res2 = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu": *resource.NewMilliQuantity(2000, resource.DecimalSI),
		},
	}

	_t_owner = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      _t_ownerref.Name,
			Namespace: _t_ownerref.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": _t_ownerref.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": _t_ownerref.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:      _t_ownerref.Name,
							Image:     _t_image,
							Resources: _t_res1,
						},
					},
				},
			},
		},
	}

	_t_owned = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      _t_ownedref.Name,
			Namespace: _t_ownedref.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": _t_ownedref.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": _t_ownedref.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:      _t_ownedref.Name,
							Image:     _t_image,
							Resources: _t_res2,
						},
					},
				},
			},
		},
	}

	_t_orm = devopsv1alpha1.OperatorResourceMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      _t_ormkey.Name,
			Namespace: _t_ormkey.Namespace,
		},
		Spec: devopsv1alpha1.OperatorResourceMappingSpec{
			Owner: devopsv1alpha1.ObjectLocator{
				ObjectReference: _t_ownerref,
			},
			Mappings: devopsv1alpha1.MappingPatterns{
				Patterns: []devopsv1alpha1.Pattern{
					{
						OwnerPath: _t_ownerpath,
						OwnedResourcePath: devopsv1alpha1.OwnedResourcePath{
							ObjectLocator: devopsv1alpha1.ObjectLocator{
								ObjectReference: _t_ownedref,
							},
							Path: _t_ownedpath,
						},
					},
				},
			},
		},
	}

	_t_ownedOmitPath = ".spec.template.spec.containers[?(@.name==\"" + _t_ownedref.Name + "\")].ports[?(@.protocol==\"TCP\")].containerPort"
	_t_ownerOmitPath = ".spec.template.spec.containers[?(@.name==\"" + _t_ownerref.Name + "\")].ports[?(@.protocol==\"TCP\")].containerPort"

	_t_ormWithOmitPathKey = types.NamespacedName{
		Name:      "ormomitpath",
		Namespace: _t_namespace,
	}
	_t_ormWithOmitPath = devopsv1alpha1.OperatorResourceMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      _t_ormWithOmitPathKey.Name,
			Namespace: _t_ormWithOmitPathKey.Namespace,
		},
		Spec: devopsv1alpha1.OperatorResourceMappingSpec{
			Owner: devopsv1alpha1.ObjectLocator{
				ObjectReference: _t_ownerref,
			},
			Mappings: devopsv1alpha1.MappingPatterns{
				Patterns: []devopsv1alpha1.Pattern{
					{
						OwnerPath: _t_ownerOmitPath,
						OwnedResourcePath: devopsv1alpha1.OwnedResourcePath{
							ObjectLocator: devopsv1alpha1.ObjectLocator{
								ObjectReference: _t_ownedref,
							},
							Path: _t_ownedOmitPath,
						},
					},
				},
			},
		},
	}

	_t_ormMixedKey = types.NamespacedName{
		Name:      "ormmixed",
		Namespace: _t_namespace,
	}
	_t_ormMixed = devopsv1alpha1.OperatorResourceMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      _t_ormMixedKey.Name,
			Namespace: _t_ormMixedKey.Namespace,
		},
		Spec: devopsv1alpha1.OperatorResourceMappingSpec{
			Owner: devopsv1alpha1.ObjectLocator{
				ObjectReference: _t_ownerref,
			},
			Mappings: devopsv1alpha1.MappingPatterns{
				Patterns: []devopsv1alpha1.Pattern{
					{
						OwnerPath: _t_ownerOmitPath,
						OwnedResourcePath: devopsv1alpha1.OwnedResourcePath{
							ObjectLocator: devopsv1alpha1.ObjectLocator{
								ObjectReference: _t_ownedref,
							},
							Path: _t_ownedOmitPath,
						},
					},
					{
						OwnerPath: _t_ownerpath,
						OwnedResourcePath: devopsv1alpha1.OwnedResourcePath{
							ObjectLocator: devopsv1alpha1.ObjectLocator{
								ObjectReference: _t_ownedref,
							},
							Path: _t_ownedpath,
						},
					},
				},
			},
		},
	}

	_t_ownerpathwithvar = ".spec.template.spec.containers[?(@.name==\"" + _t_ownerref.Name + "\")].resources"
	_t_ownedpathwithvar = ".spec.template.spec.containers[?(@.name==\"" + "{{.owned.name}}" + "\")].resources"

	_t_ormWithVarKey = types.NamespacedName{
		Name:      "ormwithvar",
		Namespace: _t_namespace,
	}
	_t_ormWithVar = devopsv1alpha1.OperatorResourceMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      _t_ormWithVarKey.Name,
			Namespace: _t_ormWithVarKey.Namespace,
		},
		Spec: devopsv1alpha1.OperatorResourceMappingSpec{
			Owner: devopsv1alpha1.ObjectLocator{
				ObjectReference: _t_ownerref,
			},
			Mappings: devopsv1alpha1.MappingPatterns{
				Patterns: []devopsv1alpha1.Pattern{
					{
						OwnerPath: _t_ownerpathwithvar,
						OwnedResourcePath: devopsv1alpha1.OwnedResourcePath{
							ObjectLocator: devopsv1alpha1.ObjectLocator{
								ObjectReference: _t_ownedref,
							},
							Path: _t_ownedpathwithvar,
						},
					},
				},
			},
		},
	}
)
