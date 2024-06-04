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

	argoprojiov1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
)

const (
	testNamespace = "default"
	testRoleName  = "test-role"
)

// func makeTestRoleWithSSOUser() *argoprojiov1alpha1.Role {
// 	r := &argoprojiov1alpha1.Role{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      testRoleName,
// 			Namespace: testNamespace,
// 		},
// 		Spec: argoprojiov1alpha1.RoleSpec{
// 			Rules: []argoprojiov1alpha1.Rule{
// 				{
// 					Resource: "applications",
// 					Verbs:    []string{"get", "list"},
// 					Objects:  []string{"*/*"},
// 				},
// 			},
// 			Subjects: []argoprojiov1alpha1.Subject{
// 				{
// 					Kind: "sso",
// 					Name: "test-user",
// 				},
// 			},
// 		},
// 	}
// 	return r
// }

// func makeTestRoleWithLocalUser() *argoprojiov1alpha1.Role {
// 	r := &argoprojiov1alpha1.Role{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      testRoleName,
// 			Namespace: testNamespace,
// 		},
// 		Spec: argoprojiov1alpha1.RoleSpec{
// 			Rules: []argoprojiov1alpha1.Rule{
// 				{
// 					Resource: "applications",
// 					Verbs:    []string{"get", "list"},
// 					Objects:  []string{"*/*"},
// 				},
// 			},
// 			Subjects: []argoprojiov1alpha1.Subject{
// 				{
// 					Kind: "local",
// 					Name: "test-local-user",
// 				},
// 			},
// 		},
// 	}
// 	return r
// }

func makeTestRole() *argoprojiov1alpha1.ArgoCDRole {
	r := &argoprojiov1alpha1.ArgoCDRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRoleName,
			Namespace: testNamespace,
		},
		Spec: argoprojiov1alpha1.ArgoCDRoleSpec{
			Rules: []argoprojiov1alpha1.Rule{
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

// func makeRBACConfigMap() *corev1.ConfigMap {
// 	cm := &corev1.ConfigMap{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      common.ArgoCDRBACConfigMapName,
// 			Namespace: common.ArgoCDRBACConfigMapNamespace,
// 		},
// 	}
// 	return cm
// }

func makeArgoCDNamespace() *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "argocd",
		},
	}
	return ns
}
