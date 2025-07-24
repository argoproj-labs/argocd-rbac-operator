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

// ArgoCDProjectRoleSpec defines the desired state of an AppProject scoped Role (patched to binded AppProject).
type ArgoCDProjectRoleSpec struct {
	// Description of the role.
	Description string        `json:"description"`
	Rules       []ProjectRule `json:"rules"`
}

// Rules define the desired set of permissions.
type ProjectRule struct {
	// +kubebuilder:validation:Enum=clusters;applications;applicationsets;repositories;logs;exec;projects
	// +kubebuilder:validation:example=applications
	// Target resource type.
	Resource string `json:"resource"`
	// Verbs define the operations that are being performed on the resource.
	Verbs []string `json:"verbs"`
	// List of resource's objects the permissions are granted for.
	Objects []string `json:"objects"`
}

// ArgoCDProjectRoleStatus defines the observed state of ArgoCDProjectRole.
type ArgoCDProjectRoleStatus struct {
	// argocdProjectRoleBindingRef defines the reference to the ArgoCDProjectRoleBinding Resource.
	ArgoCDProjectRoleBindingRef string `json:"argocdProjectRoleBindingRef,omitempty"`
	// +listType=map
	// +listMapKey=type
	// Conditions defines the list of conditions.
	Conditions []Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient

// ArgoCDProjectRole is the Schema for the argocdprojectroles API.
type ArgoCDProjectRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArgoCDProjectRoleSpec   `json:"spec,omitempty"`
	Status ArgoCDProjectRoleStatus `json:"status,omitempty"`
}

// IsBeingDeleted returns true if a deletion timestamp is set
func (r *ArgoCDProjectRole) IsBeingDeleted() bool {
	return !r.DeletionTimestamp.IsZero()
}

// ArgoCDRoleFinalizerName is the name of the finalizer used to delete the Role
const ArgoCDProjectRoleFinalizerName = "rbac-operator.argoproj-labs.io/finalizer"

// HasFinalizer returns true if the Role has the finalizer
func (r *ArgoCDProjectRole) HasFinalizer(finalizerName string) bool {
	return slices.Contains(r.Finalizers, finalizerName)
}

// AddFinalizer adds the finalizer to the Role
func (r *ArgoCDProjectRole) AddFinalizer(finalizerName string) {
	r.Finalizers = append(r.Finalizers, finalizerName)
}

// RemoveFinalizer removes the finalizer from the Role
func (r *ArgoCDProjectRole) RemoveFinalizer(finalizerName string) {
	r.Finalizers = slices.DeleteFunc(r.Finalizers, func(s string) bool {
		return s == finalizerName
	})
}

// +kubebuilder:object:root=true

// ArgoCDProjectRoleList contains a list of ArgoCDProjectRole.
type ArgoCDProjectRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArgoCDProjectRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArgoCDProjectRole{}, &ArgoCDProjectRoleList{})
}
