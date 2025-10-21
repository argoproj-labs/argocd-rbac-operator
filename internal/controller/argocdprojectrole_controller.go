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
	"time"

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
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdprojectrolebindings,verbs=get;list
// +kubebuilder:rbac:groups=argoproj.io,resources=appprojects,verbs=get;list;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
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
		}
		return ctrl.Result{}, err
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
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	if projectRole.HasArgoCDProjectRoleBindingRef() {
		projectRb := rbacoperatorv1alpha1.ArgoCDProjectRoleBinding{}

		projectRBObjectKey := client.ObjectKey{
			Name:      projectRole.Status.ArgoCDProjectRoleBindingRef,
			Namespace: req.Namespace,
		}

		if err := r.Get(ctx, projectRBObjectKey, &projectRb); err != nil {
			if errors.IsNotFound(err) {
				r.Log.Info("ArgoCDProjectRoleBinding not found", "name", projectRBObjectKey.Name)
				projectRole.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
				projectRole.Status.ArgoCDProjectRoleBindingRef = ""
				if err := r.Status().Update(ctx, &projectRole); err != nil {
					r.Log.Error(err, "Failed to update ArgoCDProjectRole status after binding not found", "name", req.Name)
				}
				return ctrl.Result{RequeueAfter: time.Minute * 2}, nil
			}
			projectRole.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
			if err := r.Status().Update(ctx, &projectRole); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDProjectRole status", "name", req.Name)
			}
			return ctrl.Result{}, fmt.Errorf("error fetching ArgoCDProjectRoleBinding: %v", err)
		}
	}
	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArgoCDProjectRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rbacoperatorv1alpha1.ArgoCDProjectRole{}).
		Named("argocdprojectrole").
		Complete(r)
}
