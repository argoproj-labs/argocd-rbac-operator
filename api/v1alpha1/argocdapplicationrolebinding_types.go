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

package v1alpha1

import (
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ArgoCDApplicationRoleBindingSpec binds a reusable role to teams and applications.
type ArgoCDApplicationRoleBindingSpec struct {
	ArgoCDApplicationRoleRef ArgoCDApplicationRoleRef `json:"argocdApplicationRoleRef"`
	AppProjectRef            AppProjectReference      `json:"appProjectRef"`
	// Applications is the list of application names or patterns to which the role applies.
	// +kubebuilder:validation:MinItems=1
	Applications []ApplicationTarget `json:"applications"`
	// Groups is the list of OIDC group claims bound to the generated AppProject role.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:items:MinLength=1
	Groups []string `json:"groups"`
}

// ArgoCDApplicationRoleRef references an ArgoCDApplicationRole in the same namespace.
type ArgoCDApplicationRoleRef struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// AppProjectReference references an AppProject in the same namespace.
type AppProjectReference struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// ApplicationTarget identifies an application or application pattern.
type ApplicationTarget struct {
	// Name is an application name or pattern. Commas, slashes, and newlines are not allowed.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[^,/\r\n]+$`
	Name string `json:"name"`
	// Namespace optionally targets an Application in Any Namespace.
	// +kubebuilder:validation:Pattern=`^[^,/\r\n]+$`
	Namespace string `json:"namespace,omitempty"`
}

// ArgoCDApplicationRoleBindingStatus defines the observed binding state.
type ArgoCDApplicationRoleBindingStatus struct {
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty"`
	// AppProjectBound is the AppProject currently containing the generated role.
	AppProjectBound string `json:"appProjectBound,omitempty"`
	// GeneratedRoleName is the name of the role generated in the AppProject.
	GeneratedRoleName string `json:"generatedRoleName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient

// ArgoCDApplicationRoleBinding is the Schema for application-scoped team access.
type ArgoCDApplicationRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArgoCDApplicationRoleBindingSpec   `json:"spec,omitempty"`
	Status ArgoCDApplicationRoleBindingStatus `json:"status,omitempty"`
}

// ArgoCDApplicationRoleBindingFinalizerName is used to remove generated AppProject roles.
const ArgoCDApplicationRoleBindingFinalizerName = "rbac-operator.argoproj-labs.io/application-role-binding-finalizer"

// IsBeingDeleted returns true if a deletion timestamp is set.
func (r *ArgoCDApplicationRoleBinding) IsBeingDeleted() bool {
	return !r.DeletionTimestamp.IsZero()
}

// HasFinalizer returns true if the binding contains finalizerName.
func (r *ArgoCDApplicationRoleBinding) HasFinalizer(finalizerName string) bool {
	return slices.Contains(r.Finalizers, finalizerName)
}

// AddFinalizer adds finalizerName to the binding.
func (r *ArgoCDApplicationRoleBinding) AddFinalizer(finalizerName string) {
	r.Finalizers = append(r.Finalizers, finalizerName)
}

// RemoveFinalizer removes finalizerName from the binding.
func (r *ArgoCDApplicationRoleBinding) RemoveFinalizer(finalizerName string) {
	r.Finalizers = slices.DeleteFunc(r.Finalizers, func(s string) bool {
		return s == finalizerName
	})
}

// +kubebuilder:object:root=true

// ArgoCDApplicationRoleBindingList contains a list of ArgoCDApplicationRoleBinding resources.
type ArgoCDApplicationRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArgoCDApplicationRoleBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArgoCDApplicationRoleBinding{}, &ArgoCDApplicationRoleBindingList{})
}
