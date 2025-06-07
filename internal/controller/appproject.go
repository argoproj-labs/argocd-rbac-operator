package controller

import (
	"context"
	"fmt"

	argocdv1alpha "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
)

func buildCasbinPolicyStrings(pr *rbacoperatorv1alpha1.ArgoCDProjectRole, appProject *argocdv1alpha.AppProject) []string {
	policies := []string{}
	for _, rule := range pr.Spec.Rules {
		resource := rule.Resource
		for _, verb := range rule.Verbs {
			for _, object := range rule.Objects {
				policy := fmt.Sprintf("p, proj:%s:%s, %s, %s, %s, allow", appProject.Name, pr.Name, resource, verb, object)
				policies = append(policies, policy)
			}
		}
	}
	return policies
}

func newAppProject(name, namespace string) *argocdv1alpha.AppProject {
	return &argocdv1alpha.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func (r *ArgoCDProjectRoleBindingReconciler) patchAppProject(appProject *argocdv1alpha.AppProject, pr *rbacoperatorv1alpha1.ArgoCDProjectRole, groups *[]string) error {
	changed := false
	apProjectRole := &argocdv1alpha.ProjectRole{
		Name:        pr.Name,
		Description: pr.Spec.Description,
		Groups:      *groups,
		Policies:    buildCasbinPolicyStrings(pr, appProject),
	}

	ogAppProject := appProject.DeepCopy()

	role, index := getRoleInAppProject(appProject, pr.Name)
	if role == nil {
		appProject.Spec.Roles = append(appProject.Spec.Roles, *apProjectRole)
		changed = true
	}
	if role != nil && !areProjectRolesEqual(role, apProjectRole) {
		appProject.Spec.Roles[index] = *apProjectRole
		changed = true
	}
	if changed {
		return r.Patch(context.TODO(), appProject, client.MergeFrom(ogAppProject))
	}
	return nil
}

func getRoleInAppProject(appProject *argocdv1alpha.AppProject, roleName string) (role *argocdv1alpha.ProjectRole, index int) {
	for i, role := range appProject.Spec.Roles {
		if role.Name == roleName {
			return &role, i
		}
	}
	return nil, -1
}

func areProjectRolesEqual(r1, r2 *argocdv1alpha.ProjectRole) bool {
	if r1.Description != r2.Description ||
		len(r1.Groups) != len(r2.Groups) ||
		len(r1.Policies) != len(r2.Policies) {
		return false
	}

	for i, group := range r1.Groups {
		if group != r2.Groups[i] {
			return false
		}
	}

	for i, policy := range r1.Policies {
		if policy != r2.Policies[i] {
			return false
		}
	}

	return true
}

func removeRoleFromAppProject(rClient client.Client, appProject *argocdv1alpha.AppProject, roleName string) error {
	ogAppProject := appProject.DeepCopy()

	_, index := getRoleInAppProject(appProject, roleName)
	if index == -1 {
		return nil // Role not found in AppProject, nothing to delete
	}
	appProject.Spec.Roles = append(appProject.Spec.Roles[:index], appProject.Spec.Roles[index+1:]...)
	if err := rClient.Patch(context.TODO(), appProject, client.MergeFrom(ogAppProject)); err != nil {
		return errors.Wrapf(err, "failed to patch AppProject %s/%s to remove role %s", appProject.Namespace, appProject.Name, roleName)
	}
	return nil
}
