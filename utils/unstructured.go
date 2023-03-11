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

package util

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/jsonpath"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	uuLog = ctrl.Log.WithName("util unstructured")
)

// NestedField returns the value of a nested field in the given object based on the given JSON-Path.
func NestedField(obj *unstructured.Unstructured, name, path string) (interface{}, bool, error) {
	j := jsonpath.New(name).AllowMissingKeys(true)
	template := fmt.Sprintf("{%s}", path)
	err := j.Parse(template)
	if err != nil {
		return nil, false, err
	}
	results, err := j.FindResults(obj.UnstructuredContent())
	if err != nil {
		return nil, false, err
	}
	if len(results) == 0 || len(results[0]) == 0 {
		return nil, false, nil
	}
	// The input path refers to a unique field, we can assume to have only one result or none.
	value := results[0][0].Interface()

	return value, true, nil
}

func SetNestedField(obj interface{}, value interface{}, path string) error {

	loc := strings.LastIndex(path, ".")
	uppath := path[:loc]
	lastfield := path[loc+1:]

	j := jsonpath.New(lastfield).AllowMissingKeys(true)
	template := fmt.Sprintf("{%s}", uppath)
	err := j.Parse(template)
	if err != nil {
		return err
	}
	results, err := j.FindResults(obj)
	if err != nil {
		return err
	}
	if len(results) == 0 || len(results[0]) == 0 {
		return nil
	}
	// The input path refers to a unique field, we can assume to have only one result or none.
	parent := results[0][0].Interface().(map[string]interface{})
	parent[lastfield] = value

	return nil
}

func PrepareRawExtensionFromUnstructured(obj *unstructured.Unstructured, objPath string) *runtime.RawExtension {

	fields := strings.Split(objPath, ".")
	lastField := fields[len(fields)-1]
	valueInObj, found, err := NestedField(obj, lastField, objPath)

	valueMap := make(map[string]interface{})
	valueMap[lastField] = valueInObj

	if err != nil {
		uuLog.Error(err, "parsing src", "fields", fields, "actual", obj.Object["metadata"])
		return nil
	}
	if !found {
		return nil
	}

	return &runtime.RawExtension{
		Object: &unstructured.Unstructured{
			Object: valueMap,
		},
	}
}
