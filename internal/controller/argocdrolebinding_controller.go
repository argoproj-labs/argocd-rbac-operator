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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
	"github.com/argoproj-labs/argocd-rbac-operator/internal/controller/common"
)

// ArgoCDRoleBindingReconciler reconciles a ArgoCDRoleBinding object
type ArgoCDRoleBindingReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdrolebindings,verbs=*
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdrolebindings/status,verbs=*
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdrolebindings/finalizers,verbs=*
// +kubebuilder:rbac:groups=rbac-operator.argoproj-labs.io,resources=argocdroles,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ArgoCDRoleBinding object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *ArgoCDRoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("argocdrole", req.NamespacedName)

	r.Log.Info("Reconciling ArgoCDRoleBinding %s", req.Name)

	var rb rbacoperatorv1alpha1.ArgoCDRoleBinding
	if err := r.Get(ctx, req.NamespacedName, &rb); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("ArgoCDRoleBinding %s not found.", req.Name)
			return ctrl.Result{}, nil
		}
		rb.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
		if err := r.Client.Status().Update(ctx, &rb); err != nil {
			r.Log.Error(err, "Failed to update ArgoCDRoleBinding %s status", req.Name)
		}
		return ctrl.Result{}, err
	}

	if rb.IsBeingDeleted() {
		if err := r.handleFinalizer(ctx, &rb); err != nil {
			rb.SetConditions(rbacoperatorv1alpha1.Deleting().WithMessage(err.Error()))
			if err := r.Client.Status().Update(ctx, &rb); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDRoleBinding %s status", req.Name)
			}
			return ctrl.Result{}, fmt.Errorf("error when handling finalizer: %v", err)
		}
		return ctrl.Result{}, nil
	}

	if !rb.HasFinalizer(rbacoperatorv1alpha1.ArgoCDRoleBindingFinalizerName) {
		if err := r.addFinalizer(ctx, &rb); err != nil {
			rb.SetConditions(rbacoperatorv1alpha1.Deleting().WithMessage(err.Error()))
			if err := r.Client.Status().Update(ctx, &rb); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDRoleBinding %s status", req.Name)
			}
			return ctrl.Result{}, fmt.Errorf("error when adding finalizer: %v", err)
		}
		return ctrl.Result{}, nil
	}

	cm := newConfigMap()

	r.Log.Info("Checking if ConfigMap exists")
	if !IsObjectFound(r.Client, cm.Namespace, cm.Name, cm) {
		rb.SetConditions(rbacoperatorv1alpha1.Pending(fmt.Errorf("ConfigMap %s not found", cm.Name)))
		if err := r.Client.Status().Update(ctx, &rb); err != nil {
			r.Log.Error(err, "Failed to update ArgoCDRoleBinding %s status", req.Name)
		}
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, fmt.Errorf("ConfigMap not found")
	}

	roleName := rb.Spec.ArgoCDRoleRef.Name
	if roleName != common.ArgoCDRoleAdmin && roleName != common.ArgoCDRoleReadOnly {
		var role rbacoperatorv1alpha1.ArgoCDRole

		typeNamespacedNameRole := types.NamespacedName{
			Name:      roleName,
			Namespace: req.Namespace,
		}

		if err := r.Get(ctx, typeNamespacedNameRole, &role); err != nil {
			if errors.IsNotFound(err) {
				r.Log.Info("ArgoCDRole %s not found.", roleName)
				return ctrl.Result{}, nil
			}
			rb.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
			if err := r.Client.Status().Update(ctx, &rb); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDRoleBinding %s status", req.Name)
			}
			return ctrl.Result{}, err
		}

		r.Log.Info("Reconciling RBAC ConfigMap")
		if err := r.reconcileRBACConfigMap(ctx, cm, &rb, &role); err != nil {
			rb.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
			if err := r.Client.Status().Update(ctx, &rb); err != nil {
				r.Log.Error(err, "Failed to update ArgoCDRoleBinding %s status", req.Name)
			}
			return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, err
		}

		rb.SetConditions(rbacoperatorv1alpha1.ReconcileSuccess().WithObservedGeneration(rb.GetGeneration()))
		if err := r.Client.Status().Update(ctx, &rb); err != nil {
			r.Log.Error(err, "Failed to update ArgoCDRoleBinding %s status", req.Name)
		}
		return ctrl.Result{RequeueAfter: time.Minute * 10}, nil

	}

	role := &rbacoperatorv1alpha1.ArgoCDRole{}

	switch roleName {
	case "admin":
		role = createBuiltInAdminRole()
	case "readonly":
		role = createBuiltInReadOnlyRole()
	}

	r.Log.Info("Reconciling RBAC ConfigMap")
	if err := r.reconcileRBACConfigMapForBuiltInRole(ctx, cm, &rb, role); err != nil {
		rb.SetConditions(rbacoperatorv1alpha1.ReconcileError(err))
		if err := r.Client.Status().Update(ctx, &rb); err != nil {
			r.Log.Error(err, "Failed to update ArgoCDRoleBinding %s status", req.Name)
		}
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, err
	}

	rb.SetConditions(rbacoperatorv1alpha1.ReconcileSuccess().WithObservedGeneration(rb.GetGeneration()))
	if err := r.Client.Status().Update(ctx, &rb); err != nil {
		r.Log.Error(err, "Failed to update ArgoCDRoleBinding %s status", req.Name)
	}
	return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArgoCDRoleBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rbacoperatorv1alpha1.ArgoCDRoleBinding{}).
		Watches(
			&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.mapConfigMapToBindings),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		Watches(
			&rbacoperatorv1alpha1.ArgoCDRole{},
			handler.EnqueueRequestsFromMapFunc(r.mapArgoCDRoleToBindings),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		Complete(r)
}

// mapConfigMapToBindings finds ArgoCDRoleBindings that should be reconciled when the ConfigMap changes.
func (r *ArgoCDRoleBindingReconciler) mapConfigMapToBindings(ctx context.Context, obj client.Object) []reconcile.Request {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		r.Log.Error(fmt.Errorf("expected ConfigMap but got %T", obj), "can't reconcile config map changes")
		return nil
	}

	if cm.Namespace != common.ArgoCDRBACConfigMapNamespace || cm.Name != common.ArgoCDRBACConfigMapName {
		return nil
	}

	bindings := &rbacoperatorv1alpha1.ArgoCDRoleBindingList{}
	if err := r.Client.List(ctx, bindings); err != nil {
		r.Log.Error(err, "failed to list ArgoCDRoleBindings for ConfigMap %s", cm.Name)
		return nil
	}

	reqs := make([]reconcile.Request, 0, len(bindings.Items))
	for _, bind := range bindings.Items {
		reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKey{
			Namespace: bind.Namespace,
			Name:      bind.Name,
		}})
	}
	return reqs
}

// mapArgoCDRoleToBindings finds ArgoCDRoleBindings that reference the given ArgoCDRole.
func (r *ArgoCDRoleBindingReconciler) mapArgoCDRoleToBindings(ctx context.Context, obj client.Object) []reconcile.Request {
	role, ok := obj.(*rbacoperatorv1alpha1.ArgoCDRole)
	if !ok {
		r.Log.Error(fmt.Errorf("expected ArgoCDRole but got %T", obj), "can't reconcile argocd role changes")
		return nil
	}

	bindings := &rbacoperatorv1alpha1.ArgoCDRoleBindingList{}
	if err := r.Client.List(ctx, bindings, &client.ListOptions{Namespace: role.Namespace}); err != nil {
		r.Log.Error(err, "failed to list ArgoCDRoleBinding")
		return nil
	}

	reqs := make([]reconcile.Request, 0)
	for _, bind := range bindings.Items {
		if bind.Spec.ArgoCDRoleRef.Name == role.Name {
			reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKey{
				Namespace: bind.Namespace,
				Name:      bind.Name,
			}})
		}
	}
	return reqs
}
