/*
Copyright 2024.

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

package controller

// import (
// 	"context"

// 	. "github.com/onsi/ginkgo/v2"
// 	. "github.com/onsi/gomega"
// 	"k8s.io/apimachinery/pkg/types"
// 	"sigs.k8s.io/controller-runtime/pkg/reconcile"

// 	argoprojiov1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
// )

// var _ = Describe("ArgoCDRoleBinding Controller", func() {
// 	Context("When reconciling a resource", func() {

// 		ctx := context.Background()

// 		rb := makeTestRoleBinding()
// 		typeNamespacedNameRoleBinding := types.NamespacedName{
// 			Name:      testRoleBindingName,
// 			Namespace: testNamespace,
// 		}

// 		nsCM := makeArgoCDNamespace()

// 		BeforeEach(func() {
// 			By("creating the namespace for the Kind RoleBinding")
// 			Expect(k8sClient.Create(ctx, makeRBACOperatorNamespace())).To(Succeed())

// 			By("creating the custom resource for the Kind RoleBinding")
// 			Expect(k8sClient.Create(ctx, rb)).To(Succeed())

// 			By("creating the namespace for the RBAC ConfigMap")
// 			Expect(k8sClient.Create(ctx, nsCM)).To(Succeed())

// 			By("creating the RBAC ConfigMap")
// 			Expect(k8sClient.Create(ctx, makeRBACConfigMap())).To(Succeed())
// 		})

// 		AfterEach(func() {
// 			resource := &argoprojiov1alpha1.ArgoCDRoleBinding{}
// 			err := k8sClient.Get(ctx, typeNamespacedNameRoleBinding, resource)
// 			Expect(err).NotTo(HaveOccurred())

// 			By("Cleanup the specific resource instance ArgoCDRoleBinding")
// 			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

// 			By("Cleanup the namespace for the Kind RoleBinding")
// 			Expect(k8sClient.Delete(ctx, makeRBACOperatorNamespace())).To(Succeed())

// 			By("Cleanup the namespace for the RBAC ConfigMap")
// 			Expect(k8sClient.Delete(ctx, nsCM)).To(Succeed())

// 			By("Cleanup the RBAC ConfigMap")
// 			Expect(k8sClient.Delete(ctx, makeRBACConfigMap())).To(Succeed())
// 		})
// 		It("should successfully reconcile the resource", func() {
// 			By("Reconciling the created resource")
// 			controllerReconciler := &ArgoCDRoleBindingReconciler{
// 				Client: k8sClient,
// 				Scheme: k8sClient.Scheme(),
// 			}

// 			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
// 				NamespacedName: typeNamespacedNameRoleBinding,
// 			})
// 			Expect(err).NotTo(HaveOccurred())
// 		})
// 	})
// })
