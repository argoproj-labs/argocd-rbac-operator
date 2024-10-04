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

// ArgoCDRoleBindingSpec defines the desired state of ArgoCDRoleBinding
type ArgoCDRoleBindingSpec struct {
	// List of subjects being bound to ArgoCDRole (argocdRoleRef).
	Subjects      []Subject     `json:"subjects"`
	ArgoCDRoleRef ArgoCDRoleRef `json:"argocdRoleRef"`
}

// Kind of the subject (sso, local).
type Subject struct {
	// +kubebuilder:validation:Enum=sso;local;role
	Kind string `json:"kind"`
	// Name of the subject. If Kind is "role", it shouldn't start with "role:"
	Name string `json:"name"`
}

// argocdRoleRef defines the reference to the role being granted.
type ArgoCDRoleRef struct {
	// Name of the ArgoCDRole. Should not start with "role:"
	Name string `json:"name"`
}

// ArgoCDRoleBindingStatus defines the observed state of ArgoCDRoleBinding
type ArgoCDRoleBindingStatus struct {
	// +listType=map
	// +listMapKey=type
	// Conditions defines the list of conditions.
	Conditions []Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient

// ArgoCDRoleBinding is the Schema for the argocdrolebindings API
type ArgoCDRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArgoCDRoleBindingSpec   `json:"spec,omitempty"`
	Status ArgoCDRoleBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ArgoCDRoleBindingList contains a list of ArgoCDRoleBinding
type ArgoCDRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArgoCDRoleBinding `json:"items"`
}

// IsBeingDeleted returns true if a deletion timestamp is set
func (r *ArgoCDRoleBinding) IsBeingDeleted() bool {
	return !r.ObjectMeta.DeletionTimestamp.IsZero()
}

// ArgoCDRoleFinalizerName is the name of the finalizer used to delete the Role
const ArgoCDRoleBindingFinalizerName = "rolebinding.rbac-operator.argoproj-labs.io"

// HasFinalizer returns true if the Role has the finalizer
func (r *ArgoCDRoleBinding) HasFinalizer(finalizerName string) bool {
	return slices.Contains(r.ObjectMeta.Finalizers, finalizerName)
}

// AddFinalizer adds the finalizer to the Role
func (r *ArgoCDRoleBinding) AddFinalizer(finalizerName string) {
	r.ObjectMeta.Finalizers = append(r.ObjectMeta.Finalizers, finalizerName)
}

// RemoveFinalizer removes the finalizer from the Role
func (r *ArgoCDRoleBinding) RemoveFinalizer(finalizerName string) {
	r.ObjectMeta.Finalizers = slices.DeleteFunc(r.ObjectMeta.Finalizers, func(s string) bool {
		return s == finalizerName
	})
}

func init() {
	SchemeBuilder.Register(&ArgoCDRoleBinding{}, &ArgoCDRoleBindingList{})
}
