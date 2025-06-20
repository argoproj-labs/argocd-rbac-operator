# Argo CD RBAC Operator

[![Go Report Card](https://goreportcard.com/badge/github.com/argoproj-labs/argocd-rbac-operator)](https://goreportcard.com/report/github.com/argoproj-labs/argocd-rbac-operator)
[![go.mod Go version](https://img.shields.io/github/go-mod/go-version/argoproj-labs/argocd-rbac-operator)](https://github.com/argoproj-labs/argocd-rbac-operator)
[![GitHub Release](https://img.shields.io/github/v/release/argoproj-labs/argocd-rbac-operator)](https://github.com/argoproj-labs/argocd-rbac-operator/releases/tag/v0.2.0)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/argocd-rbac-operator)](https://artifacthub.io/packages/search?repo=argocd-rbac-operator)

Kubernetes Operator for Argo CD RBAC Management.

## Introduction

The Argo CD RBAC Operator provides a CRD based API for the RBAC management of Argo CD. It provides a structured and easy to use way to define RBAC policies. The Operator uses the CRs as a single source of truth for RBAC management and converts them into a policy string that is patched into the Argo CD RBAC ConfigMap or AppProjects.

## Installation

### With Repository + Kustomize

#### Get the Repository

Clone the Argo CD RBAC Operator repository.

```bash
git clone https://github.com/argoproj-labs/argocd-rbac-operator.git
cd argocd-rbac-operator
```

#### Namespace

By default, the operator is installed into the `argocd-rbac-operator-system` namespace. To modify this, update the value of the namespace specified in the `config/default/kustomization.yaml` file.

#### Deploy Operator

Deploy the operator. This will create all the necessary resources, including the namespace. For running the task command you need to install [task](https://taskfile.dev/) and go-lang package on your system.

```bash
task deploy
```

The operator pod should start and enter a Running state after a few seconds.

```bash
kubectl get pods -n argocd-rbac-operator-system
```

### With Helm

First you have to add the repo:

```bash
helm repo add argocd-rbac-operator https://argoproj-labs.github.io/argocd-rbac-operator/
```

After the repo has been added, you can install the Helm chart of the operator:

```bash
helm install argocd-rbac-operator argocd-rbac-operator/argocd-rbac-operator
```

If you want to change the namespace of the Argo CD instance, image version, or other values, you have to define a values.yaml file and run following command:

```bash
helm install argocd-rbac-operator argocd-rbac-operator/argocd-rbac-operator -f values.yaml
```

## Usage

### Global-scoped RBAC

The following example shows a manifest to create a new ArgoCDRole `test-role`:

```yaml
apiVersion: rbac-operator.argoproj-labs.io/v1alpha1
kind: ArgoCDRole
metadata:
  labels:
    app.kubernetes.io/name: argocd-rbac-operator
    app.kubernetes.io/managed-by: kustomize
  name: test-role
  namespace: test-ns
spec:
  rules:
  - resource: "applications"
    verbs: ["get", "create", "update", "delete"]
    objects: ["*/*"]
```

And a ArgoCDRoleBinding `test-role-binding` to bind the specified users and a role to the new ArgoCDRole:

```yaml
apiVersion: rbac-operator.argoproj-labs.io/v1alpha1
kind: ArgoCDRoleBinding
metadata:
  labels:
    app.kubernetes.io/name: argocd-rbac-operator
    app.kubernetes.io/managed-by: kustomize
  name: test-role-binding
  namespace: test-ns
spec:
  subjects:
  - kind: "sso"
    name: "gosha"
  - kind: "local"
    name: "localUser"
  - kind: "role"
    name: "orgadmin"
  argocdRoleRef:
    name: "test-role"
```

#### Create ArgoCDRoles and ArgoCDRoleBindings

Make sure that the `argocd` Namespace exists, so that the ConfigMap can be created properly.

Create a new ArgoCDRole and ArgoCDRoleBinding using the provided example. (Make sure that both CRs are created in the same Namespace)

```bash
kubectl create -f test-role.yaml
kubectl create -f test-role-binding.yaml
```

The following ConfigMap will be created after the ArgoCDRole and ArgoCDRoleBinding has been reconciled.

```yaml
apiVersion: v1
data:
  policy.csv: ""
  policy.default: role:readonly
  policy.test-ns.test-role.csv: |
    p, role:test-role, applications, get, */*, allow
    p, role:test-role, applications, create, */*, allow
    p, role:test-role, applications, update, */*, allow
    p, role:test-role, applications, delete, */*, allow
    g, gosha, role:test-role
    p, localUser, applications, get, */*, allow
    p, localUser, applications, create, */*, allow
    p, localUser, applications, update, */*, allow
    p, localUser, applications, delete, */*, allow
    g, role:orgadmin, role:test-role
  scopes: '[groups]'
kind: ConfigMap
metadata:
  name: argocd-rbac-cm
  namespace: argocd
```

#### Delete ArgoCDRoles and ArgoCDRoleBindings

To delete a Role you can use `kubectl`

```bash
kubectl delete argocdrole.rbac-operator.argoproj-labs.io/test-role
kubectl delete argocdrolebinding.rbac-operator.argoproj-labs.io/test-role-binding
```

After the Resource is deleted, the policy string will be also deleted from the RBAC-CM.

#### Change the Policy.CSV

To change the policy.csv you have to make changes in the `internal/controller/common/defaults.go` file.

#### Deployment types

As for now only single Argo CD deployment type is supported. The default Argo CD namespace is defined as `argocd`, to change that you have to provide a flag `--argocd-rbac-cm-namespace="your-argocd-namespace"`.

### AppProject-scoped RBAC

The following example shows a manifest to create a new ArgoCDProjectRole `test-project-role`:

```yaml
apiVersion: rbac-operator.argoproj-labs.io/v1alpha1
kind: ArgoCDProjectRole
metadata:
  name: test-project-role
  namespace: test-ns
spec:
  description: "Test role for ArgoCD's AppProjects"
  rules:
  - resource: clusters
    verbs:
    - get
    - watch
    objects:
    - "*"
  - resource: applications
    verbs:
    - get
    objects:
    - "*"
```

And a ArgoCDProjectRoleBinding `test-project-role-binding` to bind the specified role to a single or multiple AppProjects:

```yaml
apiVersion: rbac-operator.argoproj-labs.io/v1alpha1
kind: ArgoCDProjectRoleBinding
metadata:
  name: test-project-role-binding
  namespace: test-ns
spec:
  argocdProjectRoleRef: 
    name: test-project-role
  subjects:
  - appProjectRef: test-appproject-1
    groups:
    - test-group-1
    - test-group-2
  - appProjectRef: test-appproject-2
    groups:
    - test-group-3
    - test-group-4
```

#### Create ArgoCDProjectRoles and ArgoCDProjectRoleBindings

Create a new ArgoCDProjectRole and ArgoCDProjectRoleBinding using the provided example. (Make sure that both CRs and AppProjects are created in the same Namespace)

```bash
kubectl create -f test-project-role.yaml
kubectl create -f test-project-role-binding.yaml
```

After the reconciliation a following role will be added to the specified AppProjects:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: test-appproject-1
  namespace: test-ns
spec:
  description: "Test AppProject 1 for ArgoCD's RBAC Operator"
  roles:
  ...
  - description: Test role for ArgoCD's AppProjects
    groups:
      - test-group-1
      - test-group-2
    name: test-project-role
    policies:
      - p, proj:test-appproject-1:test-project-role, clusters, get, *, allow
      - p, proj:test-appproject-1:test-project-role, clusters, watch, *, allow
      - p, proj:test-appproject-1:test-project-role, applications, get, *, allow
  ...
---
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: test-appproject-2
  namespace: test-ns
spec:
  description: "Test AppProject 2 for ArgoCD's RBAC Operator"
  roles:
  ...
  - description: Test role for ArgoCD's AppProjects
    groups:
      - test-group-3
      - test-group-4
    name: test-project-role
    policies:
      - p, proj:test-appproject-2:test-project-role, clusters, get, *, allow
      - p, proj:test-appproject-2:test-project-role, clusters, watch, *, allow
      - p, proj:test-appproject-2:test-project-role, applications, get, *, allow
  ...
```

#### Changes to ArgoCDProjectRoles and ArgoCDProjectRoleBindings

If changes there made to the CRs, they also will be reflected in referenced AppProjects:

- changes to `spec.rules` of ArgoCDProjectRole
  - will be patched to AppProject on next reconcile of ArgoCDProjectRoleBinding
- changes to `spec.subjects` of ArgoCDProjectRoleBindings
  - deletion of a subject, will delete the role in AppProject
  - change to subject will be reflected in AppProject on next reconcile

#### Delete ArgoCDProjectRoles and ArgoCDProjectRoleBindings

To delete a Role you can use `kubectl`

```bash
kubectl delete argocdprojectroles test-project-role
kubectl delete argocdprojectrolebindings test-project-role-binding
```

After the deletion of the Role or RoleBinding, the Role will also be deleted in AppProject.

## Roadmap

- [x] extend the operator with functionality to manage Argo CD AppProject RBAC
- [ ] achieve test coverage of >= 80% (current: ~75%)
- [ ] allow management for multi-instances set-up of Argo CD
