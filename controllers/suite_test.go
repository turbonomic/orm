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
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
	rec       *OperatorResourceMappingReconciler
)

var testns = corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "testns",
	},
}

var (
	req = types.NamespacedName{
		Name:      "testorm",
		Namespace: "testns",
	}

	testorm = &devopsv1alpha1.OperatorResourceMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: devopsv1alpha1.OperatorResourceMappingSpec{
			Operand: devopsv1alpha1.Operand{
				ObjectReference: corev1.ObjectReference{
					APIVersion: "testgroup.testorg/v1alpha1",
					Kind:       "TestOperatorKind",
				},
				AllowedManagers: []string{},
			},
			Mappings: devopsv1alpha1.MappingPatterns{
				Patterns: []devopsv1alpha1.Pattern{
					{
						OperandPath: "destnation.path",
						Source: devopsv1alpha1.SourceLocation{
							Path: "source.path",
							ObjectReference: corev1.ObjectReference{
								APIVersion: "apps/v1",
								Kind:       "Deployment",
								Name:       "testdeploy",
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

	RunSpecs(t, "ORMController")
}

var _ = Describe("ORMController", func() {

	It("can recocile orm", func() {
		Expect(k8sClient).NotTo(BeNil())
		Expect(testorm.Spec.Mappings.Patterns).NotTo(BeNil())
		err := k8sClient.Create(ctx, testorm)
		Expect(err).NotTo(HaveOccurred())

		_, err = rec.Reconcile(ctx, ctrl.Request{NamespacedName: req})
		Expect(err).NotTo(HaveOccurred())

		obj := &devopsv1alpha1.OperatorResourceMapping{}
		Expect(obj.Spec).ShouldNot(BeEquivalentTo(testorm.Spec))
		err = k8sClient.Get(ctx, req, obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(obj.Spec).Should(BeEquivalentTo(testorm.Spec))
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

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	rec = &OperatorResourceMappingReconciler{
		Client: k8sClient,
		Scheme: scheme.Scheme,
	}

	//+kubebuilder:scaffold:scheme

	err = k8sClient.Create(ctx, &testns)
	Expect(err).NotTo(HaveOccurred())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
