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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/argoproj-labs/argocd-rbac-operator/internal/controller/common"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	testNamespace       = "default"
	testRBACCMNamespace = "argocd"

	testRoleName        = "test-role"
	testRoleBindingName = "test-role-binding"
)

func ZapLogger(development bool) logr.Logger {
	return zap.New(zap.UseDevMode(development))
}

type SchemeOpt func(*runtime.Scheme) error

func makeTestArgoCDRoleReconciler(client client.Client, log logr.Logger, sch *runtime.Scheme) *ArgoCDRoleReconciler {
	return &ArgoCDRoleReconciler{
		Client: client,
		Scheme: sch,
		Log:    log,
	}
}

func makeTestArgoCDRoleBindingReconciler(client client.Client, log logr.Logger, sch *runtime.Scheme) *ArgoCDRoleBindingReconciler {
	return &ArgoCDRoleBindingReconciler{
		Client: client,
		Scheme: sch,
		Log:    log,
	}
}

func makeTestReconcilerClient(sch *runtime.Scheme, objs ...runtime.Object) client.Client {
	cl := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(objs...).Build()
	return cl
}

func makeTestReconcilerScheme(schOpts ...SchemeOpt) *runtime.Scheme {
	s := scheme.Scheme
	for _, opt := range schOpts {
		_ = opt(s)
	}
	return s
}

type argocdRoleOpt func(*rbacoperatorv1alpha1.ArgoCDRole)

type argocdRoleBindingOpt func(*rbacoperatorv1alpha1.ArgoCDRoleBinding)

func makeTestRole(opts ...argocdRoleOpt) *rbacoperatorv1alpha1.ArgoCDRole {
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
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func makeTestRBACConfigMap() *corev1.ConfigMap {
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

func makeTestArgoCDNamespace() *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testRBACCMNamespace,
		},
	}
	return ns
}
