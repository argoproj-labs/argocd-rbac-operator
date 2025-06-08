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
	"testing"
	"time"

	argocdv1alpha "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
)

var _ reconcile.Reconciler = &ArgoCDProjectRoleReconciler{}

func TestArgoCDProjectRoleReconciler_Reconcile(t *testing.T) {
	logf.SetLogger(ZapLogger(true))
	argocdProjectRole := makeTestProjectRole(addFinalizerProjectRole())

	resObjs := []client.Object{argocdProjectRole}
	subresObjs := []client.Object{argocdProjectRole}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDProjectRoleReconciler(client, scheme)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdProjectRole.Name,
			Namespace: argocdProjectRole.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter < 5*time.Minute {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}

	projectRoleRes := &rbacoperatorv1alpha1.ArgoCDProjectRole{}
	err = reconciler.Get(context.TODO(), req.NamespacedName, projectRoleRes)
	assert.NoError(t, err)
	assert.Equal(t, projectRoleRes, argocdProjectRole)
}

func TestArgoCDProjectRoleReconciler_AddFinalizer(t *testing.T) {
	logf.SetLogger(ZapLogger(true))
	argocdProjectRole := makeTestProjectRole()

	resObjs := []client.Object{argocdProjectRole}
	subresObjs := []client.Object{argocdProjectRole}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDProjectRoleReconciler(client, scheme)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdProjectRole.Name,
			Namespace: argocdProjectRole.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter < time.Second {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}
	projectRoleRes := &rbacoperatorv1alpha1.ArgoCDProjectRole{}
	err = reconciler.Get(context.TODO(), req.NamespacedName, projectRoleRes)
	assert.NoError(t, err)

	assert.Contains(t, projectRoleRes.GetFinalizers(), rbacoperatorv1alpha1.ArgoCDProjectRoleFinalizerName)
}

func TestArgoCDProjectRole_RoleNotFound(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	resObjs := []client.Object{}
	subresObjs := []client.Object{}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDProjectRoleReconciler(client, scheme)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      testProjectRoleName,
			Namespace: testNamespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter > 0 {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}
	assert.Error(t, reconciler.Get(context.TODO(), types.NamespacedName{Name: testProjectRoleName, Namespace: testNamespace}, &rbacoperatorv1alpha1.ArgoCDProjectRole{}))
}

func TestArgoCDProjectRole_HandleFinalizer(t *testing.T) {
	logf.SetLogger(ZapLogger(true))
	argocdProjectRole := makeTestProjectRole(addFinalizerProjectRole(), projectRoleDeletedAt(time.Now()), addProjectRoleBinding(testProjectRoleBindingName))

	resObjs := []client.Object{argocdProjectRole}
	subresObjs := []client.Object{argocdProjectRole}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme, addArgoCDPkgToScheme())
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDProjectRoleReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestProjectRoleBinding()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestAppProject(addTestRoleToAppProject())))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdProjectRole.Name,
			Namespace: argocdProjectRole.Namespace,
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

func TestArgoCDProjectRole_RoleBindingMissing(t *testing.T) {
	logf.SetLogger(ZapLogger(true))
	argocdProjectRole := makeTestProjectRole(addFinalizerProjectRole(), addProjectRoleBinding(testProjectRoleBindingName))

	resObjs := []client.Object{argocdProjectRole}
	subresObjs := []client.Object{argocdProjectRole}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDProjectRoleReconciler(client, scheme)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdProjectRole.Name,
			Namespace: argocdProjectRole.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter < time.Minute*2 {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}

	projectRoleRes := &rbacoperatorv1alpha1.ArgoCDProjectRole{}
	err = reconciler.Get(context.TODO(), req.NamespacedName, projectRoleRes)
	assert.NoError(t, err)
	assert.Equal(t, projectRoleRes.Status.ArgoCDProjectRoleBindingRef, "")
}
