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
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
)

var _ reconcile.Reconciler = &ArgoCDApplicationRoleBindingReconciler{}

const testGeneratedApplicationRoleName = "application-payments-deployers"

func TestBuildApplicationProjectRole(t *testing.T) {
	role := testApplicationRole()
	role.Spec.Rules = append(role.Spec.Rules, rbacoperatorv1alpha1.ApplicationRule{
		Resource: "exec",
		Verbs:    []string{"create"},
		Effect:   "deny",
	})
	binding := testApplicationRoleBinding("payments-deployers", "payments-*", "team-payments", "team-payments")
	binding.Spec.Applications = append(binding.Spec.Applications, rbacoperatorv1alpha1.ApplicationTarget{
		Name:      "worker-*",
		Namespace: "payments",
	})

	got := buildApplicationProjectRole(role, binding, "platform", testGeneratedApplicationRoleName)

	assert.Equal(t, testGeneratedApplicationRoleName, got.Name)
	assert.Equal(t, "Deploy and inspect applications", got.Description)
	assert.Equal(t, []string{"team-payments"}, got.Groups)
	assert.Equal(t, []string{
		"p, proj:platform:application-payments-deployers, applications, get, platform/payments-*, allow",
		"p, proj:platform:application-payments-deployers, applications, get, platform/payments/worker-*, allow",
		"p, proj:platform:application-payments-deployers, applications, sync, platform/payments-*, allow",
		"p, proj:platform:application-payments-deployers, applications, sync, platform/payments/worker-*, allow",
		"p, proj:platform:application-payments-deployers, exec, create, platform/payments-*, deny",
		"p, proj:platform:application-payments-deployers, exec, create, platform/payments/worker-*, deny",
	}, got.Policies)
}

func TestArgoCDApplicationRoleBindingReconcilerCreatesIndependentTeamRoles(t *testing.T) {
	ctx := context.Background()
	role := testApplicationRole()
	paymentsBinding := testApplicationRoleBinding("payments-deployers", "payments-*", "team-payments")
	checkoutBinding := testApplicationRoleBinding("checkout-deployers", "checkout-*", "team-checkout")
	appProject := testApplicationAppProject("platform")

	reconciler := testApplicationRoleBindingReconciler(
		t,
		[]client.Object{role, paymentsBinding, checkoutBinding, appProject},
		[]client.Object{paymentsBinding, checkoutBinding},
	)

	_, err := reconciler.Reconcile(ctx, requestFor(paymentsBinding))
	require.NoError(t, err)
	_, err = reconciler.Reconcile(ctx, requestFor(checkoutBinding))
	require.NoError(t, err)

	gotProject := &argocdv1alpha.AppProject{}
	require.NoError(t, reconciler.Get(ctx, types.NamespacedName{Name: "platform", Namespace: testNamespace}, gotProject))
	require.Len(t, gotProject.Spec.Roles, 2)

	paymentsRole, _ := getRoleInAppProject(gotProject, testGeneratedApplicationRoleName)
	require.NotNil(t, paymentsRole)
	assert.Equal(t, []string{"team-payments"}, paymentsRole.Groups)
	assert.Equal(t, []string{
		"p, proj:platform:application-payments-deployers, applications, get, platform/payments-*, allow",
		"p, proj:platform:application-payments-deployers, applications, sync, platform/payments-*, allow",
	}, paymentsRole.Policies)

	checkoutRole, _ := getRoleInAppProject(gotProject, "application-checkout-deployers")
	require.NotNil(t, checkoutRole)
	assert.Equal(t, []string{"team-checkout"}, checkoutRole.Groups)
	assert.Equal(t, []string{
		"p, proj:platform:application-checkout-deployers, applications, get, platform/checkout-*, allow",
		"p, proj:platform:application-checkout-deployers, applications, sync, platform/checkout-*, allow",
	}, checkoutRole.Policies)

	managedRole := &rbacoperatorv1alpha1.ArgoCDApplicationRole{}
	require.NoError(t, reconciler.Get(ctx, client.ObjectKeyFromObject(role), managedRole))
	managedRole.Spec.Rules = append(managedRole.Spec.Rules, rbacoperatorv1alpha1.ApplicationRule{
		Resource: "logs",
		Verbs:    []string{"get"},
	})
	require.NoError(t, reconciler.Update(ctx, managedRole))

	requests := reconciler.bindingsForApplicationRole(ctx, managedRole)
	require.Len(t, requests, 2)
	for _, request := range requests {
		_, err = reconciler.Reconcile(ctx, request)
		require.NoError(t, err)
	}

	require.NoError(t, reconciler.Get(ctx, types.NamespacedName{Name: "platform", Namespace: testNamespace}, gotProject))
	paymentsRole, _ = getRoleInAppProject(gotProject, "application-payments-deployers")
	checkoutRole, _ = getRoleInAppProject(gotProject, "application-checkout-deployers")
	assert.Contains(t, paymentsRole.Policies,
		"p, proj:platform:application-payments-deployers, logs, get, platform/payments-*, allow")
	assert.Contains(t, checkoutRole.Policies,
		"p, proj:platform:application-checkout-deployers, logs, get, platform/checkout-*, allow")
}

func TestArgoCDApplicationRoleBindingReconcilerAddsFinalizer(t *testing.T) {
	ctx := context.Background()
	binding := testApplicationRoleBinding("payments-deployers", "payments-*", "team-payments")
	binding.Finalizers = nil
	reconciler := testApplicationRoleBindingReconciler(t, []client.Object{binding}, []client.Object{binding})

	result, err := reconciler.Reconcile(ctx, requestFor(binding))
	require.NoError(t, err)
	assert.True(t, result.Requeue)

	gotBinding := &rbacoperatorv1alpha1.ArgoCDApplicationRoleBinding{}
	require.NoError(t, reconciler.Get(ctx, client.ObjectKeyFromObject(binding), gotBinding))
	assert.Contains(t, gotBinding.Finalizers, rbacoperatorv1alpha1.ArgoCDApplicationRoleBindingFinalizerName)
}

func TestArgoCDApplicationRoleBindingReconcilerMovesGeneratedRole(t *testing.T) {
	ctx := context.Background()
	role := testApplicationRole()
	binding := testApplicationRoleBinding("payments-deployers", "payments-*", "team-payments")
	binding.Spec.AppProjectRef.Name = "new-platform"
	binding.Status.AppProjectBound = "old-platform"
	binding.Status.GeneratedRoleName = testGeneratedApplicationRoleName
	oldProject := testApplicationAppProject("old-platform")
	oldProject.Spec.Roles = append(oldProject.Spec.Roles, argocdv1alpha.ProjectRole{Name: binding.Status.GeneratedRoleName})
	newProject := testApplicationAppProject("new-platform")

	reconciler := testApplicationRoleBindingReconciler(
		t,
		[]client.Object{role, binding, oldProject, newProject},
		[]client.Object{binding},
	)

	_, err := reconciler.Reconcile(ctx, requestFor(binding))
	require.NoError(t, err)

	gotOldProject := &argocdv1alpha.AppProject{}
	require.NoError(t, reconciler.Get(ctx, client.ObjectKeyFromObject(oldProject), gotOldProject))
	assert.Empty(t, gotOldProject.Spec.Roles)
	gotNewProject := &argocdv1alpha.AppProject{}
	require.NoError(t, reconciler.Get(ctx, client.ObjectKeyFromObject(newProject), gotNewProject))
	generatedRole, _ := getRoleInAppProject(gotNewProject, binding.Status.GeneratedRoleName)
	assert.NotNil(t, generatedRole)

	gotBinding := &rbacoperatorv1alpha1.ArgoCDApplicationRoleBinding{}
	require.NoError(t, reconciler.Get(ctx, client.ObjectKeyFromObject(binding), gotBinding))
	assert.Equal(t, "new-platform", gotBinding.Status.AppProjectBound)
}

func TestArgoCDApplicationRoleBindingReconcilerRevokesAccessWhenRoleIsMissing(t *testing.T) {
	ctx := context.Background()
	binding := testApplicationRoleBinding("payments-deployers", "payments-*", "team-payments")
	binding.Status.AppProjectBound = "platform"
	binding.Status.GeneratedRoleName = testGeneratedApplicationRoleName
	appProject := testApplicationAppProject("platform")
	appProject.Spec.Roles = append(appProject.Spec.Roles, argocdv1alpha.ProjectRole{
		Name:     binding.Status.GeneratedRoleName,
		Groups:   []string{"team-payments"},
		Policies: []string{"p, proj:platform:application-payments-deployers, applications, sync, platform/payments-*, allow"},
	})

	reconciler := testApplicationRoleBindingReconciler(
		t,
		[]client.Object{binding, appProject},
		[]client.Object{binding},
	)

	result, err := reconciler.Reconcile(ctx, requestFor(binding))
	require.NoError(t, err)
	assert.Equal(t, time.Minute, result.RequeueAfter)

	gotProject := &argocdv1alpha.AppProject{}
	require.NoError(t, reconciler.Get(ctx, client.ObjectKeyFromObject(appProject), gotProject))
	assert.Empty(t, gotProject.Spec.Roles)

	gotBinding := &rbacoperatorv1alpha1.ArgoCDApplicationRoleBinding{}
	require.NoError(t, reconciler.Get(ctx, client.ObjectKeyFromObject(binding), gotBinding))
	assert.Empty(t, gotBinding.Status.AppProjectBound)
	require.Len(t, gotBinding.Status.Conditions, 1)
	assert.Equal(t, rbacoperatorv1alpha1.TypeSynced, gotBinding.Status.Conditions[0].Type)
	assert.Equal(t, corev1.ConditionFalse, gotBinding.Status.Conditions[0].Status)

	require.NoError(t, reconciler.Create(ctx, testApplicationRole()))
	_, err = reconciler.Reconcile(ctx, requestFor(binding))
	require.NoError(t, err)
	require.NoError(t, reconciler.Get(ctx, client.ObjectKeyFromObject(binding), gotBinding))
	require.Len(t, gotBinding.Status.Conditions, 1)
	assert.Equal(t, corev1.ConditionTrue, gotBinding.Status.Conditions[0].Status)
	require.NoError(t, reconciler.Get(ctx, client.ObjectKeyFromObject(appProject), gotProject))
	generatedRole, _ := getRoleInAppProject(gotProject, testGeneratedApplicationRoleName)
	assert.NotNil(t, generatedRole)
}

func TestArgoCDApplicationRoleBindingReconcilerFinalizerRemovesGeneratedRole(t *testing.T) {
	ctx := context.Background()
	now := metav1.NewTime(time.Now())
	binding := testApplicationRoleBinding("payments-deployers", "payments-*", "team-payments")
	binding.DeletionTimestamp = &now
	binding.Status.AppProjectBound = "platform"
	binding.Status.GeneratedRoleName = testGeneratedApplicationRoleName
	appProject := testApplicationAppProject("platform")
	appProject.Spec.Roles = append(appProject.Spec.Roles, argocdv1alpha.ProjectRole{Name: binding.Status.GeneratedRoleName})

	reconciler := testApplicationRoleBindingReconciler(
		t,
		[]client.Object{binding, appProject},
		[]client.Object{binding},
	)

	_, err := reconciler.Reconcile(ctx, requestFor(binding))
	require.NoError(t, err)

	gotProject := &argocdv1alpha.AppProject{}
	require.NoError(t, reconciler.Get(ctx, client.ObjectKeyFromObject(appProject), gotProject))
	assert.Empty(t, gotProject.Spec.Roles)
}

func testApplicationRole() *rbacoperatorv1alpha1.ArgoCDApplicationRole {
	return &rbacoperatorv1alpha1.ArgoCDApplicationRole{
		ObjectMeta: metav1.ObjectMeta{Name: "deployer", Namespace: testNamespace},
		Spec: rbacoperatorv1alpha1.ArgoCDApplicationRoleSpec{
			Description: "Deploy and inspect applications",
			Rules: []rbacoperatorv1alpha1.ApplicationRule{
				{Resource: "applications", Verbs: []string{"sync", "get"}},
			},
		},
	}
}

func testApplicationRoleBinding(name, application string, groups ...string) *rbacoperatorv1alpha1.ArgoCDApplicationRoleBinding {
	return &rbacoperatorv1alpha1.ArgoCDApplicationRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  testNamespace,
			Finalizers: []string{rbacoperatorv1alpha1.ArgoCDApplicationRoleBindingFinalizerName},
		},
		Spec: rbacoperatorv1alpha1.ArgoCDApplicationRoleBindingSpec{
			ArgoCDApplicationRoleRef: rbacoperatorv1alpha1.ArgoCDApplicationRoleRef{Name: "deployer"},
			AppProjectRef:            rbacoperatorv1alpha1.AppProjectReference{Name: "platform"},
			Applications:             []rbacoperatorv1alpha1.ApplicationTarget{{Name: application}},
			Groups:                   groups,
		},
	}
}

func testApplicationAppProject(name string) *argocdv1alpha.AppProject {
	return &argocdv1alpha.AppProject{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
	}
}

func testApplicationRoleBindingReconciler(
	t *testing.T,
	objects []client.Object,
	statusObjects []client.Object,
) *ArgoCDApplicationRoleBindingReconciler {
	t.Helper()
	scheme := makeTestReconcilerScheme(rbacoperatorv1alpha1.AddToScheme, addArgoCDPkgToScheme())
	return &ArgoCDApplicationRoleBindingReconciler{
		Client: makeTestReconcilerClient(scheme, objects, statusObjects),
		Scheme: scheme,
		Log:    ZapLogger(true),
	}
}

func requestFor(object client.Object) reconcile.Request {
	return reconcile.Request{NamespacedName: client.ObjectKeyFromObject(object)}
}
