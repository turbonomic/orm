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

package kubernetes

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	dynamic.Interface
	OrmClient client.Client
}

func (c *Client) GetResourceListWithGVKWithSelector(gvk schema.GroupVersionKind, req types.NamespacedName, selector *metav1.LabelSelector) ([]unstructured.Unstructured, error) {

	var err error
	gvr, namespaced := Toolbox.FindGVRfromGVK(gvk)
	if gvr == nil {
		return nil, errors.New("Operator " + gvk.String() + "is not installed")
	}

	ls, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, err
	}

	var list *unstructured.UnstructuredList
	if namespaced {
		list, err = Toolbox.Resource(*gvr).Namespace(req.Namespace).List(Toolbox.ctx, metav1.ListOptions{
			LabelSelector: ls.String(),
		})
	} else {
		list, err = Toolbox.Resource(*gvr).List(Toolbox.ctx, metav1.ListOptions{
			LabelSelector: ls.String(),
		})
	}
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

// Search resource within the namespace of req
// Return the resource matches name or anyone exists
func (c *Client) SearchResourceWithGVK(gvk schema.GroupVersionKind, req types.NamespacedName) (*unstructured.Unstructured, error) {
	objects, err := c.GetResourceListWithGVKWithSelector(gvk, req, &metav1.LabelSelector{})

	if err != nil || len(objects) == 0 {
		return nil, err
	}

	obj := &objects[0]
	for _, o := range objects {
		if o.GetName() == req.Name {
			obj = &o
			break
		}
	}

	return obj, nil
}

func (c *Client) GetResourceWithObjectReference(objref corev1.ObjectReference) (*unstructured.Unstructured, error) {
	key := types.NamespacedName{
		Name:      objref.Name,
		Namespace: objref.Namespace,
	}

	return c.GetResourceWithGVK(objref.GroupVersionKind(), key)
}

func (c *Client) GetResourceWithGVK(gvk schema.GroupVersionKind, req types.NamespacedName) (*unstructured.Unstructured, error) {

	var err error
	gvr, namespaced := Toolbox.FindGVRfromGVK(gvk)
	if gvr == nil {
		return nil, errors.New("Operator " + gvk.String() + "is not installed")
	}

	obj := &unstructured.Unstructured{}
	if namespaced {
		obj, err = Toolbox.Resource(*gvr).Namespace(req.Namespace).Get(Toolbox.ctx, req.Name, metav1.GetOptions{})
	} else {
		obj, err = Toolbox.Resource(*gvr).Get(Toolbox.ctx, req.Name, metav1.GetOptions{})
	}

	return obj, err
}

func (c *Client) UpdateResourceWithGVK(gvk schema.GroupVersionKind, obj *unstructured.Unstructured) error {
	var err error

	gvr, namespaced := Toolbox.FindGVRfromGVK(gvk)
	if gvr == nil {
		return errors.New("Resource " + gvk.String() + "is not installed")
	}

	if obj == nil {
		return errors.New("Target resource is not available: " + gvk.String())
	}

	if namespaced {
		_, err = c.Resource(*gvr).Namespace(obj.GetNamespace()).Update(Toolbox.ctx, obj, metav1.UpdateOptions{})
	} else {
		_, err = c.Resource(*gvr).Update(Toolbox.ctx, obj, metav1.UpdateOptions{})
	}

	return err

}
