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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	argoprojiov1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
)

var _ = Describe("ArgoCDRole Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-role"

		ctx := context.Background()

		typeNamespacedNameRole := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		role := &argoprojiov1alpha1.ArgoCDRole{}

		ns := &corev1.Namespace{}
		typeNamespacedNameNs := types.NamespacedName{
			Name: "argocd",
		}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Role")
			err := k8sClient.Get(ctx, typeNamespacedNameRole, role)
			if err != nil && errors.IsNotFound(err) {
				resource := makeTestRole()
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
			By("creating the namespace for the RBAC ConfigMap")
			err = k8sClient.Get(ctx, typeNamespacedNameNs, ns)
			if err != nil && errors.IsNotFound(err) {
				resource := makeArgoCDNamespace()
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &argoprojiov1alpha1.ArgoCDRole{}
			err := k8sClient.Get(ctx, typeNamespacedNameRole, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Role")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
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
