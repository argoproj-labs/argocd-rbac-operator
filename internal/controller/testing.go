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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
	"github.com/argoproj-labs/argocd-rbac-operator/internal/controller/common"
)

const (
	testNamespace       = "default"
	testRBACCMNamespace = "argocd"

	testRoleName        = "test-role"
	testRoleBindingName = "test-role-binding"
)

func makeTestRole() *rbacoperatorv1alpha1.ArgoCDRole {
	r := &rbacoperatorv1alpha1.ArgoCDRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRoleName,
			Namespace: testNamespace,
		},
		Spec: rbacoperatorv1alpha1.ArgoCDRoleSpec{
			Rules: []rbacoperatorv1alpha1.Rule{
				{
					Resource: "applications",
					Verbs:    []string{"get", "list"},
					Objects:  []string{"*/*"},
				},
			},
		},
	}
	return r
}

func makeRBACConfigMap() *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.ArgoCDRBACConfigMapName,
			Namespace: testRBACCMNamespace,
		},
		Data: map[string]string{
			"policy.csv": "",
		},
	}
	return cm
}

func makeArgoCDNamespace() *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testRBACCMNamespace,
		},
	}
	return ns
}

func makeRBACOperatorNamespace() *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
		},
	}
	return ns
}

func makeTestRoleBinding() *rbacoperatorv1alpha1.ArgoCDRoleBinding {
	rb := &rbacoperatorv1alpha1.ArgoCDRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRoleBindingName,
			Namespace: testNamespace,
		},
		Spec: rbacoperatorv1alpha1.ArgoCDRoleBindingSpec{
			Subjects: []rbacoperatorv1alpha1.Subject{
				{
					Kind: "role",
					Name: "other-test-role",
				},
				{
					Kind: "sso",
					Name: "test-sso-user",
				},
				{
					Kind: "local",
					Name: "test-local-user",
				},
			},
			ArgoCDRoleRef: rbacoperatorv1alpha1.ArgoCDRoleRef{
				Name: testRoleName,
			},
		},
	}
	return rb
}
