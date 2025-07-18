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

// ArgoCDRoleSpec defines the desired state of global scoped Role (written to argocd-rbac-cm ConfigMap)
type ArgoCDRoleSpec struct {
	Rules []GlobalRule `json:"rules"`
}

// Rules define the desired set of permissions.
type GlobalRule struct {
	// +kubebuilder:validation:Enum=clusters;projects;applications;applicationsets;repositories;certificates;accounts;gpgkeys;logs;exec;extensions
	// +kubebuilder:validation:example=clusters
	// Target resource type.
	Resource string `json:"resource"`
	// Verbs define the operations that are being performed on the resource.
	Verbs []string `json:"verbs"`
	// List of resource's objects the permissions are granted for.
	Objects []string `json:"objects"`
}

// ArgoCDRoleStatus defines the observed state of Role
type ArgoCDRoleStatus struct {
	// argocdRoleBindingRef defines the reference to the ArgoCDRoleBinding Resource.
	ArgoCDRoleBindingRef string `json:"argocdRoleBindingRef,omitempty"`
	// +listType=map
	// +listMapKey=type
	// Conditions defines the list of conditions.
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
	return !r.DeletionTimestamp.IsZero()
}

// ArgoCDRoleFinalizerName is the name of the finalizer used to delete the Role
const ArgoCDRoleFinalizerName = "rbac-operator.argoproj-labs.io/finalizer"

// HasFinalizer returns true if the Role has the finalizer
func (r *ArgoCDRole) HasFinalizer(finalizerName string) bool {
	return slices.Contains(r.Finalizers, finalizerName)
}

// AddFinalizer adds the finalizer to the Role
func (r *ArgoCDRole) AddFinalizer(finalizerName string) {
	r.Finalizers = append(r.Finalizers, finalizerName)
}

// RemoveFinalizer removes the finalizer from the Role
func (r *ArgoCDRole) RemoveFinalizer(finalizerName string) {
	r.Finalizers = slices.DeleteFunc(r.Finalizers, func(s string) bool {
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
