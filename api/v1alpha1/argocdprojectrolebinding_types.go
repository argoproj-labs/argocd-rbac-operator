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

// ArgoCDProjectRoleBindingSpec defines the desired state of ArgoCDProjectRoleBinding.
type ArgoCDProjectRoleBindingSpec struct {
	// List of subjects being bound to ArgoCDProjectRole (argocdProjectRoleRef).
	// +kubebuilder:validation:MinItems=1
	Subjects             []AppProjectSubject  `json:"subjects"`
	ArgoCDProjectRoleRef ArgoCDProjectRoleRef `json:"argocdProjectRoleRef"`
}

// AppProjectSubject defines the subject being bound to ArgoCDProjectRole.
type AppProjectSubject struct {
	// Reference to the AppProject the ArgoCDRole is bound to.
	AppProjectRef string `json:"appProjectRef"`
	// List of groups the role will be granted to.
	Groups []string `json:"groups"`
}

// ArgocdProjectRoleRef defines the reference to the role being granted.
type ArgoCDProjectRoleRef struct {
	// Name of the ArgoCDProjectRole. Should not start with "role:"
	Name string `json:"name"`
}

// ArgoCDProjectRoleBindingStatus defines the observed state of ArgoCDProjectRoleBinding.
type ArgoCDProjectRoleBindingStatus struct {
	// +listType=map
	// +listMapKey=type
	// Conditions defines the list of conditions.
	Conditions []Condition `json:"conditions,omitempty"`
	// AppProjectsBound is a list of AppProjects that the role is bound to.
	AppProjectsBound []string `json:"appProjectsBound,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient

// ArgoCDProjectRoleBinding is the Schema for the argocdprojectrolebindings API.
type ArgoCDProjectRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArgoCDProjectRoleBindingSpec   `json:"spec,omitempty"`
	Status ArgoCDProjectRoleBindingStatus `json:"status,omitempty"`
}

// IsBeingDeleted returns true if a deletion timestamp is set
func (r *ArgoCDProjectRoleBinding) IsBeingDeleted() bool {
	return !r.DeletionTimestamp.IsZero()
}

// ArgoCDProjectRoleFinalizerName is the name of the finalizer used to delete the Role
const ArgoCDProjectRoleBindingFinalizerName = "rbac-operator.argoproj-labs.io/finalizer"

// HasFinalizer returns true if the Role has the finalizer
func (r *ArgoCDProjectRoleBinding) HasFinalizer(finalizerName string) bool {
	return slices.Contains(r.Finalizers, finalizerName)
}

// AddFinalizer adds the finalizer to the Role
func (r *ArgoCDProjectRoleBinding) AddFinalizer(finalizerName string) {
	r.Finalizers = append(r.Finalizers, finalizerName)
}

// RemoveFinalizer removes the finalizer from the Role
func (r *ArgoCDProjectRoleBinding) RemoveFinalizer(finalizerName string) {
	r.Finalizers = slices.DeleteFunc(r.Finalizers, func(s string) bool {
		return s == finalizerName
	})
}

// +kubebuilder:object:root=true

// ArgoCDProjectRoleBindingList contains a list of ArgoCDProjectRoleBinding.
type ArgoCDProjectRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArgoCDProjectRoleBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArgoCDProjectRoleBinding{}, &ArgoCDProjectRoleBindingList{})
}
