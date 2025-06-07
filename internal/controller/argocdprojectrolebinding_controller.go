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
	"slices"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
)

// ArgoCDProjectRoleBindingReconciler reconciles a ArgoCDProjectRoleBinding object
type ArgoCDProjectRoleBindingReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdprojectrolebindings,verbs=*
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdprojectrolebindings/status,verbs=*
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdprojectrolebindings/finalizers,verbs=*
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdprojectroles,verbs=get;list
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdprojectroles/status,verbs=get;list;update
// +kubebuilder:rbac:groups=argoproj.io,resources=appprojects,verbs=get;list;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ArgoCDProjectRoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) { //nolint:gocyclo
	_ = r.Log.WithValues("argocdprojectrolebinding", req.NamespacedName)

	r.Log.Info("Reconciling ArgoCDProjectRoleBinding", "name", req.Name, "namespace", req.Namespace)

	projectRoleBinding := rbacoperatorv1alpha1.ArgoCDProjectRoleBinding{}
	if err := r.Get(ctx, req.NamespacedName, &projectRoleBinding); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("ArgoCDProjectRoleBinding not found, skipping reconcile", "name", req.Name)
			return ctrl.Result{}, nil
		}
		projectRoleBinding.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
		if err := r.Status().Update(ctx, &projectRoleBinding); err != nil {
			r.Log.Error(err, "Failed to update ArgoCDProjectRoleBinding status", "name", req.Name)
			return ctrl.Result{}, err
		}
	}

	if projectRoleBinding.IsBeingDeleted() {
		if err := r.handleFinalizer(ctx, &projectRoleBinding); err != nil {
			projectRoleBinding.SetConditions(rbacoperatorv1alpha1.Deleting())
			if err := r.Status().Update(ctx, &projectRoleBinding); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDProjectRoleBinding status during finalizer handling", "name", req.Name)
			}
			return ctrl.Result{}, fmt.Errorf("error when handling finalizer: %v", err)
		}
		return ctrl.Result{}, nil
	}

	if !projectRoleBinding.HasFinalizer(rbacoperatorv1alpha1.ArgoCDProjectRoleBindingFinalizerName) {
		if err := r.addFinalizer(ctx, &projectRoleBinding); err != nil {
			projectRoleBinding.SetConditions(rbacoperatorv1alpha1.Deleting().WithMessage(err.Error()))
			if err := r.Status().Update(ctx, &projectRoleBinding); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDProjectRoleBinding status after adding finalizer", "name", req.Name)
			}
			return ctrl.Result{}, fmt.Errorf("error when adding finalizer: %v", err)
		}
		return ctrl.Result{}, nil
	}

	projectRoleName := projectRoleBinding.Spec.ArgoCDProjectRoleRef.Name
	projectRole := rbacoperatorv1alpha1.ArgoCDProjectRole{}
	projectRoleObjectKey := client.ObjectKey{
		Name:      projectRoleName,
		Namespace: req.Namespace,
	}
	if err := r.Get(ctx, projectRoleObjectKey, &projectRole); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("ArgoCDProjectRole not found, skipping reconcile", "name", projectRoleName)
			projectRoleBinding.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
			if err := r.Status().Update(ctx, &projectRoleBinding); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDProjectRoleBinding status", "name", req.Name)
			}
			return ctrl.Result{}, nil
		}
		projectRoleBinding.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
		if err := r.Status().Update(ctx, &projectRoleBinding); err != nil {
			r.Log.Error(err, "Failed to update ArgoCDProjectRoleBinding status after project role not found", "name", req.Name)
		}
		return ctrl.Result{}, fmt.Errorf("error when getting ArgoCDProjectRole: %v", err)
	}

	if !projectRole.HasArgoCDProjectRoleBindingRef() {
		projectRole.SetArgoCDProjectRoleBindingRef(projectRoleBinding.Name)
		if err := r.Status().Update(ctx, &projectRole); err != nil {
			r.Log.Error(err, "Failed to update ArgoCDProjectRole status with binding reference", "name", projectRole.Name)
		}
	}

	appProjectSubjectSet := makeAppProjectSubjectsSet(projectRoleBinding.Spec.Subjects)
	for _, boundAppProject := range projectRoleBinding.Status.AppProjectsBound {
		if _, exists := appProjectSubjectSet[boundAppProject]; !exists {
			appProject := newAppProject(boundAppProject, req.Namespace)
			if !IsObjectFound(r.Client, appProject.Namespace, appProject.Name, appProject) {
				r.Log.Info("AppProject not found", "name", boundAppProject)
				continue
			}
			r.Log.Info("Removing Role from AppProject", "appProject", boundAppProject, "role", projectRoleName)
			if err := removeRoleFromAppProject(r.Client, appProject, projectRoleName); err != nil {
				if errors.IsConflict(err) {
					r.Log.Info("Conflict while patching AppProject, requeuing", "appProject", appProject.Name)
					return ctrl.Result{RequeueAfter: time.Second}, nil
				}
				r.Log.Error(err, "Failed to remove role from AppProject", "appProject", boundAppProject, "role", projectRoleName)
				projectRoleBinding.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
				return ctrl.Result{}, fmt.Errorf("error when removing role from AppProject: %v", err)
			}
			r.Log.Info("Role removed from AppProject", "appProject", boundAppProject, "role", projectRoleName)
			projectRoleBinding.Status.AppProjectsBound = removeStringFromSlice(projectRoleBinding.Status.AppProjectsBound, boundAppProject)
		}
	}
	if err := r.Status().Update(ctx, &projectRoleBinding); err != nil {
		r.Log.Error(err, "Failed to update ArgoCDProjectRoleBinding status after removing roles from AppProjects", "name", req.Name)
	}

	r.Log.Info("Reconciling AppProjects with ArgoCDProjectRoleBinding", "name", req.Name)

	for appProjectRef, groups := range appProjectSubjectSet {
		appProject := newAppProject(appProjectRef, req.Namespace)
		if !IsObjectFound(r.Client, appProject.Namespace, appProject.Name, appProject) {
			projectRoleBinding.SetConditions(rbacoperatorv1alpha1.Pending(fmt.Errorf("AppProject %s not found", appProjectRef)))
			if err := r.Status().Update(ctx, &projectRoleBinding); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDProjectRoleBinding status", "name", req.Name)
			}
			continue
		}
		r.Log.Info("Reconciling AppProject", "appProject", appProjectRef)
		if err := r.patchAppProject(appProject, &projectRole, &groups); err != nil {
			if errors.IsConflict(err) {
				r.Log.Info("Conflict while patching AppProject, requeuing", "appProject", appProjectRef)
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			projectRoleBinding.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
			if err := r.Status().Update(ctx, &projectRoleBinding); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDProjectRoleBinding status after patching AppProject", "name", req.Name)
			}
			return ctrl.Result{RequeueAfter: time.Second}, fmt.Errorf("error when patching AppProject: %v", err)
		}
		r.Log.Info("AppProject patched successfully", "appProject", appProjectRef)
		if !isAppProjectInStatus(projectRoleBinding.Status.AppProjectsBound, appProjectRef) {
			projectRoleBinding.Status.AppProjectsBound = append(projectRoleBinding.Status.AppProjectsBound, appProjectRef)
			r.Log.Info("AppProject added to status", "appProject", appProjectRef)
			if err := r.Status().Update(ctx, &projectRoleBinding); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDProjectRoleBinding status after patching AppProject", "name", req.Name)
				return ctrl.Result{RequeueAfter: time.Second}, fmt.Errorf("error when updating status: %v", err)
			}
		}
	}
	r.Log.Info("ArgoCDProjectRoleBinding reconciliation completed", "name", req.Name)

	projectRoleBinding.SetConditions(rbacoperatorv1alpha1.ReconcileSuccess().WithObservedGeneration(projectRoleBinding.GetGeneration()))
	if err := r.Status().Update(ctx, &projectRoleBinding); err != nil {
		r.Log.Error(err, "Failed to update ArgoCDProjectRoleBinding status after reconciliation", "name", req.Name)
	}

	return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
}

func makeAppProjectSubjectsSet(appProjectSubjects []rbacoperatorv1alpha1.AppProjectSubject) map[string][]string {
	appProjectSubjectSet := make(map[string][]string, len(appProjectSubjects))
	for _, subject := range appProjectSubjects {
		appProjectSubjectSet[subject.AppProjectRef] = subject.Groups
	}
	return appProjectSubjectSet
}

func removeStringFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func isAppProjectInStatus(appProjects []string, appProject string) bool {
	return slices.Contains(appProjects, appProject)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArgoCDProjectRoleBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rbacoperatorv1alpha1.ArgoCDProjectRoleBinding{}).
		Named("argocdprojectrolebinding").
		Complete(r)
}
