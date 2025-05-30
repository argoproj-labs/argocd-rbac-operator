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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
)

// ArgoCDProjectRoleReconciler reconciles a ArgoCDProjectRole object
type ArgoCDProjectRoleReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdprojectroles,verbs=*
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdprojectroles/status,verbs=*
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdprojectroles/finalizers,verbs=*

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ArgoCDProjectRole object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *ArgoCDProjectRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("argocdprojectrole", req.NamespacedName)

	r.Log.Info("Reconciling ArgoCDProjectRole", "name", req.Name, "namespace", req.Namespace)

	projectRole := rbacoperatorv1alpha1.ArgoCDProjectRole{}
	if err := r.Get(ctx, req.NamespacedName, &projectRole); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("ArgoCDProjectRole not found, skipping reconcile", "name", req.Name)
			return ctrl.Result{}, nil
		}
		projectRole.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
		if err := r.Status().Update(ctx, &projectRole); err != nil {
			r.Log.Error(err, "Failed to update ArgoCDProjectRole status", "name", req.Name)
			return ctrl.Result{}, err
		}
	}

	if projectRole.IsBeingDeleted() {
		if err := r.handleFinalizer(ctx, &projectRole); err != nil {
			projectRole.SetConditions(rbacoperatorv1alpha1.Deleting())
			if err := r.Status().Update(ctx, &projectRole); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDProjectRole status during finalizer handling", "name", req.Name)
			}
			return ctrl.Result{}, fmt.Errorf("error when handling finalizer: %v", err)
		}
		return ctrl.Result{}, nil
	}

	if !projectRole.HasFinalizer(rbacoperatorv1alpha1.ArgoCDProjectRoleFinalizerName) {
		if err := r.addFinalizer(ctx, &projectRole); err != nil {
			projectRole.SetConditions(rbacoperatorv1alpha1.Deleting().WithMessage(err.Error()))
			if err := r.Status().Update(ctx, &projectRole); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDProjectRole status after adding finalizer", "name", req.Name)
			}
			return ctrl.Result{}, fmt.Errorf("error when adding finalizer: %v", err)
		}
		return ctrl.Result{}, nil
	}

	if projectRole.HasArgoCDProjectRoleBindingRef() {
		projectRb := rbacoperatorv1alpha1.ArgoCDProjectRoleBinding{}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArgoCDProjectRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rbacoperatorv1alpha1.ArgoCDProjectRole{}).
		Named("argocdprojectrole").
		Complete(r)
}
