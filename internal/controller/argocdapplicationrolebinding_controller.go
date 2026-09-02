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
	"reflect"
	"slices"
	"time"

	argocdv1alpha "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
)

const generatedApplicationRolePrefix = "application-"

// ArgoCDApplicationRoleBindingReconciler generates an AppProject role for one
// permission set, set of applications, and set of groups.
type ArgoCDApplicationRoleBindingReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdapplicationrolebindings,verbs=*
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdapplicationrolebindings/status,verbs=*
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdapplicationrolebindings/finalizers,verbs=*
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdapplicationroles,verbs=get;list;watch
// +kubebuilder:rbac:groups=argoproj.io,resources=appprojects,verbs=get;list;patch;watch

// Reconcile ensures the binding has one independently managed role in its AppProject.
func (r *ArgoCDApplicationRoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("argocdapplicationrolebinding", req.NamespacedName)

	binding := &rbacoperatorv1alpha1.ArgoCDApplicationRoleBinding{}
	if err := r.Get(ctx, req.NamespacedName, binding); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get ArgoCDApplicationRoleBinding: %w", err)
	}
	statusBefore := binding.DeepCopy().Status

	generatedRoleName := generatedApplicationRoleName(binding.Name)
	if binding.Status.GeneratedRoleName != "" {
		generatedRoleName = binding.Status.GeneratedRoleName
	}

	if binding.IsBeingDeleted() {
		if !binding.HasFinalizer(rbacoperatorv1alpha1.ArgoCDApplicationRoleBindingFinalizerName) {
			return ctrl.Result{}, nil
		}
		if err := r.removeBoundRole(ctx, binding, generatedRoleName); err != nil {
			return ctrl.Result{}, fmt.Errorf("remove generated role during finalization: %w", err)
		}
		binding.RemoveFinalizer(rbacoperatorv1alpha1.ArgoCDApplicationRoleBindingFinalizerName)
		if err := r.Update(ctx, binding); err != nil {
			return ctrl.Result{}, fmt.Errorf("remove finalizer: %w", err)
		}
		return ctrl.Result{}, nil
	}

	if !binding.HasFinalizer(rbacoperatorv1alpha1.ArgoCDApplicationRoleBindingFinalizerName) {
		binding.AddFinalizer(rbacoperatorv1alpha1.ArgoCDApplicationRoleBindingFinalizerName)
		if err := r.Update(ctx, binding); err != nil {
			return ctrl.Result{}, fmt.Errorf("add finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if binding.Status.AppProjectBound != "" && binding.Status.AppProjectBound != binding.Spec.AppProjectRef.Name {
		if err := r.removeRoleFromProject(ctx, binding.Namespace, binding.Status.AppProjectBound, generatedRoleName); err != nil {
			return ctrl.Result{}, fmt.Errorf("remove generated role from previous AppProject: %w", err)
		}
		binding.Status.AppProjectBound = ""
	}

	applicationRole := &rbacoperatorv1alpha1.ArgoCDApplicationRole{}
	roleKey := types.NamespacedName{Name: binding.Spec.ArgoCDApplicationRoleRef.Name, Namespace: binding.Namespace}
	if err := r.Get(ctx, roleKey, applicationRole); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, r.reconcileError(ctx, binding, statusBefore, fmt.Errorf("get ArgoCDApplicationRole: %w", err))
		}

		if cleanupErr := r.removeRoleFromProject(ctx, binding.Namespace, binding.Spec.AppProjectRef.Name, generatedRoleName); cleanupErr != nil {
			return ctrl.Result{}, r.reconcileError(ctx, binding, statusBefore, fmt.Errorf("remove access for missing ArgoCDApplicationRole: %w", cleanupErr))
		}
		binding.Status.AppProjectBound = ""
		binding.Status.GeneratedRoleName = generatedRoleName
		pendingErr := fmt.Errorf("ArgoCDApplicationRole %s not found", roleKey.Name)
		binding.SetConditions(rbacoperatorv1alpha1.ReconcileError(pendingErr).WithObservedGeneration(binding.Generation))
		if err := r.updateBindingStatus(ctx, binding, statusBefore); err != nil {
			return ctrl.Result{}, fmt.Errorf("update unsynced binding status: %w", err)
		}
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	appProject := &argocdv1alpha.AppProject{}
	projectKey := types.NamespacedName{Name: binding.Spec.AppProjectRef.Name, Namespace: binding.Namespace}
	if err := r.Get(ctx, projectKey, appProject); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, r.reconcileError(ctx, binding, statusBefore, fmt.Errorf("get AppProject: %w", err))
		}

		binding.Status.AppProjectBound = ""
		binding.Status.GeneratedRoleName = generatedRoleName
		pendingErr := fmt.Errorf("AppProject %s not found", projectKey.Name)
		binding.SetConditions(rbacoperatorv1alpha1.ReconcileError(pendingErr).WithObservedGeneration(binding.Generation))
		if err := r.updateBindingStatus(ctx, binding, statusBefore); err != nil {
			return ctrl.Result{}, fmt.Errorf("update unsynced binding status: %w", err)
		}
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	desiredRole := buildApplicationProjectRole(applicationRole, binding, appProject.Name, generatedRoleName)
	if err := r.patchApplicationProjectRole(ctx, appProject, desiredRole); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, r.reconcileError(ctx, binding, statusBefore, fmt.Errorf("patch AppProject: %w", err))
	}

	binding.Status.AppProjectBound = appProject.Name
	binding.Status.GeneratedRoleName = generatedRoleName
	binding.SetConditions(rbacoperatorv1alpha1.ReconcileSuccess().WithObservedGeneration(binding.Generation))
	if err := r.updateBindingStatus(ctx, binding, statusBefore); err != nil {
		return ctrl.Result{}, fmt.Errorf("update successful binding status: %w", err)
	}

	log.Info("Reconciled per-application access", "appProject", appProject.Name, "generatedRole", generatedRoleName)
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func generatedApplicationRoleName(bindingName string) string {
	return generatedApplicationRolePrefix + bindingName
}

func buildApplicationProjectRole(
	applicationRole *rbacoperatorv1alpha1.ArgoCDApplicationRole,
	binding *rbacoperatorv1alpha1.ArgoCDApplicationRoleBinding,
	appProjectName string,
	generatedRoleName string,
) *argocdv1alpha.ProjectRole {
	groups := append([]string(nil), binding.Spec.Groups...)
	slices.Sort(groups)
	groups = slices.Compact(groups)

	policySet := map[string]struct{}{}
	for _, rule := range applicationRole.Spec.Rules {
		effect := rule.Effect
		if effect == "" {
			effect = "allow"
		}
		for _, verb := range rule.Verbs {
			for _, application := range binding.Spec.Applications {
				object := fmt.Sprintf("%s/%s", appProjectName, application.Name)
				if application.Namespace != "" {
					object = fmt.Sprintf("%s/%s/%s", appProjectName, application.Namespace, application.Name)
				}
				policy := fmt.Sprintf(
					"p, proj:%s:%s, %s, %s, %s, %s",
					appProjectName,
					generatedRoleName,
					rule.Resource,
					verb,
					object,
					effect,
				)
				policySet[policy] = struct{}{}
			}
		}
	}

	policies := make([]string, 0, len(policySet))
	for policy := range policySet {
		policies = append(policies, policy)
	}
	slices.Sort(policies)

	return &argocdv1alpha.ProjectRole{
		Name:        generatedRoleName,
		Description: applicationRole.Spec.Description,
		Groups:      groups,
		Policies:    policies,
	}
}

func (r *ArgoCDApplicationRoleBindingReconciler) patchApplicationProjectRole(
	ctx context.Context,
	appProject *argocdv1alpha.AppProject,
	desiredRole *argocdv1alpha.ProjectRole,
) error {
	existingRole, index := getRoleInAppProject(appProject, desiredRole.Name)
	if existingRole != nil && areProjectRolesEqual(existingRole, desiredRole) {
		return nil
	}

	original := appProject.DeepCopy()
	if existingRole == nil {
		appProject.Spec.Roles = append(appProject.Spec.Roles, *desiredRole)
	} else {
		appProject.Spec.Roles[index] = *desiredRole
	}
	return r.Patch(ctx, appProject, client.MergeFromWithOptions(original, client.MergeFromWithOptimisticLock{}))
}

func (r *ArgoCDApplicationRoleBindingReconciler) removeBoundRole(
	ctx context.Context,
	binding *rbacoperatorv1alpha1.ArgoCDApplicationRoleBinding,
	generatedRoleName string,
) error {
	projectNames := []string{binding.Spec.AppProjectRef.Name}
	if binding.Status.AppProjectBound != "" && binding.Status.AppProjectBound != binding.Spec.AppProjectRef.Name {
		projectNames = append(projectNames, binding.Status.AppProjectBound)
	}
	for _, projectName := range projectNames {
		if err := r.removeRoleFromProject(ctx, binding.Namespace, projectName, generatedRoleName); err != nil {
			return err
		}
	}
	return nil
}

func (r *ArgoCDApplicationRoleBindingReconciler) removeRoleFromProject(
	ctx context.Context,
	namespace string,
	appProjectName string,
	generatedRoleName string,
) error {
	appProject := &argocdv1alpha.AppProject{}
	if err := r.Get(ctx, types.NamespacedName{Name: appProjectName, Namespace: namespace}, appProject); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	_, index := getRoleInAppProject(appProject, generatedRoleName)
	if index == -1 {
		return nil
	}
	original := appProject.DeepCopy()
	appProject.Spec.Roles = append(appProject.Spec.Roles[:index], appProject.Spec.Roles[index+1:]...)
	return r.Patch(ctx, appProject, client.MergeFromWithOptions(original, client.MergeFromWithOptimisticLock{}))
}

func (r *ArgoCDApplicationRoleBindingReconciler) reconcileError(
	ctx context.Context,
	binding *rbacoperatorv1alpha1.ArgoCDApplicationRoleBinding,
	statusBefore rbacoperatorv1alpha1.ArgoCDApplicationRoleBindingStatus,
	reconcileErr error,
) error {
	binding.SetConditions(rbacoperatorv1alpha1.ReconcileError(reconcileErr).WithObservedGeneration(binding.Generation))
	if err := r.updateBindingStatus(ctx, binding, statusBefore); err != nil {
		return fmt.Errorf("%w; update error status: %v", reconcileErr, err)
	}
	return reconcileErr
}

func (r *ArgoCDApplicationRoleBindingReconciler) updateBindingStatus(
	ctx context.Context,
	binding *rbacoperatorv1alpha1.ArgoCDApplicationRoleBinding,
	statusBefore rbacoperatorv1alpha1.ArgoCDApplicationRoleBindingStatus,
) error {
	if reflect.DeepEqual(statusBefore, binding.Status) {
		return nil
	}
	return r.Status().Update(ctx, binding)
}

func (r *ArgoCDApplicationRoleBindingReconciler) bindingsForApplicationRole(
	ctx context.Context,
	object client.Object,
) []reconcile.Request {
	bindings := &rbacoperatorv1alpha1.ArgoCDApplicationRoleBindingList{}
	if err := r.List(ctx, bindings, client.InNamespace(object.GetNamespace())); err != nil {
		r.Log.Error(err, "Unable to list application role bindings")
		return nil
	}

	requests := []reconcile.Request{}
	for i := range bindings.Items {
		binding := &bindings.Items[i]
		if binding.Spec.ArgoCDApplicationRoleRef.Name == object.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(binding)})
		}
	}
	return requests
}

func (r *ArgoCDApplicationRoleBindingReconciler) bindingsForAppProject(
	ctx context.Context,
	object client.Object,
) []reconcile.Request {
	bindings := &rbacoperatorv1alpha1.ArgoCDApplicationRoleBindingList{}
	if err := r.List(ctx, bindings, client.InNamespace(object.GetNamespace())); err != nil {
		r.Log.Error(err, "Unable to list application role bindings")
		return nil
	}

	requests := []reconcile.Request{}
	for i := range bindings.Items {
		binding := &bindings.Items[i]
		if binding.Spec.AppProjectRef.Name == object.GetName() || binding.Status.AppProjectBound == object.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(binding)})
		}
	}
	return requests
}

// SetupWithManager sets up the controller with watches for referenced roles and AppProjects.
func (r *ArgoCDApplicationRoleBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rbacoperatorv1alpha1.ArgoCDApplicationRoleBinding{}).
		Watches(
			&rbacoperatorv1alpha1.ArgoCDApplicationRole{},
			handler.EnqueueRequestsFromMapFunc(r.bindingsForApplicationRole),
		).
		Watches(
			&argocdv1alpha.AppProject{},
			handler.EnqueueRequestsFromMapFunc(r.bindingsForAppProject),
		).
		Named("argocdapplicationrolebinding").
		Complete(r)
}
