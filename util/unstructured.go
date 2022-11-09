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
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

// Set nested field in an unstructured object. Certain field in the given fields could be the key
// of a map of the index of a slice.
func OldSetNestedField(obj interface{}, value interface{}, fields ...string) error {
	m := obj

	for i, field := range fields[:len(fields)-1] {
		if mMap, ok := m.(map[string]interface{}); ok {
			if val, ok := mMap[field]; ok {
				if valMap, ok := val.(map[string]interface{}); ok {
					m = valMap
				} else if valSlice, ok := val.([]interface{}); ok {
					m = valSlice
				} else {
					return fmt.Errorf("value cannot be set because %v is not a map[string]interface{} or a slice []interface{}", JSONPath(fields[:i+1]))
				}
			} else {
				newVal := make(map[string]interface{})
				mMap[field] = newVal
				m = newVal
			}
		} else if mSlice, ok := m.([]interface{}); ok {
			sliceInd, err := strconv.Atoi(field)
			// direct index
			if err == nil {
				if sliceInd < len(mSlice) {
					m = mSlice[sliceInd]
				} else if sliceInd == len(mSlice) {
					mSlice = append(mSlice, make(map[string]interface{}))
					m = mSlice[len(mSlice)-1]
				} else {
					return fmt.Errorf("value cannot be set to the slice path %v because index %v exceeds the slice length %v", JSONPath(fields[:i+1]), field, len(mSlice))
				}
			} else {
				// pattern
			}

		} else {
			return fmt.Errorf("value cannot be set because %v is not a map[string]interface{} or a slice []interface{}", JSONPath(fields[:i+1]))
		}
	}
	lastField := fields[len(fields)-1]
	if mMap, ok := m.(map[string]interface{}); ok {
		mMap[lastField] = value
	} else if mSlice, ok := m.([]interface{}); ok {
		sliceInd, err := strconv.Atoi(lastField)
		if err != nil {
			return fmt.Errorf("value cannot be set to the slice because last field %v in path %v is not integer", lastField, fields)
		}
		mSlice[sliceInd] = value
	} else {
		return fmt.Errorf("value cannot be set because %v is not a map[string]interface{} or a slice []interface{}", JSONPath(fields))
	}
	return nil
}

// JSONPath construct JSON-Path from given slice of fields.
func JSONPath(fields []string) string {
	return "." + strings.Join(fields, ".")
}
