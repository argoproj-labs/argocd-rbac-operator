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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
	"github.com/argoproj-labs/argocd-rbac-operator/internal/controller/common"
)

func (r *ArgoCDRoleReconciler) addFinalizer(ctx context.Context, role *rbacoperatorv1alpha1.ArgoCDRole) error {
	role.AddFinalizer(rbacoperatorv1alpha1.ArgoCDRoleFinalizerName)
	return r.Update(ctx, role)
}

func (r *ArgoCDRoleReconciler) handleFinalizer(ctx context.Context, role *rbacoperatorv1alpha1.ArgoCDRole) error {
	if !role.HasFinalizer(rbacoperatorv1alpha1.ArgoCDRoleFinalizerName) {
		return nil
	}

	if err := r.delete(role); err != nil {
		return err
	}

	role.RemoveFinalizer(rbacoperatorv1alpha1.ArgoCDRoleFinalizerName)
	return r.Update(ctx, role)
}

func (r *ArgoCDRoleReconciler) delete(role *rbacoperatorv1alpha1.ArgoCDRole) error {
	cm := newConfigMap(r)
	overlayKey := fmt.Sprintf("policy.%s.%s.csv", role.Namespace, role.ObjectMeta.Name)
	if IsObjectFound(r.Client, cm.Namespace, cm.Name, cm) {
		delete(cm.Data, overlayKey)
		if err := r.Client.Update(context.TODO(), cm); err != nil {
			return err
		}
	}

	return nil
}

func (r *ArgoCDRoleBindingReconciler) addFinalizer(ctx context.Context, rb *rbacoperatorv1alpha1.ArgoCDRoleBinding) error {
	rb.AddFinalizer(rbacoperatorv1alpha1.ArgoCDRoleBindingFinalizerName)
	return r.Update(ctx, rb)
}

func (r *ArgoCDRoleBindingReconciler) handleFinalizer(ctx context.Context, rb *rbacoperatorv1alpha1.ArgoCDRoleBinding) error {
	if !rb.HasFinalizer(rbacoperatorv1alpha1.ArgoCDRoleBindingFinalizerName) {
		return nil
	}

	if err := r.delete(rb); err != nil {
		return err
	}

	rb.RemoveFinalizer(rbacoperatorv1alpha1.ArgoCDRoleBindingFinalizerName)
	return r.Update(ctx, rb)
}

func (r *ArgoCDRoleBindingReconciler) delete(rb *rbacoperatorv1alpha1.ArgoCDRoleBinding) error {
	roleRefName := rb.Spec.ArgoCDRoleRef.Name
	if roleRefName == common.ArgoCDRoleAdmin || roleRefName == common.ArgoCDRoleReadOnly {
		cm := newConfigMap(r)
		overlayKey := fmt.Sprintf("policy.%s.%s.csv", rb.Namespace, roleRefName)
		if IsObjectFound(r.Client, cm.Namespace, cm.Name, cm) {
			delete(cm.Data, overlayKey)
			if err := r.Client.Update(context.TODO(), cm); err != nil {
				return err
			}
		}
		return nil
	}

	role := &rbacoperatorv1alpha1.ArgoCDRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleRefName,
			Namespace: rb.Namespace,
		},
	}
	if IsObjectFound(r.Client, role.Namespace, role.Name, role) {
		role.Status.ArgoCDRoleBindingRef = ""

		if err := r.Client.Status().Update(context.TODO(), role); err != nil {
			return err
		}
	}
	return nil
}
