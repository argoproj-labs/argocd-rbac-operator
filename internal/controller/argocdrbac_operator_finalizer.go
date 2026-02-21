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

	argocdv1alpha "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

	if err := r.delete(ctx, role); err != nil {
		return err
	}

	role.RemoveFinalizer(rbacoperatorv1alpha1.ArgoCDRoleFinalizerName)
	return r.Update(ctx, role)
}

func (r *ArgoCDRoleReconciler) delete(ctx context.Context, role *rbacoperatorv1alpha1.ArgoCDRole) error {
	cm := newConfigMap(r.ArgoCDRBACConfigMapName, r.ArgoCDRBACConfigMapNamespace)
	overlayKey := fmt.Sprintf("policy.%s.%s.csv", role.Namespace, role.Name)
	if IsObjectFound(ctx, r.Client, cm.Namespace, cm.Name, cm) {
		delete(cm.Data, overlayKey)
		if err := r.Update(ctx, cm); err != nil {
			return err
		}
	}

	return nil
}

func (r *ArgoCDProjectRoleReconciler) addFinalizer(ctx context.Context, projectRole *rbacoperatorv1alpha1.ArgoCDProjectRole) error {
	projectRole.AddFinalizer(rbacoperatorv1alpha1.ArgoCDProjectRoleFinalizerName)
	return r.Update(ctx, projectRole)
}

func (r *ArgoCDProjectRoleReconciler) handleFinalizer(ctx context.Context, projectRole *rbacoperatorv1alpha1.ArgoCDProjectRole) error {
	if !projectRole.HasFinalizer(rbacoperatorv1alpha1.ArgoCDProjectRoleFinalizerName) {
		return nil
	}

	if err := r.delete(ctx, projectRole); err != nil {
		return err
	}

	projectRole.RemoveFinalizer(rbacoperatorv1alpha1.ArgoCDProjectRoleFinalizerName)
	return r.Update(ctx, projectRole)
}

func (r *ArgoCDProjectRoleReconciler) delete(ctx context.Context, projectRole *rbacoperatorv1alpha1.ArgoCDProjectRole) error {
	rbName := projectRole.Status.ArgoCDProjectRoleBindingRef
	if rbName == "" {
		return nil // Role not bound to any AppProject, nothing to delete
	}
	rb := &rbacoperatorv1alpha1.ArgoCDProjectRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbName,
			Namespace: projectRole.Namespace,
		},
	}
	if !IsObjectFound(ctx, r.Client, rb.Namespace, rb.Name, rb) {
		// RoleBinding does not exist, nothing to delete
		return nil
	}
	appProjectNames := []string{}
	// get all AppProjects this role is bound to
	for _, subject := range rb.Spec.Subjects {
		appProjectNames = append(appProjectNames, subject.AppProjectRef)
	}
	return deleteProjectRoles(ctx, r.Client, appProjectNames, projectRole.Name, projectRole.Namespace)
}

func (r *ArgoCDRoleBindingReconciler) addFinalizer(ctx context.Context, rb *rbacoperatorv1alpha1.ArgoCDRoleBinding) error {
	rb.AddFinalizer(rbacoperatorv1alpha1.ArgoCDRoleBindingFinalizerName)
	return r.Update(ctx, rb)
}

func (r *ArgoCDRoleBindingReconciler) handleFinalizer(ctx context.Context, rb *rbacoperatorv1alpha1.ArgoCDRoleBinding) error {
	if !rb.HasFinalizer(rbacoperatorv1alpha1.ArgoCDRoleBindingFinalizerName) {
		return nil
	}

	if err := r.delete(ctx, rb); err != nil {
		return err
	}

	rb.RemoveFinalizer(rbacoperatorv1alpha1.ArgoCDRoleBindingFinalizerName)
	return r.Update(ctx, rb)
}

func (r *ArgoCDRoleBindingReconciler) delete(ctx context.Context, rb *rbacoperatorv1alpha1.ArgoCDRoleBinding) error {
	roleRefName := rb.Spec.ArgoCDRoleRef.Name
	if roleRefName == common.ArgoCDRoleAdmin || roleRefName == common.ArgoCDRoleReadOnly {
		cm := newConfigMap(r.ArgoCDRBACConfigMapName, r.ArgoCDRBACConfigMapNamespace)
		overlayKey := fmt.Sprintf("policy.%s.%s.csv", rb.Namespace, roleRefName)
		if IsObjectFound(ctx, r.Client, cm.Namespace, cm.Name, cm) {
			delete(cm.Data, overlayKey)
			if err := r.Update(ctx, cm); err != nil {
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
	if IsObjectFound(ctx, r.Client, role.Namespace, role.Name, role) {
		role.Status.ArgoCDRoleBindingRef = ""

		if err := r.Status().Update(ctx, role); err != nil {
			return err
		}
	}
	return nil
}

func (r *ArgoCDProjectRoleBindingReconciler) addFinalizer(ctx context.Context, projectRoleBinding *rbacoperatorv1alpha1.ArgoCDProjectRoleBinding) error {
	projectRoleBinding.AddFinalizer(rbacoperatorv1alpha1.ArgoCDProjectRoleBindingFinalizerName)
	return r.Update(ctx, projectRoleBinding)
}

func (r *ArgoCDProjectRoleBindingReconciler) handleFinalizer(ctx context.Context, projectRoleBinding *rbacoperatorv1alpha1.ArgoCDProjectRoleBinding) error {
	if !projectRoleBinding.HasFinalizer(rbacoperatorv1alpha1.ArgoCDProjectRoleBindingFinalizerName) {
		return nil
	}

	if err := r.delete(ctx, projectRoleBinding); err != nil {
		return err
	}

	projectRoleBinding.RemoveFinalizer(rbacoperatorv1alpha1.ArgoCDProjectRoleBindingFinalizerName)
	return r.Update(ctx, projectRoleBinding)
}

func (r *ArgoCDProjectRoleBindingReconciler) delete(ctx context.Context, projectRoleBinding *rbacoperatorv1alpha1.ArgoCDProjectRoleBinding) error {
	roleName := projectRoleBinding.Spec.ArgoCDProjectRoleRef.Name

	appProjectNames := []string{}
	for _, subject := range projectRoleBinding.Spec.Subjects {
		appProjectNames = append(appProjectNames, subject.AppProjectRef)
	}
	return deleteProjectRoles(ctx, r.Client, appProjectNames, roleName, projectRoleBinding.Namespace)
}

func deleteProjectRoles(ctx context.Context, rClient client.Client, appProjects []string, roleName string, namespace string) error {
	for _, appProjectName := range appProjects {
		appProject := &argocdv1alpha.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      appProjectName,
				Namespace: namespace,
			},
		}
		if !IsObjectFound(ctx, rClient, appProject.Namespace, appProject.Name, appProject) {
			continue // AppProject does not exist, nothing to delete
		}
		if err := removeRoleFromAppProject(ctx, rClient, appProject, roleName); err != nil {
			return errors.Wrapf(err, "failed to remove role %s from AppProject %s", roleName, appProjectName)
		}
	}
	return nil
}
