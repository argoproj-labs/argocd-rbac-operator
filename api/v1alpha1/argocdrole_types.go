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

// ArgoCDRoleSpec defines the desired state of Role
type ArgoCDRoleSpec struct {
	Rules []Rule `json:"rules"`
}

// Rules define the desired set of permissions.
type Rule struct {
	// +kubebuilder:validation:Enum=clusters;projects;applications;applicationsets;repositories;certificates;accounts;gpgkeys;logs;exec;extensions
	// +kubebuilder:validation:example=clusters
	Resource string   `json:"resource"`
	Verbs    []string `json:"verbs"`
	Objects  []string `json:"objects"`
}

// ArgoCDRoleStatus defines the observed state of Role
type ArgoCDRoleStatus struct {
	ArgoCDRoleBindingRef string `json:"argocdRoleBindingRef,omitempty"`
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient

// ArgoCDRole is the Schema for the roles API
type ArgoCDRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArgoCDRoleSpec   `json:"spec,omitempty"`
	Status ArgoCDRoleStatus `json:"status,omitempty"`
}

// IsBeingDeleted returns true if a deletion timestamp is set
func (r *ArgoCDRole) IsBeingDeleted() bool {
	return !r.ObjectMeta.DeletionTimestamp.IsZero()
}

// ArgoCDRoleFinalizerName is the name of the finalizer used to delete the Role
const ArgoCDRoleFinalizerName = "role.rbac-operator.argoproj-labs.io"

// HasFinalizer returns true if the Role has the finalizer
func (r *ArgoCDRole) HasFinalizer(finalizerName string) bool {
	return slices.Contains(r.ObjectMeta.Finalizers, finalizerName)
}

// AddFinalizer adds the finalizer to the Role
func (r *ArgoCDRole) AddFinalizer(finalizerName string) {
	r.ObjectMeta.Finalizers = append(r.ObjectMeta.Finalizers, finalizerName)
}

// RemoveFinalizer removes the finalizer from the Role
func (r *ArgoCDRole) RemoveFinalizer(finalizerName string) {
	r.ObjectMeta.Finalizers = slices.DeleteFunc(r.ObjectMeta.Finalizers, func(s string) bool {
		return s == finalizerName
	})
}

// +kubebuilder:object:root=true

// ArgoCDRoleList contains a list of Role
type ArgoCDRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArgoCDRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArgoCDRole{}, &ArgoCDRoleList{})
}
