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
	"fmt"
	"time"

	argocdv1alpha "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
	"github.com/argoproj-labs/argocd-rbac-operator/internal/controller/common"
)

const (
	testNamespace       = "default"
	testRBACCMNamespace = "argocd"
	testRBACCMName      = "argocd-rbac-cm"

	testRoleName        = "test-role"
	testRoleBindingName = "test-role-binding"

	testProjectRoleName        = "test-project-role"
	testProjectRoleBindingName = "test-project-role-binding"

	testAppProjectName = "test-appproject"
)

func ZapLogger(development bool) logr.Logger {
	return zap.New(zap.UseDevMode(development))
}

type SchemeOpt func(*runtime.Scheme) error

func addArgoCDPkgToScheme() SchemeOpt {
	return func(s *runtime.Scheme) error {
		if err := argocdv1alpha.AddToScheme(s); err != nil {
			return fmt.Errorf("failed to add ArgoCD API to scheme: %w", err)
		}
		return nil
	}
}

func makeTestArgoCDRoleReconciler(client client.Client, sch *runtime.Scheme) *ArgoCDRoleReconciler {
	return &ArgoCDRoleReconciler{
		Client:                       client,
		Scheme:                       sch,
		ArgoCDRBACConfigMapName:      testRBACCMName,
		ArgoCDRBACConfigMapNamespace: testRBACCMNamespace,
	}
}

func makeTestArgoCDRoleBindingReconciler(client client.Client, sch *runtime.Scheme) *ArgoCDRoleBindingReconciler {
	return &ArgoCDRoleBindingReconciler{
		Client:                       client,
		Scheme:                       sch,
		ArgoCDRBACConfigMapName:      testRBACCMName,
		ArgoCDRBACConfigMapNamespace: testRBACCMNamespace,
	}
}

func makeTestArgoCDProjectRoleReconciler(client client.Client, sch *runtime.Scheme) *ArgoCDProjectRoleReconciler {
	return &ArgoCDProjectRoleReconciler{
		Client: client,
		Scheme: sch,
	}
}

func makeTestArgoCDProjectRoleBindingReconciler(client client.Client, sch *runtime.Scheme) *ArgoCDProjectRoleBindingReconciler {
	return &ArgoCDProjectRoleBindingReconciler{
		Client: client,
		Scheme: sch,
	}
}

func makeTestReconcilerClient(sch *runtime.Scheme, resObjs, subresObjs []client.Object) client.Client {
	client := fake.NewClientBuilder().WithScheme(sch)
	if len(resObjs) > 0 {
		client = client.WithObjects(resObjs...)
	}
	if len(subresObjs) > 0 {
		client = client.WithStatusSubresource(subresObjs...)
	}
	return client.Build()
}

func makeTestReconcilerScheme(schOpts ...SchemeOpt) *runtime.Scheme {
	s := scheme.Scheme
	for _, opt := range schOpts {
		_ = opt(s)
	}
	return s
}

// Global RBAC objects used in tests

type argocdRoleOpt func(*rbacoperatorv1alpha1.ArgoCDRole)

type argocdRoleBindingOpt func(*rbacoperatorv1alpha1.ArgoCDRoleBinding)

func makeTestRole(opts ...argocdRoleOpt) *rbacoperatorv1alpha1.ArgoCDRole {
	r := &rbacoperatorv1alpha1.ArgoCDRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRoleName,
			Namespace: testNamespace,
		},
		Spec: rbacoperatorv1alpha1.ArgoCDRoleSpec{
			Rules: []rbacoperatorv1alpha1.GlobalRule{
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

func makeTestRoleBindingWithRoleSubject(opts ...argocdRoleBindingOpt) *rbacoperatorv1alpha1.ArgoCDRoleBinding {
	rb := &rbacoperatorv1alpha1.ArgoCDRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRoleBindingName,
			Namespace: testNamespace,
		},
		Spec: rbacoperatorv1alpha1.ArgoCDRoleBindingSpec{
			ArgoCDRoleRef: rbacoperatorv1alpha1.ArgoCDRoleRef{
				Name: testRoleName,
			},
			Subjects: []rbacoperatorv1alpha1.GlobalSubject{
				{
					Kind: "role",
					Name: "rb-role-test",
				},
			},
		},
	}
	for _, opt := range opts {
		opt(rb)
	}
	return rb
}

func makeTestRoleBindingWithSSOSubject(opts ...argocdRoleBindingOpt) *rbacoperatorv1alpha1.ArgoCDRoleBinding {
	rb := &rbacoperatorv1alpha1.ArgoCDRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRoleBindingName,
			Namespace: testNamespace,
		},
		Spec: rbacoperatorv1alpha1.ArgoCDRoleBindingSpec{
			ArgoCDRoleRef: rbacoperatorv1alpha1.ArgoCDRoleRef{
				Name: testRoleName,
			},
			Subjects: []rbacoperatorv1alpha1.GlobalSubject{
				{
					Kind: "sso",
					Name: "gosha",
				},
			},
		},
	}
	for _, opt := range opts {
		opt(rb)
	}
	return rb
}

func makeTestRoleBindingWithLocalSubject(opts ...argocdRoleBindingOpt) *rbacoperatorv1alpha1.ArgoCDRoleBinding {
	rb := &rbacoperatorv1alpha1.ArgoCDRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRoleBindingName,
			Namespace: testNamespace,
		},
		Spec: rbacoperatorv1alpha1.ArgoCDRoleBindingSpec{
			ArgoCDRoleRef: rbacoperatorv1alpha1.ArgoCDRoleRef{
				Name: testRoleName,
			},
			Subjects: []rbacoperatorv1alpha1.GlobalSubject{
				{
					Kind: "local",
					Name: "localUser",
				},
			},
		},
	}
	for _, opt := range opts {
		opt(rb)
	}
	return rb
}

func makeTestRoleBindingForBuiltInAdmin(opts ...argocdRoleBindingOpt) *rbacoperatorv1alpha1.ArgoCDRoleBinding {
	rb := &rbacoperatorv1alpha1.ArgoCDRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRoleBindingName,
			Namespace: testNamespace,
		},
		Spec: rbacoperatorv1alpha1.ArgoCDRoleBindingSpec{
			ArgoCDRoleRef: rbacoperatorv1alpha1.ArgoCDRoleRef{
				Name: common.ArgoCDRoleAdmin,
			},
			Subjects: []rbacoperatorv1alpha1.GlobalSubject{
				{
					Kind: "role",
					Name: "rb-role-test",
				},
			},
		},
	}
	for _, opt := range opts {
		opt(rb)
	}
	return rb
}

func makeTestRoleBindingForBuiltInReadOnly(opts ...argocdRoleBindingOpt) *rbacoperatorv1alpha1.ArgoCDRoleBinding {
	rb := &rbacoperatorv1alpha1.ArgoCDRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRoleBindingName,
			Namespace: testNamespace,
		},
		Spec: rbacoperatorv1alpha1.ArgoCDRoleBindingSpec{
			ArgoCDRoleRef: rbacoperatorv1alpha1.ArgoCDRoleRef{
				Name: common.ArgoCDRoleReadOnly,
			},
			Subjects: []rbacoperatorv1alpha1.GlobalSubject{
				{
					Kind: "role",
					Name: "rb-role-test",
				},
			},
		},
	}
	for _, opt := range opts {
		opt(rb)
	}
	return rb
}

func makeTestRBACConfigMap() *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRBACCMName,
			Namespace: testRBACCMNamespace,
		},
		Data: map[string]string{
			"policy.csv": "",
		},
	}
	return cm
}

func makeTestRBACConfigMap_WithChangedPolicyCSV() *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRBACCMName,
			Namespace: testRBACCMNamespace,
		},
		Data: map[string]string{
			"policy.csv": "test",
		},
	}
	return cm
}

func makeTestCMArgoCDRoleExpected() *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRBACCMName,
			Namespace: testRBACCMNamespace,
		},
		Data: map[string]string{
			"policy.csv": "",
			fmt.Sprintf("policy.%s.%s.csv", testNamespace, testRoleName): fmt.Sprintf("p, role:%s, applications, get, */*, allow\np, role:%s, applications, list, */*, allow\n", testRoleName, testRoleName),
		},
	}
	return cm
}

func makeTestCM_ArgoCDRole_WithRoleBindingRoleSubject_Expected() *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRBACCMName,
			Namespace: testRBACCMNamespace,
		},
		Data: map[string]string{
			"policy.csv": "",
			fmt.Sprintf("policy.%s.%s.csv", testNamespace, testRoleName): fmt.Sprintf("p, role:%s, applications, get, */*, allow\np, role:%s, applications, list, */*, allow\ng, role:rb-role-test, role:%s\n", testRoleName, testRoleName, testRoleName),
		},
	}
	return cm
}

func makeTestCM_ArgoCDRole_WithRoleBindingSSOSubject_Expected() *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRBACCMName,
			Namespace: testRBACCMNamespace,
		},
		Data: map[string]string{
			"policy.csv": "",
			fmt.Sprintf("policy.%s.%s.csv", testNamespace, testRoleName): fmt.Sprintf("p, role:%s, applications, get, */*, allow\np, role:%s, applications, list, */*, allow\ng, gosha, role:%s\n", testRoleName, testRoleName, testRoleName),
		},
	}
	return cm
}

func makeTestCM_ArgoCDRole_WithRoleBindingLocalSubject_Expected() *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRBACCMName,
			Namespace: testRBACCMNamespace,
		},
		Data: map[string]string{
			"policy.csv": "",
			fmt.Sprintf("policy.%s.%s.csv", testNamespace, testRoleName): fmt.Sprintf("p, role:%s, applications, get, */*, allow\np, role:%s, applications, list, */*, allow\np, localUser, applications, get, */*, allow\np, localUser, applications, list, */*, allow\n", testRoleName, testRoleName),
		},
	}
	return cm
}

func makeTestCM_BuiltInAdmin_WithRoleBinding_Expected(rbNamespace string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRBACCMName,
			Namespace: testRBACCMNamespace,
		},
		Data: map[string]string{
			"policy.csv": "",
			fmt.Sprintf("policy.%s.%s.csv", rbNamespace, common.ArgoCDRoleAdmin): fmt.Sprintf("g, role:rb-role-test, role:%s\n", common.ArgoCDRoleAdmin),
		},
	}
	return cm
}

func makeTestCM_BuiltInReadOnly_WithRoleBinding_Expected(rbNamespace string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRBACCMName,
			Namespace: testRBACCMNamespace,
		},
		Data: map[string]string{
			"policy.csv": "",
			fmt.Sprintf("policy.%s.%s.csv", rbNamespace, common.ArgoCDRoleReadOnly): fmt.Sprintf("g, role:rb-role-test, role:%s\n", common.ArgoCDRoleReadOnly),
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

func addFinalizerRole() argocdRoleOpt {
	return func(r *rbacoperatorv1alpha1.ArgoCDRole) {
		r.Finalizers = append(r.Finalizers, rbacoperatorv1alpha1.ArgoCDRoleFinalizerName)
	}
}

func roleDeletedAt(now time.Time) argocdRoleOpt {
	return func(r *rbacoperatorv1alpha1.ArgoCDRole) {
		wrapped := metav1.NewTime(now)
		r.DeletionTimestamp = &wrapped
	}
}

func addRoleBinding(roleBindingName string) argocdRoleOpt {
	return func(r *rbacoperatorv1alpha1.ArgoCDRole) {
		r.Status.ArgoCDRoleBindingRef = roleBindingName
	}
}

func addFinalizerRoleBinding() argocdRoleBindingOpt {
	return func(r *rbacoperatorv1alpha1.ArgoCDRoleBinding) {
		r.Finalizers = append(r.Finalizers, rbacoperatorv1alpha1.ArgoCDRoleBindingFinalizerName)
	}
}

func roleBindingDeletedAt(now time.Time) argocdRoleBindingOpt {
	return func(r *rbacoperatorv1alpha1.ArgoCDRoleBinding) {
		wrapped := metav1.NewTime(now)
		r.DeletionTimestamp = &wrapped
	}
}

// AppProject RBAC Objects used in tests

// Options for ArgoCDProjectRole and ArgoCDProjectRoleBinding
type argocdProjectRoleOpt func(*rbacoperatorv1alpha1.ArgoCDProjectRole)

func addFinalizerProjectRole() argocdProjectRoleOpt {
	return func(r *rbacoperatorv1alpha1.ArgoCDProjectRole) {
		r.Finalizers = append(r.Finalizers, rbacoperatorv1alpha1.ArgoCDProjectRoleFinalizerName)
	}
}

func projectRoleDeletedAt(now time.Time) argocdProjectRoleOpt {
	return func(r *rbacoperatorv1alpha1.ArgoCDProjectRole) {
		wrapped := metav1.NewTime(now)
		r.DeletionTimestamp = &wrapped
	}
}

func addProjectRoleBinding(roleBindingName string) argocdProjectRoleOpt {
	return func(r *rbacoperatorv1alpha1.ArgoCDProjectRole) {
		r.Status.ArgoCDProjectRoleBindingRef = roleBindingName
	}
}

type argocdProjectRoleBindingOpt func(*rbacoperatorv1alpha1.ArgoCDProjectRoleBinding)

func addFinalizerProjectRoleBinding() argocdProjectRoleBindingOpt {
	return func(r *rbacoperatorv1alpha1.ArgoCDProjectRoleBinding) {
		r.Finalizers = append(r.Finalizers, rbacoperatorv1alpha1.ArgoCDProjectRoleBindingFinalizerName)
	}
}

func projectRoleBindingDeletedAt(now time.Time) argocdProjectRoleBindingOpt {
	return func(r *rbacoperatorv1alpha1.ArgoCDProjectRoleBinding) {
		wrapped := metav1.NewTime(now)
		r.DeletionTimestamp = &wrapped
	}
}

func addBoundAppProjects(appProjects []string) argocdProjectRoleBindingOpt {
	return func(r *rbacoperatorv1alpha1.ArgoCDProjectRoleBinding) {
		r.Status.AppProjectsBound = appProjects
	}
}

type argocdAppProjectOpt func(*argocdv1alpha.AppProject)

func addTestRoleToAppProject() argocdAppProjectOpt {
	return func(ap *argocdv1alpha.AppProject) {
		ap.Spec.Roles = append(ap.Spec.Roles, argocdv1alpha.ProjectRole{
			Name:        testProjectRoleName,
			Description: "Test Project Role",
			Policies: []string{
				fmt.Sprintf("p, proj:%s:%s, applications, get, */*, allow", testAppProjectName, testProjectRoleName),
				fmt.Sprintf("p, proj:%s:%s, applications, list, */*, allow", testAppProjectName, testProjectRoleName),
				fmt.Sprintf("p, proj:%s:%s, projects, get, *, allow", testAppProjectName, testProjectRoleName),
			},
			Groups: []string{"group1", "group2"},
		})
	}
}

func setAppProjectName(name string) argocdAppProjectOpt {
	return func(ap *argocdv1alpha.AppProject) {
		ap.Name = name
	}
}

// AppProject RBAC Objects

func makeTestAppProject(opts ...argocdAppProjectOpt) *argocdv1alpha.AppProject {
	ap := &argocdv1alpha.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAppProjectName,
			Namespace: testNamespace,
		},
		Spec: argocdv1alpha.AppProjectSpec{
			Description: "Test App Project",
			Roles: []argocdv1alpha.ProjectRole{
				{
					Name:        "existing-role",
					Description: "Existing Role",
				},
			},
		},
	}
	for _, opt := range opts {
		opt(ap)
	}
	return ap
}

func makeTestProjectRole(opts ...argocdProjectRoleOpt) *rbacoperatorv1alpha1.ArgoCDProjectRole {
	r := &rbacoperatorv1alpha1.ArgoCDProjectRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testProjectRoleName,
			Namespace: testNamespace,
		},
		Spec: rbacoperatorv1alpha1.ArgoCDProjectRoleSpec{
			Description: "Test Project Role",
			Rules: []rbacoperatorv1alpha1.ProjectRule{
				{
					Resource: "applications",
					Verbs:    []string{"get", "list"},
					Objects:  []string{"*/*"},
				},
				{
					Resource: "projects",
					Verbs:    []string{"get"},
					Objects:  []string{"*"},
				},
			},
		},
	}

	for _, opt := range opts {
		opt(r)
	}
	return r
}

func makeTestProjectRoleBinding(opts ...argocdProjectRoleBindingOpt) *rbacoperatorv1alpha1.ArgoCDProjectRoleBinding {
	rb := &rbacoperatorv1alpha1.ArgoCDProjectRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testProjectRoleBindingName,
			Namespace: testNamespace,
		},
		Spec: rbacoperatorv1alpha1.ArgoCDProjectRoleBindingSpec{
			ArgoCDProjectRoleRef: rbacoperatorv1alpha1.ArgoCDProjectRoleRef{
				Name: testProjectRoleName,
			},
			Subjects: []rbacoperatorv1alpha1.AppProjectSubject{
				{
					AppProjectRef: testAppProjectName,
					Groups:        []string{"group1", "group2"},
				},
			},
		},
	}

	for _, opt := range opts {
		opt(rb)
	}
	return rb
}
