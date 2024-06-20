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

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("ArgoCDRole Controller", func() {
	Context("When reconciling a resource", func() {

		ctx := context.Background()

		role := makeTestRole()
		typeNamespacedNameRole := types.NamespacedName{
			Name:      testRoleName,
			Namespace: testNamespace,
		}

		nsCM := makeArgoCDNamespace()
		/* 		typeNamespacedNameNsCM := types.NamespacedName{
			Name: testRBACCMNamespace,
		} */

		BeforeEach(func() {
			By("creating the namespace for the RBAC ConfigMap")
			Expect(k8sClient.Create(ctx, nsCM)).Should(Succeed())

			By("creating the RBAC ConfigMap")
			Expect(k8sClient.Create(ctx, makeRBACConfigMap())).Should(Succeed())

			By("creating the custom resource for the Kind Role")
			Expect(k8sClient.Create(ctx, role)).Should(Succeed())
		})

		AfterEach(func() {
			By("Cleanup the specific resource instance Role")
			Expect(k8sClient.Delete(ctx, makeTestRole())).Should(Succeed())

			By("Ensuring the Role is deleted")
			Expect(k8sClient.Get(ctx, typeNamespacedNameRole, makeTestRole())).ShouldNot(Succeed())
			cm := makeRBACConfigMap()
			Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: cm.Namespace, Name: cm.Name}, cm)).Should(Succeed())
			Expect(cm.Data).ShouldNot(HaveKey(fmt.Sprintf("policy.%s.%s.csv", role.Namespace, role.Name)))

			By("Cleanup the RBAC ConfigMap")
			Expect(k8sClient.Delete(ctx, makeRBACConfigMap())).Should(Succeed())

			By("Cleanup the namespace for the RBAC ConfigMap")
			Expect(k8sClient.Delete(ctx, nsCM)).Should(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ArgoCDRoleReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedNameRole,
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
