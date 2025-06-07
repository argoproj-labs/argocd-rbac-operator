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

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
)

var _ reconcile.Reconciler = &ArgoCDRoleReconciler{}
var _ reconcile.Reconciler = &ArgoCDRoleBindingReconciler{}

func TestArgoCDRoleReconciler_Reconcile(t *testing.T) {
	logf.SetLogger(ZapLogger(true))
	argocdRole := makeTestRole(addFinalizerRole())

	resObjs := []client.Object{argocdRole}
	subresObjs := []client.Object{argocdRole}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap_WithChangedPolicyCSV()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRole.Name,
			Namespace: argocdRole.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter < 10*time.Minute {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}

	cm := &corev1.ConfigMap{}
	err = reconciler.Get(context.TODO(), types.NamespacedName{Name: testRBACCMName, Namespace: testRBACCMNamespace}, cm)
	assert.NoError(t, err)
	resCM := makeTestCMArgoCDRoleExpected()
	assert.Equal(t, resCM.Data, cm.Data)
}

func TestArgoCDRoleReconciler_AddFinalizer(t *testing.T) {
	logf.SetLogger(ZapLogger(true))
	argocdRole := makeTestRole()

	resObjs := []client.Object{argocdRole}
	subresObjs := []client.Object{argocdRole}

	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRole.Name,
			Namespace: argocdRole.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter > 0 {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}

	argocdRoleRes := &rbacoperatorv1alpha1.ArgoCDRole{}
	err = reconciler.Get(context.TODO(), types.NamespacedName{Name: argocdRole.Name, Namespace: argocdRole.Namespace}, argocdRoleRes)
	assert.NoError(t, err)

	assert.Contains(t, argocdRoleRes.GetFinalizers(), rbacoperatorv1alpha1.ArgoCDRoleFinalizerName)
}

func TestArgoCDRoleReconciler_RoleNotFound(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	resObjs := []client.Object{}
	subresObjs := []client.Object{}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      testRoleName,
			Namespace: testNamespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter > 0 {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}
	assert.Error(t, reconciler.Get(context.TODO(), types.NamespacedName{Name: testRoleName, Namespace: testNamespace}, &rbacoperatorv1alpha1.ArgoCDRole{}))
}

func TestArgoCDRoleReconciler_CMNotFound(t *testing.T) {
	logf.SetLogger(ZapLogger(true))
	argocdRole := makeTestRole(addFinalizerRole())

	resObjs := []client.Object{argocdRole}
	subresObjs := []client.Object{argocdRole}

	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      testRoleName,
			Namespace: testNamespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.Error(t, err)
	assert.True(t, res.RequeueAfter > 0, "expected requeue after to be greater than 0")
}

func TestArgoCDRoleReconciler_HandleFinalizer(t *testing.T) {
	logf.SetLogger(ZapLogger(true))
	argocdRole := makeTestRole(addFinalizerRole(), roleDeletedAt(time.Now()))

	resObjs := []client.Object{argocdRole}
	subresObjs := []client.Object{argocdRole}

	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestCMArgoCDRoleExpected()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRole.Name,
			Namespace: argocdRole.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter > 0 {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}

	cm := &corev1.ConfigMap{}
	err = reconciler.Get(context.TODO(), types.NamespacedName{Name: testRBACCMName, Namespace: testRBACCMNamespace}, cm)
	assert.NoError(t, err)
	wantCM := makeTestRBACConfigMap()
	assert.Equal(t, wantCM.Data, cm.Data)
}

func TestArgoCDRoleReconciler_RoleHasRoleBinding(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	argocdRoleBinding := makeTestRoleBindingWithRoleSubject()
	argocdRole := makeTestRole(addFinalizerRole(), addRoleBinding(argocdRoleBinding.Name))

	resObjs := []client.Object{argocdRole, argocdRoleBinding}
	subresObjs := []client.Object{argocdRole, argocdRoleBinding}

	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap_WithChangedPolicyCSV()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRole.Name,
			Namespace: argocdRole.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter < 10*time.Minute {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}

	cm := &corev1.ConfigMap{}
	err = reconciler.Get(context.TODO(), types.NamespacedName{Name: testRBACCMName, Namespace: testRBACCMNamespace}, cm)
	assert.NoError(t, err)
	resCM := makeTestCM_ArgoCDRole_WithRoleBindingRoleSubject_Expected()
	assert.Equal(t, resCM.Data, cm.Data)
}

func TestArgoCDRoleReconciler_RoleBindingObjectMissing(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	argocdRole := makeTestRole(addFinalizerRole(), addRoleBinding("rb-role-test"))

	resObjs := []client.Object{argocdRole}
	subresObjs := []client.Object{argocdRole}

	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRole.Name,
			Namespace: argocdRole.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.Error(t, err)
	if res.RequeueAfter > 0 {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}
}
