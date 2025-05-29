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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
	"github.com/argoproj-labs/argocd-rbac-operator/internal/controller/common"
)

// getRBACDefaultPolicyCSV will return the Argo CD RBAC default policy CSV.
func getDefaultRBACPolicy() string {
	return common.ArgoCDDefaultRBACPolicy
}

func getRBACPolicyCSV(role *rbacoperatorv1alpha1.ArgoCDRole, rb *rbacoperatorv1alpha1.ArgoCDRoleBinding) string {
	policy := ""
	roleName := fmt.Sprintf("role:%s", role.ObjectMeta.Name)

	policy += buildPolicyStringRules(role, roleName)
	policy += buildPolicyStringSubjects(rb, role)

	return policy
}

// buildPolicyStringRules will build the policy string for Rules field of the given role.
func buildPolicyStringRules(role *rbacoperatorv1alpha1.ArgoCDRole, roleName string) string {
	policy := ""
	for _, rule := range role.Spec.Rules {
		resource := rule.Resource
		for _, verb := range rule.Verbs {
			for _, object := range rule.Objects {
				policy += fmt.Sprintf("p, %s, %s, %s, %s, allow\n", roleName, resource, verb, object)
			}
		}
	}
	return policy
}

// buildPolicyStringSubjects will build the policy string for Subjects field of the given role.
func buildPolicyStringSubjects(rb *rbacoperatorv1alpha1.ArgoCDRoleBinding, role *rbacoperatorv1alpha1.ArgoCDRole) string {
	policy := ""
	roleName := fmt.Sprintf("role:%s", role.ObjectMeta.Name)
	for _, subject := range rb.Spec.Subjects {
		switch subject.Kind {
		case "sso":
			policy += fmt.Sprintf("g, %s, %s\n", subject.Name, roleName)
		case "role":
			subjectRoleName := fmt.Sprintf("role:%s", subject.Name)
			policy += fmt.Sprintf("g, %s, %s\n", subjectRoleName, roleName)
		case "local":
			policy += buildPolicyStringRules(role, subject.Name)
		}
	}
	return policy
}

// newConfigMap will return a new ConfigMap resource.
func newConfigMap(name, namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// reconcileRBACConfigMap will ensure that the ArgoCD RBAC ConfigMap is up-to-date.
func (r *ArgoCDRoleReconciler) reconcileRBACConfigMap(cm *corev1.ConfigMap, role *rbacoperatorv1alpha1.ArgoCDRole) error {
	changed := false
	overlayKey := fmt.Sprintf("policy.%s.%s.csv", role.Namespace, role.Name)
	roleName := fmt.Sprintf("role:%s", role.Name)

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	// Default Policy String
	if cm.Data[common.ArgoCDKeyRBACPolicyCSV] != getDefaultRBACPolicy() {
		cm.Data[common.ArgoCDKeyRBACPolicyCSV] = getDefaultRBACPolicy()
		changed = true
	}
	// Policy OverlayKey CSV
	if cm.Data[overlayKey] != buildPolicyStringRules(role, roleName) {
		cm.Data[overlayKey] = buildPolicyStringRules(role, roleName)
		changed = true
	}

	if changed {
		return r.Client.Update(context.TODO(), cm)
	}
	return nil
}

// reconcileRBACConfigMapWithRoleBinding will ensure that the ArgoCD RBAC ConfigMap is up-to-date.
func (r *ArgoCDRoleReconciler) reconcileRBACConfigMapWithRoleBinding(cm *corev1.ConfigMap, role *rbacoperatorv1alpha1.ArgoCDRole, rb *rbacoperatorv1alpha1.ArgoCDRoleBinding) error {
	changed := false
	overlayKey := fmt.Sprintf("policy.%s.%s.csv", role.Namespace, role.Name)

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	// Default Policy String
	if cm.Data[common.ArgoCDKeyRBACPolicyCSV] != getDefaultRBACPolicy() {
		cm.Data[common.ArgoCDKeyRBACPolicyCSV] = getDefaultRBACPolicy()
		changed = true
	}
	// Policy OverlayKey CSV
	if cm.Data[overlayKey] != getRBACPolicyCSV(role, rb) {
		cm.Data[overlayKey] = getRBACPolicyCSV(role, rb)
		changed = true
	}

	if changed {
		return r.Client.Update(context.TODO(), cm)
	}
	return nil
}

// reconcileRBACConfigMap will ensure that the ArgoCD RBAC ConfigMap is up-to-date.
func (r *ArgoCDRoleBindingReconciler) reconcileRBACConfigMap(cm *corev1.ConfigMap, rb *rbacoperatorv1alpha1.ArgoCDRoleBinding, role *rbacoperatorv1alpha1.ArgoCDRole) error {
	changed := false
	overlayKey := fmt.Sprintf("policy.%s.%s.csv", role.Namespace, role.Name)

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	// Default Policy String
	if cm.Data[common.ArgoCDKeyRBACPolicyCSV] != getDefaultRBACPolicy() {
		cm.Data[common.ArgoCDKeyRBACPolicyCSV] = getDefaultRBACPolicy()
		changed = true
	}
	// Policy OverlayKey CSV
	if cm.Data[overlayKey] != getRBACPolicyCSV(role, rb) {
		cm.Data[overlayKey] = getRBACPolicyCSV(role, rb)
		changed = true
	}

	if changed {
		return r.Client.Update(context.TODO(), cm)
	}
	return nil
}

// reconcileRBACConfigMap will ensure that the ArgoCD RBAC ConfigMap is up-to-date.
func (r *ArgoCDRoleBindingReconciler) reconcileRBACConfigMapForBuiltInRole(cm *corev1.ConfigMap, rb *rbacoperatorv1alpha1.ArgoCDRoleBinding, role *rbacoperatorv1alpha1.ArgoCDRole) error {
	changed := false
	overlayKey := fmt.Sprintf("policy.%s.%s.csv", role.Namespace, role.Name)

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	// Default Policy String
	if cm.Data[common.ArgoCDKeyRBACPolicyCSV] != getDefaultRBACPolicy() {
		cm.Data[common.ArgoCDKeyRBACPolicyCSV] = getDefaultRBACPolicy()
		changed = true
	}
	// Policy OverlayKey CSV
	if cm.Data[overlayKey] != buildPolicyStringSubjects(rb, role) {
		cm.Data[overlayKey] = buildPolicyStringSubjects(rb, role)
		changed = true
	}

	if changed {
		return r.Client.Update(context.TODO(), cm)
	}
	return nil
}

// IsObjectFound will perform a basic check that the given object exists via the Kubernetes API.
// If an error occurs as part of the check, the function will return false.
func IsObjectFound(client client.Client, namespace string, name string, obj client.Object) bool {
	return !apierrors.IsNotFound(FetchObject(client, namespace, name, obj))
}

// FetchObject will retrieve the object with the given namespace and name using the Kubernetes API.
// The result will be stored in the given object.
func FetchObject(client client.Client, namespace string, name string, obj client.Object) error {
	return client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, obj)
}

// createBuiltInAdminRole will return a new built-in ArgoCDRole with admin permissions.
func (r *ArgoCDRoleBindingReconciler) createBuiltInAdminRole() *rbacoperatorv1alpha1.ArgoCDRole {
	return &rbacoperatorv1alpha1.ArgoCDRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.ArgoCDRoleAdmin,
			Namespace: r.ArgoCDRBACConfigMapNamespace,
		},
		Spec: rbacoperatorv1alpha1.ArgoCDRoleSpec{
			Rules: []rbacoperatorv1alpha1.Rule{
				{
					Resource: "applications",
					Verbs:    []string{"override", "sync", "create", "update", "delete", "action", "get"},
					Objects:  []string{"*/*"},
				},
				{
					Resource: "applicationsets",
					Verbs:    []string{"create", "update", "delete", "get"},
					Objects:  []string{"*/*"},
				},
				{
					Resource: "certificates",
					Verbs:    []string{"create", "update", "delete", "get"},
					Objects:  []string{"*"},
				},
				{
					Resource: "clusters",
					Verbs:    []string{"create", "update", "delete", "get"},
					Objects:  []string{"*"},
				},
				{
					Resource: "repositories",
					Verbs:    []string{"create", "update", "delete", "get"},
					Objects:  []string{"*"},
				},
				{
					Resource: "projects",
					Verbs:    []string{"create", "update", "delete", "get"},
					Objects:  []string{"*"},
				},
				{
					Resource: "accounts",
					Verbs:    []string{"update", "get"},
					Objects:  []string{"*"},
				},
				{
					Resource: "gpgkeys",
					Verbs:    []string{"create", "get", "delete"},
					Objects:  []string{"*"},
				},
				{
					Resource: "exec",
					Verbs:    []string{"create"},
					Objects:  []string{"*/*"},
				},
				{
					Resource: "logs",
					Verbs:    []string{"get"},
					Objects:  []string{"*/*"},
				},
			},
		},
	}
}

// createBuiltInReadOnlyRole will return a new built-in ArgoCDRole with read-only permissions.
func (r *ArgoCDRoleBindingReconciler) createBuiltInReadOnlyRole() *rbacoperatorv1alpha1.ArgoCDRole {
	return &rbacoperatorv1alpha1.ArgoCDRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.ArgoCDRoleReadOnly,
			Namespace: r.ArgoCDRBACConfigMapNamespace,
		},
		Spec: rbacoperatorv1alpha1.ArgoCDRoleSpec{
			Rules: []rbacoperatorv1alpha1.Rule{
				{
					Resource: "applications",
					Verbs:    []string{"get"},
					Objects:  []string{"*/*"},
				},
				{
					Resource: "certificates",
					Verbs:    []string{"get"},
					Objects:  []string{"*"},
				},
				{
					Resource: "clusters",
					Verbs:    []string{"get"},
					Objects:  []string{"*"},
				},
				{
					Resource: "repositories",
					Verbs:    []string{"get"},
					Objects:  []string{"*"},
				},
				{
					Resource: "projects",
					Verbs:    []string{"get"},
					Objects:  []string{"*"},
				},
				{
					Resource: "accounts",
					Verbs:    []string{"get"},
					Objects:  []string{"*"},
				},
				{
					Resource: "gpgkeys",
					Verbs:    []string{"get"},
					Objects:  []string{"*"},
				},
				{
					Resource: "logs",
					Verbs:    []string{"get"},
					Objects:  []string{"*/*"},
				},
			},
		},
	}
}
