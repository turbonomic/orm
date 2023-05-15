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

package utils

import (
	"errors"
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
func NestedField(obj interface{}, path string) (interface{}, bool, error) {

	if obj == nil || path == "" {
		return nil, false, nil
	}

	j := jsonpath.New(path).AllowMissingKeys(true)
	template := fmt.Sprintf("{%s}", path)
	err := j.Parse(template)
	if err != nil {
		return nil, false, err
	}
	results, err := j.FindResults(obj)
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

	if obj == nil {
		return errors.New("setting value to nil object")
	}

	if path == "" {
		return errors.New("setting value to empty path")
	}

	var parent interface{}
	var err error
	found := false
	slice := false

	// remove all spaces because it is not allowed as path name
	path = strings.ReplaceAll(path, " ", "")

	loc := len(path)
	lastField := ""
	upperPath := ""

	for loc > 0 {
		loc = strings.LastIndex(path, ".")
		lastField = path[loc+1:]
		upperPath = path[:loc]
		slice = false

		// special processing for slice filter
		// only support format: field[?(@.key==\"value\"")]
		if strings.ContainsAny(lastField, "]") {
			// get full field and remove the "." before key
			tmp := lastField
			path = upperPath
			loc = strings.LastIndex(path, ".")
			lastField = path[loc+1:] + tmp
			upperPath = path[:loc]

			var k, v string
			lastField = strings.ReplaceAll(lastField, "[?(@", " ")
			lastField = strings.ReplaceAll(lastField, "==", " ")
			lastField = strings.ReplaceAll(lastField, "\"", "")
			lastField = strings.ReplaceAll(lastField, ")]", " ")
			n, err := fmt.Sscanf(lastField, "%s %s %s", &lastField, &k, &v)
			if n != 3 || err != nil {
				if err == nil {
					err = fmt.Errorf("failed to parse slice filter, only got %d, need 3", n)
				}
				return err
			}
			value.(map[string]interface{})[k] = v
			value = []map[string]interface{}{value.(map[string]interface{})}
			slice = true
		}

		parent, found, err = NestedField(obj, upperPath)

		if err != nil {
			return err
		}

		if found {
			break
		}

		// full path is not located, try upper path

		value = map[string]interface{}{
			lastField: value,
		}
		path = upperPath
	}

	if found {
		if slice {
			s := parent.(map[string]interface{})[lastField].([]map[string]interface{})
			s = append(s, value.([]map[string]interface{})...)
			parent.(map[string]interface{})[lastField] = s
		} else {
			parent.(map[string]interface{})[lastField] = value
		}
	} else {
		obj.(map[string]interface{})[lastField] = value.(map[string]interface{})[lastField]
	}

	return nil
}

func PrepareRawExtensionFromUnstructured(obj *unstructured.Unstructured, objPath string) *runtime.RawExtension {

	fields := strings.Split(objPath, ".")
	lastField := fields[len(fields)-1]
	valueInObj, found, err := NestedField(obj.Object, objPath)

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
