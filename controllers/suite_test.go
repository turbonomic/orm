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

package controllers

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	devopsv1alpha1 "github.com/turbonomic/orm/api/v1alpha1"
	"github.com/turbonomic/orm/kubernetes"
	"github.com/turbonomic/orm/registry"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg        *rest.Config
	k8sClient  client.Client
	k8sManager ctrl.Manager
	testEnv    *envtest.Environment
	ctx        context.Context
	cancel     context.CancelFunc
)

const (
	objName      = "testorm"
	objNamespace = "default"
)

var (
	req = types.NamespacedName{
		Name:      objName,
		Namespace: objNamespace,
	}

	testorm = &devopsv1alpha1.OperatorResourceMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: devopsv1alpha1.OperatorResourceMappingSpec{
			Owner: devopsv1alpha1.ObjectLocator{
				ObjectReference: corev1.ObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
			},
			Mappings: devopsv1alpha1.MappingPatterns{
				Patterns: []devopsv1alpha1.Pattern{
					{
						OwnerPath: "destination.path",
						OwnedResourcePath: devopsv1alpha1.OwnedResourcePath{
							Path: "source.path",
							ObjectLocator: devopsv1alpha1.ObjectLocator{
								ObjectReference: corev1.ObjectReference{
									APIVersion: "apps/v1",
									Kind:       "Deployment",
									Name:       "deploy",
								},
							},
						},
					},
				},
			},
		},
	}
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = Describe("ORM Resource", func() {

	It("can create orm resource", func() {
		obj := testorm.DeepCopy()
		err := k8sClient.Create(ctx, obj)
		Expect(err).NotTo(HaveOccurred())
	})
})

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

	err = devopsv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).NotTo(BeNil())

	err = kubernetes.InitToolbox(k8sManager.GetConfig(), k8sManager.GetScheme(), k8sManager.GetEventRecorderFor("test"))
	Expect(err).ToNot(HaveOccurred())

	err = (&OperatorResourceMappingReconciler{
		Client: k8sClient,
		Scheme: scheme.Scheme,
	}).SetupWithManagerAndRegistry(k8sManager, &registry.ResourceMappingRegistry{})
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
