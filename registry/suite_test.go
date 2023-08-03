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
	"path/filepath"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	devopsv1alpha1 "github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/kubernetes"
)

var (
	cfg     *rest.Config
	cli     client.Client
	testEnv *envtest.Environment
	ctx     context.Context
	cancel  context.CancelFunc
)

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	ctx, cancel = context.WithCancel(context.TODO())

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = devopsv1alpha1.AddToScheme(testEnv.Scheme)
	Expect(err).NotTo(HaveOccurred())

	cli, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())

	err = cli.Create(ctx, &_t_owner)
	Expect(err).ToNot(HaveOccurred())

	err = cli.Create(ctx, &_t_owned)
	Expect(err).ToNot(HaveOccurred())

	err = cli.Create(ctx, &_t_orm)
	Expect(err).ToNot(HaveOccurred())

	err = cli.Create(ctx, &_t_ormWithOmitPath)
	Expect(err).ToNot(HaveOccurred())

	err = cli.Create(ctx, &_t_ormMixed)
	Expect(err).ToNot(HaveOccurred())

	err = cli.Create(ctx, &_t_ormWithVar)
	Expect(err).ToNot(HaveOccurred())

	err = kubernetes.InitToolbox(cfg, testEnv.Scheme, nil)
	Expect(err).ToNot(HaveOccurred())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var (
	rmr ResourceMappingRegistry
)

var _ = Describe("Registry Test", func() {

	It("can detect empty registry when register ", func() {
		err := registerMappingToRegistry(nil, _t_ownerpath, _t_ownedpath, _t_ormkey, _t_ownerref, _t_ownedref)
		Expect(err).To(HaveOccurred())
	})

	registry := make(map[corev1.ObjectReference]ResourceMappingEntryType)
	It("can register path", func() {
		err := registerMappingToRegistry(registry, _t_ownerpath, _t_ownedpath, _t_ormkey, _t_ownerref, _t_ownedref)
		Expect(err).NotTo(HaveOccurred())
	})

	It("can fetch the path registered", func() {
		oe := retrieveObjectEntryForObjectAndORMFromRegistry(registry, _t_ownedref, _t_ormkey)
		Expect(oe).NotTo(BeNil())
		pmap, ok := (*oe)[_t_ownerref]
		Expect(ok).To(BeTrue())
		Expect(pmap).NotTo(BeNil())
		v, ok := pmap[_t_ownerpath]
		Expect(ok).To(BeTrue())
		Expect(v).To(BeEquivalentTo(_t_ownedpath))
	})

	It("can delete path", func() {
		deleteMappingFromRegistry(registry, _t_ownerpath, _t_ownedpath, _t_ormkey, _t_ownerref, _t_ownedref)
	})

	It("can verify the deleted path is gone", func() {
		oe := retrieveObjectEntryForObjectAndORMFromRegistry(registry, _t_ownedref, _t_ormkey)
		Expect(oe).NotTo(BeNil())
		pmap, ok := (*oe)[_t_ownerref]
		Expect(ok).To(BeTrue())
		Expect(pmap).NotTo(BeNil())
		_, ok = pmap[_t_ownerpath]
		Expect(ok).NotTo(BeTrue())
	})
})

var _ = Describe("ORM Test", func() {

	var ormobj *devopsv1alpha1.OperatorResourceMapping
	var ownerobj *unstructured.Unstructured
	var err error

	It("can register orm", func() {
		ormobj, ownerobj, err = rmr.ValidateAndRegisterORM(&_t_orm)
		Expect(err).ToNot(HaveOccurred())
		Expect(ownerobj).ToNot(BeNil())
		Expect(ormobj).ToNot(BeNil())
	})

	It("can find the path registered by orm in owner registry", func() {
		oe := retrieveObjectEntryForObjectAndORMFromRegistry(rmr.ownerRegistry, _t_ownerref, _t_ormkey)
		Expect(oe).NotTo(BeNil())
		pmap, ok := (*oe)[_t_ownedref]
		Expect(ok).To(BeTrue())
		Expect(pmap).NotTo(BeNil())
		v, ok := pmap[_t_ownerpath]
		Expect(ok).To(BeTrue())
		Expect(v).To(BeEquivalentTo(_t_ownedpath))
	})

	It("can find the path registered by orm in owned registry", func() {
		oe := retrieveObjectEntryForObjectAndORMFromRegistry(rmr.ownedRegistry, _t_ownedref, _t_ormkey)
		Expect(oe).NotTo(BeNil())
		pmap, ok := (*oe)[_t_ownerref]
		Expect(ok).To(BeTrue())
		Expect(pmap).NotTo(BeNil())
		v, ok := pmap[_t_ownedpath]
		Expect(ok).To(BeTrue())
		Expect(v).To(BeEquivalentTo(_t_ownerpath))
	})

	It("can generate status while all fields are valid", func() {
		rmr.SetORMStatusForOwner(ownerobj, ormobj)
		Expect(ormobj.Status.State).To(BeEquivalentTo(devopsv1alpha1.ORMTypeOK))
		Expect(ormobj.Status.Owner).To(BeEquivalentTo(_t_ownerref))
		Expect(ormobj.Status.OwnerMappingValues).NotTo(BeNil())
		Expect(len(ormobj.Status.OwnerMappingValues)).To(BeEquivalentTo(1))
		omv := ormobj.Status.OwnerMappingValues[0]
		Expect(omv.Message).To(BeEmpty())
		Expect(omv.Reason).To(BeEmpty())
		Expect(omv.OwnedResourcePath.ObjectLocator.ObjectReference).To(BeEquivalentTo(_t_ownedref))
		Expect(omv.OwnedResourcePath.Path).To(BeEquivalentTo(_t_ownedpath))
		Expect(omv.OwnerPath).To(BeEquivalentTo(_t_ownerpath))
		uv, err := runtime.DefaultUnstructuredConverter.ToUnstructured(omv.Value)
		Expect(err).ToNot(HaveOccurred())
		rv, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&_t_res1)
		Expect(err).ToNot(HaveOccurred())
		vk := _t_ownerpath[strings.LastIndex(_t_ownerpath, ".")+1:]
		Expect(uv[vk]).To(BeEquivalentTo(rv))
	})

	It("can show error for orm with omitted path while validation is enabled (default)", func() {
		ormobj, ownerobj, err = rmr.ValidateAndRegisterORM(&_t_ormWithOmitPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(ownerobj).ToNot(BeNil())
		Expect(ormobj).ToNot(BeNil())
		rmr.SetORMStatusForOwner(ownerobj, ormobj)
		Expect(ormobj.Status.State).To(BeEquivalentTo(devopsv1alpha1.ORMTypeOK))
		Expect(ormobj.Status.Owner).To(BeEquivalentTo(_t_ownerref))
		Expect(ormobj.Status.OwnerMappingValues).NotTo(BeNil())
		Expect(len(ormobj.Status.OwnerMappingValues)).To(BeEquivalentTo(1))
		omv := ormobj.Status.OwnerMappingValues[0]
		Expect(omv.Message).NotTo(BeEmpty())
		Expect(omv.Reason).To(BeEquivalentTo(devopsv1alpha1.ORMStatusReasonOwnerError))
	})

	It("can show owner path error while owned validation is disabled", func() {
		ormobj, ownerobj, err = rmr.ValidateAndRegisterORM(&_t_ormWithOmitPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(ownerobj).ToNot(BeNil())
		Expect(ormobj).ToNot(BeNil())
		ormobj.Annotations = map[string]string{
			devopsv1alpha1.ANNOTATIONKEY_VALIDATE_OWNED_PATH: devopsv1alpha1.ANNOTATIONVALUE_DISABLED,
		}
		rmr.SetORMStatusForOwner(ownerobj, ormobj)
		Expect(ormobj.Status.State).To(BeEquivalentTo(devopsv1alpha1.ORMTypeOK))
		Expect(ormobj.Status.Owner).To(BeEquivalentTo(_t_ownerref))
		Expect(len(ormobj.Status.OwnerMappingValues)).To(BeEquivalentTo(1))
		omv := ormobj.Status.OwnerMappingValues[0]
		Expect(omv.Message).NotTo(BeEmpty())
		Expect(omv.Reason).To(BeEquivalentTo(devopsv1alpha1.ORMStatusReasonOwnerError))
	})

	It("can show owned resource path error while owner validation is disabled", func() {
		ormobj, ownerobj, err = rmr.ValidateAndRegisterORM(&_t_ormWithOmitPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(ownerobj).ToNot(BeNil())
		Expect(ormobj).ToNot(BeNil())
		ormobj.Annotations = map[string]string{
			devopsv1alpha1.ANNOTATIONKEY_VALIDATE_OWNER_PATH: devopsv1alpha1.ANNOTATIONVALUE_DISABLED,
		}
		// manually fix owner path for testing
		ormobj.Spec.Mappings.Patterns[0].OwnerPath = _t_ownerpath

		rmr.SetORMStatusForOwner(ownerobj, ormobj)
		Expect(ormobj.Status.State).To(BeEquivalentTo(devopsv1alpha1.ORMTypeOK))
		Expect(ormobj.Status.Owner).To(BeEquivalentTo(_t_ownerref))
		Expect(len(ormobj.Status.OwnerMappingValues)).To(BeEquivalentTo(1))
		omv := ormobj.Status.OwnerMappingValues[0]
		Expect(omv.Message).NotTo(BeEmpty())
		Expect(omv.Reason).To(BeEquivalentTo(devopsv1alpha1.ORMStatusReasonOwnedResourceError))
	})

	It("does not show error nor mapping in status for orm with omitted path while validation is disabled", func() {
		ormobj, ownerobj, err = rmr.ValidateAndRegisterORM(&_t_ormWithOmitPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(ownerobj).ToNot(BeNil())
		Expect(ormobj).ToNot(BeNil())
		ormobj.Annotations = map[string]string{
			devopsv1alpha1.ANNOTATIONKEY_VALIDATE_OWNED_PATH: devopsv1alpha1.ANNOTATIONVALUE_DISABLED,
			devopsv1alpha1.ANNOTATIONKEY_VALIDATE_OWNER_PATH: devopsv1alpha1.ANNOTATIONVALUE_DISABLED,
		}
		rmr.SetORMStatusForOwner(ownerobj, ormobj)
		Expect(ormobj.Status.State).To(BeEquivalentTo(devopsv1alpha1.ORMTypeOK))
		Expect(ormobj.Status.Owner).To(BeEquivalentTo(_t_ownerref))
		Expect(len(ormobj.Status.OwnerMappingValues)).To(BeEquivalentTo(0))
	})

	It("can show error for orm with mixed good and bad patterns while validation is enabled (default)", func() {
		ormobj, ownerobj, err = rmr.ValidateAndRegisterORM(&_t_ormMixed)
		Expect(err).ToNot(HaveOccurred())
		Expect(ownerobj).ToNot(BeNil())
		Expect(ormobj).ToNot(BeNil())

		// enable validation
		rmr.SetORMStatusForOwner(ownerobj, ormobj)
		Expect(ormobj.Status.State).To(BeEquivalentTo(devopsv1alpha1.ORMTypeOK))
		Expect(ormobj.Status.Owner).To(BeEquivalentTo(_t_ownerref))
		Expect(ormobj.Status.OwnerMappingValues).NotTo(BeNil())
		Expect(len(ormobj.Status.OwnerMappingValues)).To(BeEquivalentTo(2))

		//there must be 1 good and 1 bad mapping
		n := 0
		if ormobj.Status.OwnerMappingValues[0].Message == "" {
			n = 1 - n
		}
		omv := ormobj.Status.OwnerMappingValues[n]
		Expect(omv.Message).NotTo(BeEmpty())
		Expect(omv.Reason).NotTo(BeEmpty())
		omv = ormobj.Status.OwnerMappingValues[1-n]
		Expect(omv.Message).To(BeEmpty())
		Expect(omv.Reason).To(BeEmpty())
	})

	It("can hide error with mixed good and bad patterns while validation is disabled", func() {
		ormobj, ownerobj, err = rmr.ValidateAndRegisterORM(&_t_ormMixed)

		// disable validation
		ormobj.Annotations = map[string]string{
			devopsv1alpha1.ANNOTATIONKEY_VALIDATE_OWNED_PATH: devopsv1alpha1.ANNOTATIONVALUE_DISABLED,
			devopsv1alpha1.ANNOTATIONKEY_VALIDATE_OWNER_PATH: devopsv1alpha1.ANNOTATIONVALUE_DISABLED,
		}
		rmr.SetORMStatusForOwner(ownerobj, ormobj)
		Expect(ormobj.Status.State).To(BeEquivalentTo(devopsv1alpha1.ORMTypeOK))
		Expect(ormobj.Status.Owner).To(BeEquivalentTo(_t_ownerref))
		Expect(len(ormobj.Status.OwnerMappingValues)).To(BeEquivalentTo(1))
		omv := ormobj.Status.OwnerMappingValues[0]
		Expect(omv.Message).To(BeEmpty())
		Expect(omv.Reason).To(BeEmpty())

	})

	It("can register orm with var in owned path and show status", func() {
		ormobj, ownerobj, err = rmr.ValidateAndRegisterORM(&_t_ormWithVar)
		Expect(err).ToNot(HaveOccurred())
		Expect(ownerobj).ToNot(BeNil())
		Expect(ormobj).ToNot(BeNil())

		rmr.SetORMStatusForOwner(ownerobj, ormobj)
		Expect(ormobj.Status.State).To(BeEquivalentTo(devopsv1alpha1.ORMTypeOK))
		Expect(ormobj.Status.Owner).To(BeEquivalentTo(_t_ownerref))
		Expect(len(ormobj.Status.OwnerMappingValues)).To(BeEquivalentTo(1))
		omv := ormobj.Status.OwnerMappingValues[0]
		Expect(omv.Message).To(BeEmpty())
		Expect(omv.Reason).To(BeEmpty())
	})

	// this test is temporary, should be removed after client-go implemented the feature
	It("can process labels and annotations with dot(.) in the key", func() {
		var (
			normalkey   = "normal"
			normalvalue = "normal"

			key         = "test.dot.key"
			annovalue   = "annotation-value"
			labelvalue  = "label-value"
			annotations = map[string]string{
				key:       annovalue,
				normalkey: normalvalue,
			}
			labels = map[string]string{
				key:       labelvalue,
				normalkey: normalvalue,
			}
			pathInNormal  = ".spec.{{.owned.metadata.labels." + normalkey + "}}.plus.{{.owned.metadata.annotations." + normalkey + "}}"
			pathOutNormal = ".spec." + normalvalue + ".plus." + normalvalue
			pathIn        = ".spec.{{.owned.metadata.labels.['" + key + "']}}.plus.{{.owned.metadata.annotations.['" + key + "']}}"
			pathOut       = ".spec." + labelvalue + ".plus." + annovalue
		)

		obj := &unstructured.Unstructured{}
		obj.SetAnnotations(annotations)
		obj.SetLabels(labels)
		result, err := processDotInLabelsAndAnnotationKey(obj, pathIn)
		Expect(err).To(BeNil())
		Expect(result).To(BeEquivalentTo(pathOut))
		result, err = processDotInLabelsAndAnnotationKey(obj, pathInNormal)
		Expect(err).To(BeNil())
		Expect(result).To(BeEquivalentTo(pathOutNormal))
	})
})

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Utils Suite")
}
