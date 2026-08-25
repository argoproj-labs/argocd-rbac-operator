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
	"testing"

	"github.com/stretchr/testify/assert"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
)

func TestBuildPolicyStringRules_Effect(t *testing.T) {
	tests := []struct {
		name   string
		effect string
		want   string
	}{
		{name: "empty effect defaults to allow", effect: "", want: "p, role:test, applications, get, */*, allow\n"},
		{name: "explicit allow", effect: "allow", want: "p, role:test, applications, get, */*, allow\n"},
		{name: "explicit deny", effect: "deny", want: "p, role:test, applications, get, */*, deny\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := &rbacoperatorv1alpha1.ArgoCDRole{
				Spec: rbacoperatorv1alpha1.ArgoCDRoleSpec{
					Rules: []rbacoperatorv1alpha1.GlobalRule{
						{
							Resource: "applications",
							Verbs:    []string{"get"},
							Objects:  []string{"*/*"},
							Effect:   tt.effect,
						},
					},
				},
			}
			assert.Equal(t, tt.want, buildPolicyStringRules(role, "role:test"))
		})
	}
}
