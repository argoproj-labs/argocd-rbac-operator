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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ArgoCDApplicationRoleSpec defines a reusable set of application-scoped permissions.
type ArgoCDApplicationRoleSpec struct {
	// Description is copied to every generated AppProject role.
	Description string `json:"description,omitempty"`
	// Rules define the permissions granted by this role.
	// +kubebuilder:validation:MinItems=1
	Rules []ApplicationRule `json:"rules"`
}

// ApplicationRule defines permissions whose object is supplied by an ArgoCDApplicationRoleBinding.
type ApplicationRule struct {
	// Resource is an Argo CD resource scoped to an application.
	// +kubebuilder:validation:Enum=applications;logs;exec
	Resource string `json:"resource"`
	// Verbs define the operations that may be performed.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:items:MinLength=1
	// +kubebuilder:validation:items:Pattern=`^[^,\r\n]+$`
	Verbs []string `json:"verbs"`
	// Effect controls whether matching requests are allowed or denied.
	// +kubebuilder:default=allow
	// +kubebuilder:validation:Enum=allow;deny
	Effect string `json:"effect,omitempty"`
}

// +kubebuilder:object:root=true
// +genclient

// ArgoCDApplicationRole is the Schema for reusable application-scoped permission sets.
type ArgoCDApplicationRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ArgoCDApplicationRoleSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// ArgoCDApplicationRoleList contains a list of ArgoCDApplicationRole resources.
type ArgoCDApplicationRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArgoCDApplicationRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArgoCDApplicationRole{}, &ArgoCDApplicationRoleList{})
}
