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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Utils Suite")
}

var (
	pathSimple      = ".spec.template.spec.container.resources"
	pathWithFilter  = ".spec.template.spec.containers[?(@.name==\"test\")].resources"
	pathWithFilter2 = ".spec.template.spec.containers[?(@.name==\"another\")].resources"
	resourceValue1  = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu": *resource.NewMilliQuantity(1000, resource.DecimalSI),
		},
	}
	resourceValue2 = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu": *resource.NewMilliQuantity(2000, resource.DecimalSI),
		},
	}
)

var _ = Describe("Set Nested_Field", func() {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("group/v1")
	obj.SetKind("Kind")

	It("can create omitted path and fill value", func() {
		err := SetNestedField(obj.Object, resourceValue1, pathSimple)
		Expect(err).NotTo(HaveOccurred())
	})

	It("can get value from omitted path just created", func() {
		v, found, err := NestedField(obj.Object, "resources", pathSimple)
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())
		Expect(v).To(BeEquivalentTo(resourceValue1))
	})

	It("work with existing path and fill value", func() {
		err := SetNestedField(obj.Object, resourceValue2, pathSimple)
		Expect(err).NotTo(HaveOccurred())
	})

	It("can get value from existing path just created", func() {
		v, found, err := NestedField(obj.Object, "resources", pathSimple)
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())
		Expect(v).To(BeEquivalentTo(resourceValue2))
	})
})

var _ = Describe("Set Nested_Field With Slice Filter", func() {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("group/v1")
	obj.SetKind("Kind")

	It("can create omitted path with slice filter and fill value", func() {
		err := SetNestedField(obj.Object, resourceValue1, pathWithFilter)
		Expect(err).NotTo(HaveOccurred())
	})

	It("can get value from omitted path with filter just created", func() {
		v, found, err := NestedField(obj.Object, "resources", pathWithFilter)
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())
		Expect(v).To(BeEquivalentTo(resourceValue1))
	})

	It("can work with existing path with slice filter and fill value", func() {
		err := SetNestedField(obj.Object, resourceValue2, pathWithFilter)
		Expect(err).NotTo(HaveOccurred())
	})

	It("can get value from existing path with filter just created", func() {
		v, found, err := NestedField(obj.Object, "resources", pathWithFilter)
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())
		Expect(v).To(BeEquivalentTo(resourceValue2))
	})

	It("can work with partially existing path with slice filter and fill value", func() {
		err := SetNestedField(obj.Object, resourceValue1, pathWithFilter2)
		Expect(err).NotTo(HaveOccurred())
	})

	It("can get value from partially existing path with filter just created", func() {
		v, found, err := NestedField(obj.Object, "resources", pathWithFilter2)
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())
		Expect(v).To(BeEquivalentTo(resourceValue1))
	})

	It("does not impact value from existing path with filter created before", func() {
		v, found, err := NestedField(obj.Object, "resources", pathWithFilter)
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())
		Expect(v).To(BeEquivalentTo(resourceValue2))
	})
})
