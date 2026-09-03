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

	argocdv1alpha "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rbacoperatorv1alpha1 "github.com/argoproj-labs/argocd-rbac-operator/api/v1alpha1"
)

func TestBuildCasbinPolicyStrings_Effect(t *testing.T) {
	tests := []struct {
		name   string
		effect string
		want   string
	}{
		{name: "empty effect defaults to allow", effect: "", want: "p, proj:proj-test:role-test, applications, get, */*, allow"},
		{name: "explicit allow", effect: "allow", want: "p, proj:proj-test:role-test, applications, get, */*, allow"},
		{name: "explicit deny", effect: "deny", want: "p, proj:proj-test:role-test, applications, get, */*, deny"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &rbacoperatorv1alpha1.ArgoCDProjectRole{
				ObjectMeta: metav1.ObjectMeta{Name: "role-test"},
				Spec: rbacoperatorv1alpha1.ArgoCDProjectRoleSpec{
					Rules: []rbacoperatorv1alpha1.ProjectRule{
						{
							Resource: "applications",
							Verbs:    []string{"get"},
							Objects:  []string{"*/*"},
							Effect:   tt.effect,
						},
					},
				},
			}
			appProject := &argocdv1alpha.AppProject{ObjectMeta: metav1.ObjectMeta{Name: "proj-test"}}

			got := buildCasbinPolicyStrings(pr, appProject)
			assert.Equal(t, []string{tt.want}, got)
		})
	}
}
