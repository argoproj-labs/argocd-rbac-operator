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

func TestArgoCDRoleBindingReconciler_ReconcileRoleSubject(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	argocdRoleBinding := makeTestRoleBindingWithRoleSubject(addFinalizerRoleBinding())
	argocdRole := makeTestRole()

	resObjs := []client.Object{argocdRole, argocdRoleBinding}
	subresObjs := []client.Object{argocdRole, argocdRoleBinding}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleBindingReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRoleBinding.Name,
			Namespace: argocdRoleBinding.Namespace,
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

func TestArgoCDRoleBindingReconciler_AddFinalizer(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	argocdRoleBinding := makeTestRoleBindingWithRoleSubject()

	resObjs := []client.Object{argocdRoleBinding}
	subresObjs := []client.Object{argocdRoleBinding}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleBindingReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRoleBinding.Name,
			Namespace: argocdRoleBinding.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter > 0 {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}

	argocdRoleBindingRes := &rbacoperatorv1alpha1.ArgoCDRoleBinding{}
	err = reconciler.Get(context.TODO(), types.NamespacedName{Name: argocdRoleBinding.Name, Namespace: argocdRoleBinding.Namespace}, argocdRoleBindingRes)
	assert.NoError(t, err)

	assert.Contains(t, argocdRoleBindingRes.Finalizers, rbacoperatorv1alpha1.ArgoCDRoleBindingFinalizerName)
}

func TestArgoCDRoleBindingReconciler_RoleBindingNotFound(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	resObjs := []client.Object{}
	subresObjs := []client.Object{}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleBindingReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      testRoleBindingName,
			Namespace: testNamespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter > 0 {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}
	assert.Error(t, reconciler.Get(context.TODO(), types.NamespacedName{Name: testRoleBindingName, Namespace: testNamespace}, &rbacoperatorv1alpha1.ArgoCDRoleBinding{}))
}

func TestArgoCDRoleBindingReconciler_CMNotFound(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	argocdRoleBinding := makeTestRoleBindingWithRoleSubject(addFinalizerRoleBinding())

	resObjs := []client.Object{argocdRoleBinding}
	subresObjs := []client.Object{argocdRoleBinding}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleBindingReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRoleBinding.Name,
			Namespace: argocdRoleBinding.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.Error(t, err)
	assert.True(t, res.RequeueAfter > 0, "expected requeue after to be greater than 0")
}

func TestArgoCDRoleBindingReconciler_HandleFinalizer(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	argocdRoleBinding := makeTestRoleBindingWithRoleSubject(addFinalizerRoleBinding(), roleBindingDeletedAt(time.Now()))
	argocdRole := makeTestRole(addFinalizerRole())

	resObjs := []client.Object{argocdRole, argocdRoleBinding}
	subresObjs := []client.Object{argocdRole, argocdRoleBinding}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleBindingReconciler(client, scheme)
	roleReconciler := makeTestArgoCDRoleReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRoleBinding.Name,
			Namespace: argocdRoleBinding.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter > 0 {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}

	roleReq := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRole.Name,
			Namespace: argocdRole.Namespace,
		},
	}

	roleRes, roleErr := roleReconciler.Reconcile(context.TODO(), roleReq)
	assert.NoError(t, roleErr)
	if roleRes.RequeueAfter < 10*time.Minute {
		t.Fatalf("reconcile requeued request after %s", roleRes.RequeueAfter)
	}

	cm := &corev1.ConfigMap{}
	err = reconciler.Get(context.TODO(), types.NamespacedName{Name: testRBACCMName, Namespace: testRBACCMNamespace}, cm)
	assert.NoError(t, err)
	wantCM := makeTestCMArgoCDRoleExpected()
	assert.Equal(t, wantCM.Data, cm.Data)
}

func TestArgoCDRoleBindingReconciler_RoleNotFound(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	argocdRoleBinding := makeTestRoleBindingWithRoleSubject(addFinalizerRoleBinding())

	resObjs := []client.Object{argocdRoleBinding}
	subresObjs := []client.Object{argocdRoleBinding}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleBindingReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRoleBinding.Name,
			Namespace: argocdRoleBinding.Namespace,
		},
	}

	res, err := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err)
	if res.RequeueAfter > 0 {
		t.Fatalf("reconcile requeued request after %s", res.RequeueAfter)
	}
	assert.Error(t, reconciler.Get(context.TODO(), types.NamespacedName{Name: argocdRoleBinding.Spec.ArgoCDRoleRef.Name, Namespace: testNamespace}, &rbacoperatorv1alpha1.ArgoCDRole{}))
}

func TestArgoCDRoleBindingReconciler_ReconcileSSOSubject(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	argocdRoleBinding := makeTestRoleBindingWithSSOSubject(addFinalizerRoleBinding())
	argocdRole := makeTestRole()

	resObjs := []client.Object{argocdRole, argocdRoleBinding}
	subresObjs := []client.Object{argocdRole, argocdRoleBinding}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleBindingReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRoleBinding.Name,
			Namespace: argocdRoleBinding.Namespace,
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
	resCM := makeTestCM_ArgoCDRole_WithRoleBindingSSOSubject_Expected()
	assert.Equal(t, resCM.Data, cm.Data)
}

func TestArgoCDRoleBindingReconciler_ReconcileLocalSubject(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	argocdRoleBinding := makeTestRoleBindingWithLocalSubject(addFinalizerRoleBinding())
	argocdRole := makeTestRole()

	resObjs := []client.Object{argocdRole, argocdRoleBinding}
	subresObjs := []client.Object{argocdRole, argocdRoleBinding}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleBindingReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap_WithChangedPolicyCSV()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRoleBinding.Name,
			Namespace: argocdRoleBinding.Namespace,
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
	resCM := makeTestCM_ArgoCDRole_WithRoleBindingLocalSubject_Expected()
	assert.Equal(t, resCM.Data, cm.Data)
}

func TestArgoCDRoleBindingReconciler_ReconcileBuiltInAdmin(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	argocdRoleBinding := makeTestRoleBindingForBuiltInAdmin(addFinalizerRoleBinding())

	resObjs := []client.Object{argocdRoleBinding}
	subresObjs := []client.Object{argocdRoleBinding}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleBindingReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap_WithChangedPolicyCSV()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRoleBinding.Name,
			Namespace: argocdRoleBinding.Namespace,
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
	resCM := makeTestCM_BuiltInAdmin_WithRoleBinding_Expected()
	assert.Equal(t, resCM.Data, cm.Data)
}

func TestArgoCDRoleBindingReconciler_ReconcileBuiltInReadOnly(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	argocdRoleBinding := makeTestRoleBindingForBuiltInReadOnly(addFinalizerRoleBinding())

	resObjs := []client.Object{argocdRoleBinding}
	subresObjs := []client.Object{argocdRoleBinding}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleBindingReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap_WithChangedPolicyCSV()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRoleBinding.Name,
			Namespace: argocdRoleBinding.Namespace,
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
	resCM := makeTestCM_BuiltInReadOnly_WithRoleBinding_Expected()
	assert.Equal(t, resCM.Data, cm.Data)
}

func TestArgoCDRoleBindingReconciler_HandleFinalizerBuiltInRole(t *testing.T) {
	logf.SetLogger(ZapLogger(true))

	argocdRoleBinding := makeTestRoleBindingForBuiltInReadOnly(addFinalizerRoleBinding(), roleBindingDeletedAt(time.Now()))

	resObjs := []client.Object{argocdRoleBinding}
	subresObjs := []client.Object{argocdRoleBinding}
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme)
	client := makeTestReconcilerClient(scheme, resObjs, subresObjs)
	reconciler := makeTestArgoCDRoleBindingReconciler(client, scheme)

	assert.NoError(t, reconciler.Create(context.TODO(), makeTestArgoCDNamespace()))
	assert.NoError(t, reconciler.Create(context.TODO(), makeTestRBACConfigMap()))

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      argocdRoleBinding.Name,
			Namespace: argocdRoleBinding.Namespace,
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
	resCM := makeTestRBACConfigMap()
	assert.Equal(t, resCM.Data, cm.Data)
}
