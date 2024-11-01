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
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.ibm.com/turbonomic/orm/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	devopsv1alpha1 "github.ibm.com/turbonomic/orm/api/v1alpha1"
	ormutils "github.ibm.com/turbonomic/orm/utils"
)

var (
	errorMessageMultipleSourceForOwner = "allow 1 and only 1 input from owned.name, owned.selector, owner.labelSelector"
	errorMessageUnknownSelector        = "unknown selector"
	errorMessageSyntaxError            = "syntax error in pattern "
)

func (or *ResourceMappingRegistry) SeekTopOwnersResourcePathsForOwnedResourcePath(owned devopsv1alpha1.ResourcePath) []devopsv1alpha1.ResourcePath {
	owners := []devopsv1alpha1.ResourcePath{owned}

	more := true
	for more {
		more = false
		old := owners
		owners = []devopsv1alpha1.ResourcePath{}
		for _, rp := range old {
			orme := or.retrieveORMEntryForOwned(rp.ObjectReference)
			for _, oe := range orme {
				for o, m := range oe {
					if m[rp.Path] == "" {
						continue
					}
					more = true
					owners = append(owners, devopsv1alpha1.ResourcePath{
						ObjectReference: o,
						Path:            m[rp.Path],
					})
				}
			}
			if !more {
				owners = append(owners, rp)
			}
		}
	}

	return owners

}

const predefinedquotestart = "{{"
const predefinedquoteend = "}}"
const predefinedownedprefix = ".owned"

var predefinedshorthand map[string]string = map[string]string{
	".name":        ".metadata.name",
	".labels":      ".metadata.labels",
	".annotations": ".metadata.annotations",
}

const predefinedParameterPlaceHolder = ".."

func (or *ResourceMappingRegistry) validateORMOwner(orm *devopsv1alpha1.OperatorResourceMapping) (*unstructured.Unstructured, error) {
	var err error
	if orm == nil {
		return nil, nil
	}

	// get owner
	var obj *unstructured.Unstructured
	if orm.Spec.Owner.Name != "" {
		objk := types.NamespacedName{
			Namespace: orm.Spec.Owner.Namespace,
			Name:      orm.Spec.Owner.Name,
		}
		if objk.Namespace == "" {
			objk.Namespace = orm.Namespace
		}
		obj, err = kubernetes.Toolbox.GetResourceWithGVK(orm.Spec.Owner.GroupVersionKind(), objk)
		if err != nil {
			rLog.Error(err, "failed to find owner", "owner", orm.Spec.Owner)
			return nil, err
		}
	} else {
		if orm.Spec.Owner.Namespace == "" {
			orm.Spec.Owner.Namespace = orm.Namespace
		}
		objs, err := kubernetes.Toolbox.GetResourceListWithGVKWithSelector(orm.Spec.Owner.GroupVersionKind(),
			types.NamespacedName{Namespace: orm.Spec.Owner.Namespace, Name: orm.Spec.Owner.Name}, &orm.Spec.Owner.LabelSelector)
		if err != nil || len(objs) == 0 {
			rLog.Error(err, "failed to find owner", "owner", orm.Spec.Owner)
			return nil, err
		}
		obj = &objs[0]
	}

	orm.Spec.Owner.Name = obj.GetName()
	orm.Spec.Owner.Namespace = obj.GetNamespace()

	return obj, nil
}

func (or *ResourceMappingRegistry) SetORMStatusForOwner(owner *unstructured.Unstructured, orm *devopsv1alpha1.OperatorResourceMapping) {
	var err error

	if orm != nil && !or.staticCheckORMSpec(orm) {
		err = kubernetes.Toolbox.OrmClient.Status().Update(context.TODO(), orm)
		if err != nil {
			rLog.Error(err, "static check failed")
		}
		return
	}

	objref := corev1.ObjectReference{}
	objref.Name = owner.GetName()
	objref.Namespace = owner.GetNamespace()
	objref.SetGroupVersionKind(owner.GetObjectKind().GroupVersionKind())

	if orm != nil {
		or.setORMStatus(owner, orm)
		err = kubernetes.Toolbox.OrmClient.Status().Update(context.TODO(), orm)
		if err != nil {
			rLog.Error(err, "updating orm status")
		}
		return
	}

	ormEntry := or.retrieveORMEntryForOwner(objref)
	// orm and owner are 1-1 mapping, ormEntry should be 1 only
	for ormk := range ormEntry {

		orm = &devopsv1alpha1.OperatorResourceMapping{}
		err = kubernetes.Toolbox.OrmClient.Get(context.TODO(), ormk, orm)
		if err != nil {
			rLog.Error(err, "retrieving ", "orm", ormk)
		}

		or.setORMStatus(owner, orm)

		err = kubernetes.Toolbox.OrmClient.Status().Update(context.TODO(), orm)
		if err != nil {
			rLog.Error(err, "updating orm status")
		}
	}
}

func (or *ResourceMappingRegistry) staticCheckPattern(p *devopsv1alpha1.Pattern, selectors map[string]metav1.LabelSelector, parameters map[string][]string) error {
	count := 0
	if p.OwnedResourcePath.Name != "" {
		count++
	}
	if p.OwnedResourcePath.Selector != nil {
		if selectors == nil {
			return errors.New(errorMessageUnknownSelector)
		}

		if _, ok := selectors[*p.OwnedResourcePath.Selector]; !ok {
			return errors.New(errorMessageUnknownSelector)
		}
		count++
	}

	if !reflect.DeepEqual(p.OwnedResourcePath.LabelSelector, metav1.LabelSelector{}) {
		count++
	}

	if count != 1 {
		return errors.New(errorMessageMultipleSourceForOwner)
	}

	return nil
}

func (or *ResourceMappingRegistry) staticCheckORMSpec(orm *devopsv1alpha1.OperatorResourceMapping) bool {
	var passed = true

	if !validateOwnerPathEnabled(orm) {
		orm.Status.Message += "owner path validation is disabled;"
	}

	if !validateOwnedPathEnabled(orm) {
		orm.Status.Message += "owned resource path validation is disabled;"
	}

	errMappingValues := []devopsv1alpha1.OwnerMappingValue{}
	// only allow 1 way to choose owner
	for _, p := range orm.Spec.Mappings.Patterns {
		if err := or.staticCheckPattern(&p, orm.Spec.Mappings.Selectors, orm.Spec.Mappings.Parameters); err != nil {
			mv := devopsv1alpha1.OwnerMappingValue{
				OwnerPath:         p.OwnerPath,
				OwnedResourcePath: &p.OwnedResourcePath,
				Reason:            devopsv1alpha1.ORMStatusReasonOwnedResourceError,
				Message:           err.Error(),
			}
			errMappingValues = append(errMappingValues, mv)
			passed = false
		}
	}

	if !passed {
		orm.Status.OwnerMappingValues = errMappingValues
		orm.Status.State = devopsv1alpha1.ORMTypeError
	}

	return passed
}

func (or *ResourceMappingRegistry) setORMStatus(owner *unstructured.Unstructured, orm *devopsv1alpha1.OperatorResourceMapping) {
	// set owner info and clean previous error status at status top level
	orm.Status = devopsv1alpha1.OperatorResourceMappingStatus{}

	orm.Status.Owner.APIVersion = owner.GetAPIVersion()
	orm.Status.Owner.Kind = owner.GetKind()
	orm.Status.Owner.Namespace = owner.GetNamespace()
	orm.Status.Owner.Name = owner.GetName()
	orm.Status.Message = ""
	orm.Status.Reason = ""
	orm.Status.State = devopsv1alpha1.ORMTypeOK

	orm.Status.OwnerMappingValues = nil

	ownerRef := corev1.ObjectReference{
		Namespace: owner.GetNamespace(),
		Name:      owner.GetName(),
	}
	ownerRef.SetGroupVersionKind(owner.GetObjectKind().GroupVersionKind())

	oe := or.retrieveObjectEntryForOwnerAndORM(ownerRef, types.NamespacedName{
		Namespace: orm.GetNamespace(),
		Name:      orm.GetName(),
	})

	for ownedref, mappings := range *oe {
		ownedobj := &unstructured.Unstructured{}
		ownedobj.SetGroupVersionKind(ownedref.GroupVersionKind())

		var err error
		ownedobj, err = kubernetes.Toolbox.GetResourceWithGVK(ownedref.GroupVersionKind(),
			types.NamespacedName{Namespace: ownedref.Namespace, Name: ownedref.Name})
		// failed to locate owned resource, set error to all paths
		if err != nil {
			for op := range mappings {
				omv := devopsv1alpha1.OwnerMappingValue{
					OwnerPath: op,
					Message:   "Failed to locate owned resource: " + ownedref.String(),
					Reason:    devopsv1alpha1.ORMStatusReasonOwnedResourceError,
				}
				orm.Status.OwnerMappingValues = append(orm.Status.OwnerMappingValues, omv)
			}
			continue
		}

		for op, sp := range mappings {
			omv := devopsv1alpha1.OwnerMappingValue{
				OwnerPath: op,
				OwnedResourcePath: &devopsv1alpha1.OwnedResourcePath{
					ObjectLocator: devopsv1alpha1.ObjectLocator{
						ObjectReference: ownedref,
					},
					Path: sp,
				},
				Value: ormutils.PrepareRawExtensionFromUnstructured(owner, op),
			}

			hide := false
			if omv.Value == nil {
				omv.Message = "Failed to locate ownerPath " + op
				omv.Reason = devopsv1alpha1.ORMStatusReasonOwnerError
				if validateOwnerPathEnabled(orm) {
					orm.Status.OwnerMappingValues = append(orm.Status.OwnerMappingValues, omv)
					continue
				}
				hide = true
			}

			testValue := ormutils.PrepareRawExtensionFromUnstructured(ownedobj, sp)
			if testValue == nil {
				omv.Message = "Failed to locate owned resource path " + sp
				omv.Reason = devopsv1alpha1.ORMStatusReasonOwnedResourceError
				if validateOwnedPathEnabled(orm) {
					orm.Status.OwnerMappingValues = append(orm.Status.OwnerMappingValues, omv)
					continue
				}
				hide = true
			}

			if !hide {
				orm.Status.OwnerMappingValues = append(orm.Status.OwnerMappingValues, omv)
			}
		}
	}

	orm.Status.LastTransitionTime = &metav1.Time{
		Time: time.Now(),
	}
}

func (or *ResourceMappingRegistry) ValidateAndRegisterORM(orm *devopsv1alpha1.OperatorResourceMapping) (*devopsv1alpha1.OperatorResourceMapping, *unstructured.Unstructured, error) {
	var err error

	if orm == nil {
		return nil, nil, nil
	}

	var owner *unstructured.Unstructured
	owner, err = or.validateORMOwner(orm)
	if err != nil {
		return orm, owner, err
	}

	if orm.Spec.Mappings.Patterns == nil || len(orm.Spec.Mappings.Patterns) == 0 {
		return orm, owner, nil
	}

	allpatterns := []devopsv1alpha1.Pattern{}
	allowned := make(map[types.NamespacedName]*unstructured.Unstructured)
	for _, p := range orm.Spec.Mappings.Patterns {

		if or.staticCheckPattern(&p, orm.Spec.Mappings.Selectors, orm.Spec.Mappings.Parameters) != nil {
			continue
		}

		k := types.NamespacedName{Namespace: p.OwnedResourcePath.Namespace, Name: p.OwnedResourcePath.Name}
		if k.Namespace == "" {
			k.Namespace = orm.Namespace
		}

		// TODO: avoid to retrieve same source repeatedly
		if k.Name != "" {
			allpatterns = append(allpatterns, *p.DeepCopy())
			var obj *unstructured.Unstructured
			obj, err = kubernetes.Toolbox.GetResourceWithGVK(p.OwnedResourcePath.GroupVersionKind(), k)
			if err != nil {
				rLog.Error(err, "getting resource", "source", p.OwnedResourcePath)
				continue
			}
			allowned[types.NamespacedName{
				Namespace: obj.GetNamespace(),
				Name:      obj.GetName(),
			}] = obj
		} else {
			var srcObjs []unstructured.Unstructured
			selector := p.OwnedResourcePath.LabelSelector
			if p.OwnedResourcePath.Selector != nil && *p.OwnedResourcePath.Selector != "" {
				selector = orm.Spec.Mappings.Selectors[*p.OwnedResourcePath.Selector]
			}
			srcObjs, err = kubernetes.Toolbox.GetResourceListWithGVKWithSelector(p.OwnedResourcePath.GroupVersionKind(), k, &selector)
			if err != nil {
				rLog.Error(err, "listing resource", "source", p.OwnedResourcePath)
				continue
			}

			if len(srcObjs) == 0 {
				// add a fake pattern to generate error message, use name to carry label selector info
				fakePattern := *p.DeepCopy()
				fakePattern.OwnedResourcePath.ObjectReference.Name = p.OwnedResourcePath.ObjectLocator.LabelSelector.String()
				allpatterns = append(allpatterns, fakePattern)
				continue
			}
			for _, obj := range srcObjs {
				newp := *p.DeepCopy()
				newp.OwnedResourcePath.ObjectReference.Name = obj.GetName()
				newp.OwnedResourcePath.ObjectReference.Namespace = obj.GetNamespace()
				allpatterns = append(allpatterns, newp)
				allowned[types.NamespacedName{
					Namespace: obj.GetNamespace(),
					Name:      obj.GetName(),
				}] = obj.DeepCopy()
			}
		}
	}

	or.CleanupRegistryForORM(types.NamespacedName{
		Namespace: orm.Namespace,
		Name:      orm.Name,
	})

	for _, p := range allpatterns {

		var patterns []devopsv1alpha1.Pattern
		patterns, err = populatePatterns(allowned, orm.Spec.Mappings.Parameters, p)
		if err != nil {
			return orm, owner, err
		}

		for _, pattern := range patterns {
			err = or.registerOwnershipMapping(pattern.OwnerPath, pattern.OwnedResourcePath.Path,
				types.NamespacedName{Name: orm.Name, Namespace: orm.Namespace},
				orm.Spec.Owner.ObjectReference,
				p.OwnedResourcePath.ObjectReference)
			if err != nil {
				return orm, owner, err
			}
		}

	}

	return orm, owner, nil
}

func (or *ResourceMappingRegistry) CleanupRegistryForORM(orm types.NamespacedName) {

	if or.ownerRegistry != nil {
		cleanupORMInRegistry(or.ownerRegistry, orm)
	}
	if or.ownedRegistry != nil {
		cleanupORMInRegistry(or.ownedRegistry, orm)
	}
}

func (or *ResourceMappingRegistry) retrieveORMEntryForOwner(owner corev1.ObjectReference) ResourceMappingEntryType {
	return retrieveResourceMappingEntryForObjectFromRegistry(or.ownerRegistry, owner)
}

func (or *ResourceMappingRegistry) retrieveORMEntryForOwned(owned corev1.ObjectReference) ResourceMappingEntryType {
	return retrieveResourceMappingEntryForObjectFromRegistry(or.ownedRegistry, owned)
}

func (or *ResourceMappingRegistry) retrieveObjectEntryForOwnerAndORM(owner corev1.ObjectReference, orm types.NamespacedName) *ObjectEntry {
	return retrieveObjectEntryForObjectAndORMFromRegistry(or.ownerRegistry, owner, orm)
}

func processShorthandVariables(str string) string {
	result := str

	for k, v := range predefinedshorthand {
		short := predefinedquotestart + predefinedownedprefix + k + predefinedquoteend
		full := predefinedquotestart + predefinedownedprefix + v + predefinedquoteend
		result = strings.ReplaceAll(result, short, full)
	}

	return result
}

func populatePatterns(ownedMap map[types.NamespacedName]*unstructured.Unstructured, parameters map[string][]string, pattern devopsv1alpha1.Pattern) ([]devopsv1alpha1.Pattern, error) {
	var err error
	var allpatterns []devopsv1alpha1.Pattern

	pattern.OwnerPath = processShorthandVariables(pattern.OwnerPath)
	pattern.OwnedResourcePath.Path = processShorthandVariables(pattern.OwnedResourcePath.Path)

	allpatterns = append(allpatterns, pattern)

	if parameters == nil {
		parameters = make(map[string][]string)
	}
	parameters[predefinedParameterPlaceHolder] = []string{predefinedParameterPlaceHolder}
	var prevpatterns []devopsv1alpha1.Pattern
	for name, values := range parameters {
		prevpatterns = allpatterns
		allpatterns = []devopsv1alpha1.Pattern{}

		for _, p := range prevpatterns {
			for _, v := range values {
				newp := p.DeepCopy()
				newp.OwnerPath = strings.ReplaceAll(p.OwnerPath, predefinedquotestart+name+predefinedquoteend, v)
				newp.OwnedResourcePath.Path = strings.ReplaceAll(p.OwnedResourcePath.Path, predefinedquotestart+name+predefinedquoteend, v)
				allpatterns = append(allpatterns, *newp)
			}
		}
	}

	// retrieve and replace generic string value from owned resource
	prevpatterns = allpatterns
	allpatterns = []devopsv1alpha1.Pattern{}
	for _, p := range prevpatterns {
		more := true
		ownedKey := types.NamespacedName{
			Namespace: p.OwnedResourcePath.Namespace,
			Name:      p.OwnedResourcePath.Name,
		}
		ownedObj := ownedMap[ownedKey]
		for more {
			more = false
			var found bool
			p.OwnerPath, found, err = fillOwnedResourceValue(ownedObj, p.OwnerPath)
			if err != nil {
				return allpatterns, err
			}
			if found {
				more = true
			}

			p.OwnedResourcePath.Path, found, err = fillOwnedResourceValue(ownedObj, p.OwnedResourcePath.Path)
			if err != nil {
				return allpatterns, err
			}
			if found {
				more = true
			}
		}
		allpatterns = append(allpatterns, p)
	}

	return allpatterns, nil
}

// const for temporary function processDotInLabelsAndAnnotationKey
const (
	labelsPrefix      = "{{.owned.metadata.labels."
	annotationsPrefix = "{{.owned.metadata.annotations."
	inPathQuoteStart  = "['"
	inPathQuoteEnd    = "']"
)

// processOwnedResourceLabelAndAnnotationValueWithDotInKey
// according to jsonpath spec, dot(.) could be escaped with [' ']
// e.g. the path to the label of following owned resource
// metadata:
//
//	labels:
//	  turbonomic.org/key: "value"
//
// should be
//
//	.owned.metadata.labels.['turbonomic.org/key']
//
// NOT
//
//	.owned.metadata.labels.turbonomic.org/key
//
// This could happen not just in labels and annotations, but normal fields or a CRD as well.
// Ideally this should be addressed in jsonpath library from client-go, but community is not responding to this gap right now
// We develop this feature to ONLY support labels and annotations for now
// until a thorough implementation is provided by client-go library.
func processDotInLabelsAndAnnotationKey(obj *unstructured.Unstructured, pathIn string) (string, error) {
	var err error

	found := true
	path := pathIn
	for found {
		path, found, err = processDotInMapKey(obj.GetLabels(), labelsPrefix, path)
		if err != nil {
			return pathIn, err
		}
	}

	found = true
	for found {
		path, found, err = processDotInMapKey(obj.GetAnnotations(), annotationsPrefix, path)
		if err != nil {
			return pathIn, err
		}
	}

	return path, err
}

func processDotInMapKey(store map[string]string, prefix string, pathIn string) (string, bool, error) {
	if len(store) == 0 {
		return pathIn, false, nil
	}

	start := strings.Index(pathIn, prefix)
	if start == -1 {
		return pathIn, false, nil
	}
	path := pathIn[start:]
	end := strings.Index(path, predefinedquoteend)
	if end == -1 {
		return path, false, errors.New(errorMessageSyntaxError + path)
	}

	full := path[:end+len(predefinedquoteend)]
	key := full[len(prefix) : len(full)-len(predefinedquoteend)]
	key = strings.ReplaceAll(key, inPathQuoteStart, "")
	key = strings.ReplaceAll(key, inPathQuoteEnd, "")
	value := store[key]

	return strings.ReplaceAll(pathIn, full, value), true, nil
}

// fillOwnedResourceValue - use the value from owned resource to fill the variables defined in ORM pattern path
// e.g. replace all {{.owned.metadata.namespace}} with the namespace of owned resource.
// input parameters
// - obj, the owned resource object.
// - path, the path in ORM pattern w/o variables
// return values
// - string, the updated path
// - bool, true: variable found and replacement happened; false: original path returned
// - error, errors found during value extraction
func fillOwnedResourceValue(obj *unstructured.Unstructured, pathIn string) (string, bool, error) {
	if obj == nil {
		return pathIn, false, nil
	}

	path, err := processDotInLabelsAndAnnotationKey(obj, pathIn)
	if err != nil {
		return path, false, err
	}

	start := strings.Index(path, predefinedquotestart)
	if start == -1 {
		return path, false, nil
	}

	end := strings.Index(path, predefinedquoteend)
	if end == -1 {
		return path, false, errors.New(errorMessageSyntaxError + path)
	}

	// description to variables used here:
	// full is {{.owned.xxx.xxx.xxx}} 				- to identify the variable
	// content is .owned.xxx.xxx.xxx from full 		- for syntax checking
	// objPath is .xxx.xxx.xxx from content			- real path in the object to retrieve the value
	full := path[start : end+len(predefinedquoteend)]
	content := full[len(predefinedquotestart) : len(full)-len(predefinedquoteend)]

	if strings.Index(content, predefinedownedprefix) != 0 {
		return path, false, errors.New(errorMessageSyntaxError + path)
	}

	objPath := content[len(predefinedownedprefix):]
	v, found, err := ormutils.NestedField(obj.Object, objPath)

	if err != nil {
		return path, false, err
	}

	if !found {
		return path, false, nil
	}

	switch v.(type) {
	case string:
		break
	default:
		return path, false, errors.New(errorMessageSyntaxError + path)

	}

	return strings.ReplaceAll(path, full, v.(string)), false, nil
}

// ValidateOwnedPathEnabled checks if the annotation is set to "disabled", otherwise it is enabled
func validateOwnedPathEnabled(orm *devopsv1alpha1.OperatorResourceMapping) bool {
	if orm == nil {
		return true
	}

	annotations := orm.GetAnnotations()
	if annotations == nil {
		return true
	}

	v, ok := annotations[devopsv1alpha1.ANNOTATIONKEY_VALIDATE_OWNED_PATH]
	if ok && strings.EqualFold(v, devopsv1alpha1.ANNOTATIONVALUE_DISABLED) {
		return false
	}

	return true
}

// ValidateOwnedPathEnabled checks if the annotation is set to "disabled", otherwise it is enabled
func validateOwnerPathEnabled(orm *devopsv1alpha1.OperatorResourceMapping) bool {
	if orm == nil {
		return true
	}

	annotations := orm.GetAnnotations()
	if annotations == nil {
		return true
	}

	v, ok := annotations[devopsv1alpha1.ANNOTATIONKEY_VALIDATE_OWNER_PATH]
	if ok && strings.EqualFold(v, devopsv1alpha1.ANNOTATIONVALUE_DISABLED) {
		return false
	}

	return true
}
