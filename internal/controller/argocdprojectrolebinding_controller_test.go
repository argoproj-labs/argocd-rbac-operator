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
	"testing"
	"time"

	argocdv1alpha "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
)

var _ reconcile.Reconciler = &ArgoCDProjectRoleBindingReconciler{}

func TestArgoCDProjectRoleBindingReconciler_Reconcile(t *testing.T) {
	logf.SetLogger(ZapLogger(true))
	argocdProjectRoleBinding := makeTestProjectRoleBinding(addFinalizerProjectRoleBinding())
	argocdProjectRole := makeTestProjectRole()

	resObjs := []client.Object{argocdProjectRoleBinding}
	subresObjs := []client.Object{argocdProjectRoleBinding, argocdProjectRole}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme, addArgoCDPkgToScheme())
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDProjectRoleBindingReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), argocdProjectRole))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestAppProject()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdProjectRoleBinding.Name,
			Namespace: argocdProjectRoleBinding.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter < 5*time.Minute {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}
	projectRoleBindingRes := &rbacoperatorv1alpha1.ArgoCDProjectRoleBinding{}
	err = reconciler.Get(context.TODO(), req.NamespacedName, projectRoleBindingRes)
	assert.NoError(t, err)
	assert.Equal(t, projectRoleBindingRes.Status.AppProjectsBound, []string{testAppProjectName})

	appProject := &argocdv1alpha.AppProject{}
	err = reconciler.Get(context.TODO(), types.NamespacedName{Name: testAppProjectName, Namespace: testNamespace}, appProject)
	assert.NoError(t, err)

	wantAppProject := makeTestAppProject(addTestRoleToAppProject())
	assert.Equal(t, wantAppProject.Spec.Roles, appProject.Spec.Roles)

	projectRole := &rbacoperatorv1alpha1.ArgoCDProjectRole{}
	err = reconciler.Get(context.TODO(), types.NamespacedName{Name: argocdProjectRoleBinding.Spec.ArgoCDProjectRoleRef.Name, Namespace: testNamespace}, projectRole)
	assert.NoError(t, err)

	wantProjectRole := makeTestProjectRole(addProjectRoleBinding(argocdProjectRoleBinding.Name))
	assert.Equal(t, wantProjectRole.Status.ArgoCDProjectRoleBindingRef, projectRole.Status.ArgoCDProjectRoleBindingRef)
}

func TestArgoCDProjectRoleBindingReconciler_AddFinalizer(t *testing.T) {
	logf.SetLogger(ZapLogger(true))
	argocdProjectRoleBinding := makeTestProjectRoleBinding()

	resObjs := []client.Object{argocdProjectRoleBinding}
	subresObjs := []client.Object{argocdProjectRoleBinding}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDProjectRoleBindingReconciler(client, scheme)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdProjectRoleBinding.Name,
			Namespace: argocdProjectRoleBinding.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter < time.Second {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}
	projectRoleBindingRes := &rbacoperatorv1alpha1.ArgoCDProjectRoleBinding{}
	err = reconciler.Get(context.TODO(), req.NamespacedName, projectRoleBindingRes)
	assert.NoError(t, err)

	assert.Contains(t, projectRoleBindingRes.Finalizers, rbacoperatorv1alpha1.ArgoCDProjectRoleBindingFinalizerName)
}

func TestArgoCDProjectRoleBindingReconciler_RoleBindingNotFound(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	resObjs := []client.Object{}
	subresObjs := []client.Object{}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDProjectRoleBindingReconciler(client, scheme)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      testProjectRoleBindingName,
			Namespace: testNamespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter > 0 {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}
	assert.Error(t, reconciler.Get(context.TODO(), types.NamespacedName{Name: testProjectRoleBindingName, Namespace: testNamespace}, &rbacoperatorv1alpha1.ArgoCDProjectRoleBinding{}))
}

func TestArgoCDProjectRoleBindingReconciler_HandleFinalizer(t *testing.T) {
	logf.SetLogger(ZapLogger(true))
	argocdProjectRoleBinding := makeTestProjectRoleBinding(addFinalizerProjectRoleBinding(), projectRoleBindingDeletedAt(time.Now()))

	resObjs := []client.Object{argocdProjectRoleBinding}
	subresObjs := []client.Object{argocdProjectRoleBinding}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme, addArgoCDPkgToScheme())
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDProjectRoleBindingReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestAppProject(addTestRoleToAppProject())))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdProjectRoleBinding.Name,
			Namespace: argocdProjectRoleBinding.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter > 0 {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}

	appProject := &argocdv1alpha.AppProject{}
	err = reconciler.Get(context.TODO(), types.NamespacedName{Name: testAppProjectName, Namespace: testNamespace}, appProject)
	assert.NoError(t, err)
	wantAppProject := makeTestAppProject()
	assert.Equal(t, wantAppProject.Spec.Roles, appProject.Spec.Roles)
}

func TestArgoCDProjectRoleBindingReconciler_RoleNotFound(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	argocdProjectRoleBinding := makeTestProjectRoleBinding(addFinalizerProjectRoleBinding())

	resObjs := []client.Object{argocdProjectRoleBinding}
	subresObjs := []client.Object{argocdProjectRoleBinding}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDProjectRoleBindingReconciler(client, scheme)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdProjectRoleBinding.Name,
			Namespace: argocdProjectRoleBinding.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter < time.Second {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}
	projectRoleBindingRes := &rbacoperatorv1alpha1.ArgoCDProjectRoleBinding{}
	err = reconciler.Get(context.TODO(), req.NamespacedName, projectRoleBindingRes)
	assert.NoError(t, err)
	assert.Contains(t, projectRoleBindingRes.Status.Conditions, rbacoperatorv1alpha1.Condition{
		Type:               rbacoperatorv1alpha1.TypeSynced,
		Status:             corev1.ConditionFalse,
		Reason:             rbacoperatorv1alpha1.ReasonReconcileError,
		Message:            fmt.Sprintf("argocdprojectroles.rbac-operator.argoproj-labs.io \"%s\" not found", argocdProjectRoleBinding.Spec.ArgoCDProjectRoleRef.Name),
		LastTransitionTime: projectRoleBindingRes.Status.Conditions[0].LastTransitionTime,
	})
}

func TestArgoCDProjectRoleBindingReconciler_BoundAppProjectNotInSpec(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	argocdProjectRoleBinding := makeTestProjectRoleBinding(addFinalizerProjectRoleBinding(), addBoundAppProjects([]string{testAppProjectName, "another-app-project"}))

	resObjs := []client.Object{argocdProjectRoleBinding}
	subresObjs := []client.Object{argocdProjectRoleBinding}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme, addArgoCDPkgToScheme())
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDProjectRoleBindingReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestAppProject(addTestRoleToAppProject(), setAppProjectName("another-app-project"))))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestAppProject(addTestRoleToAppProject())))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestProjectRole(addProjectRoleBinding(argocdProjectRoleBinding.Name))))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdProjectRoleBinding.Name,
			Namespace: argocdProjectRoleBinding.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter < 5*time.Minute {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}

	projectRoleBindingRes := &rbacoperatorv1alpha1.ArgoCDProjectRoleBinding{}
	err = reconciler.Get(context.TODO(), req.NamespacedName, projectRoleBindingRes)
	assert.NoError(t, err)
	assert.Equal(t, projectRoleBindingRes.Status.AppProjectsBound, []string{testAppProjectName})

	appProject := &argocdv1alpha.AppProject{}
	err = reconciler.Get(context.TODO(), types.NamespacedName{Name: "another-app-project", Namespace: testNamespace}, appProject)
	assert.NoError(t, err)
	wantAppProject := makeTestAppProject(setAppProjectName("another-app-project"))
	assert.Equal(t, wantAppProject.Spec.Roles, appProject.Spec.Roles)
}
