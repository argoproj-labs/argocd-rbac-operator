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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
)

// blank assignment to verify that RoleReconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &ArgoCDRoleReconciler{}

// ArgoCDRoleReconciler reconciles a Role object
type ArgoCDRoleReconciler struct {
	client.Client
	Log                          logr.Logger
	Scheme                       *runtime.Scheme
	ArgoCDRBACConfigMapName      string
	ArgoCDRBACConfigMapNamespace string
}

// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdroles,verbs=*
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdroles/status,verbs=*
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdroles/finalizers,verbs=*
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdrolebindings,verbs=get;list
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ArgoCDRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("argocdrole", req.NamespacedName)

	r.Log.Info("Reconciling ArgoCDRole", "name", req.Name)

	var role rbacoperatorv1alpha1.ArgoCDRole
	if err := r.Get(ctx, req.NamespacedName, &role); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("ArgoCDRole not found.", "name", req.Name)
			return ctrl.Result{}, nil
		}
		role.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
		if err := r.Client.Status().Update(ctx, &role); err != nil {
			r.Log.Error(err, "Failed to update ArgoCDRole status", "name", req.Name)
		}
		return ctrl.Result{}, err
	}

	if role.IsBeingDeleted() {
		if err := r.handleFinalizer(ctx, &role); err != nil {
			if errors.IsConflict(err) {
				r.Log.Info("Conflict while handling finalizer for ArgoCDRole", "name", req.Name)
				return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, nil
			}
			role.SetConditions(rbacoperatorv1alpha1.Deleting().WithMessage(err.Error()))
			if err := r.Client.Status().Update(ctx, &role); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDRole status", "name", req.Name)
			}
			return ctrl.Result{}, fmt.Errorf("error when handling finalizer: %v", err)
		}
		return ctrl.Result{}, nil
	}

	if !role.HasFinalizer(rbacoperatorv1alpha1.ArgoCDRoleFinalizerName) {
		if err := r.addFinalizer(ctx, &role); err != nil {
			role.SetConditions(rbacoperatorv1alpha1.Deleting().WithMessage(err.Error()))
			if err := r.Client.Status().Update(ctx, &role); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDRole status", "name", req.Name)
			}
			return ctrl.Result{}, fmt.Errorf("error when adding finalizer: %v", err)
		}
		return ctrl.Result{}, nil
	}

	cm := newConfigMap(r.ArgoCDRBACConfigMapName, r.ArgoCDRBACConfigMapNamespace)

	r.Log.Info("Checking if ConfigMap exists")
	if !IsObjectFound(r.Client, cm.Namespace, cm.Name, cm) {
		role.SetConditions(rbacoperatorv1alpha1.Pending(fmt.Errorf("ConfigMap %s not found", cm.Name)))
		if err := r.Client.Status().Update(ctx, &role); err != nil {
			r.Log.Error(err, "Failed to update ArgoCDRole status", "name", req.Name)
		}
		return ctrl.Result{}, fmt.Errorf("ConfigMap not found")
	}

	if role.HasArgoCDRoleBindingRef() {
		var rb rbacoperatorv1alpha1.ArgoCDRoleBinding

		typeNamespacedNameRoleBinding := types.NamespacedName{
			Name:      role.Status.ArgoCDRoleBindingRef,
			Namespace: req.Namespace,
		}
		if err := r.Get(ctx, typeNamespacedNameRoleBinding, &rb); err != nil {
			if errors.IsNotFound(err) {
				r.Log.Info("ArgoCDRoleBinding not found.", "name", role.Status.ArgoCDRoleBindingRef)
				role.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
				if err := r.Client.Status().Update(ctx, &role); err != nil {
					r.Log.Error(err, "Failed to update ArgoCDRole status", "name", req.Name)
				}
				return ctrl.Result{}, err
			}
		}

		r.Log.Info("Reconciling RBAC ConfigMap")
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Fetch the latest version of the ConfigMap
			cm := newConfigMap(r.ArgoCDRBACConfigMapName, r.ArgoCDRBACConfigMapNamespace)
			if err := r.Get(ctx, client.ObjectKeyFromObject(cm), cm); err != nil {
				return err
			}
			return r.reconcileRBACConfigMapWithRoleBinding(cm, &role, &rb)
		})

		if err != nil {
			role.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
			if updateErr := r.Client.Status().Update(ctx, &role); updateErr != nil {
				r.Log.Error(updateErr, "Failed to update ArgoCDRole status", "name", req.Name)
			}
			return ctrl.Result{}, err
		}

		role.SetConditions(rbacoperatorv1alpha1.ReconcileSuccess().WithObservedGeneration(role.GetGeneration()))
		if err := r.Client.Status().Update(ctx, &role); err != nil {
			r.Log.Error(err, "Failed to update ArgoCDRole status", "name", req.Name)
		}
		return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
	}

	r.Log.Info("Reconciling RBAC ConfigMap")
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of the ConfigMap
		cm := newConfigMap(r.ArgoCDRBACConfigMapName, r.ArgoCDRBACConfigMapNamespace)
		if err := r.Get(ctx, client.ObjectKeyFromObject(cm), cm); err != nil {
			return err
		}

		return r.reconcileRBACConfigMap(cm, &role)
	})

	if err != nil {
		role.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
		if updateErr := r.Client.Status().Update(ctx, &role); updateErr != nil {
			r.Log.Error(updateErr, "Failed to update ArgoCDRole status", "name", req.Name)
		}
		return ctrl.Result{}, err
	}

	role.SetConditions(rbacoperatorv1alpha1.ReconcileSuccess().WithObservedGeneration(role.GetGeneration()))
	if err := r.Client.Status().Update(ctx, &role); err != nil {
		r.Log.Error(err, "Failed to update ArgoCDRole status", "name", req.Name)
	}
	return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArgoCDRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rbacoperatorv1alpha1.ArgoCDRole{}).
		Complete(r)
}
