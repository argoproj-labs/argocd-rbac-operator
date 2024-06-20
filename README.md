# Argo CD RBAC Operator

Kubernetes Operator for Argo CD RBAC Management.

## Introduction

The Argo CD RBAC Operator provides a CRD based API for the RBAC management of Argo CD. It provides a structured and easy to use way to define RBAC policies. The Operator uses the CRs as a single source of truth for RBAC management and converts them into a policy string that is patched into the Argo CD RBAC ConfigMap.

## Installation

### Get the Repository

Clone the Argo CD RBAC Operator repository.

```
git clone https://github.com/argoproj-labs/argocd-rbac-operator.git
cd argocd-rbac-operator
```

### Namespace

By default, the operator is installed into the `argocd-rbac-operator-system` namespace. To modify this, update the value of the namespace specified in the `config/default/kustomization.yaml` file.

### Deploy Operator

Deploy the operator. This will create all the necessary resources, including the namespace. For running the make command you need to install go-lang package on your system.

```
make deploy
```

The operator pod should start and enter a Running state after a few seconds.

```
kubectl get pods -n argocd-rbac-operator-system
```

## Usage

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

### Create

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

### Delete

To delete a Role you can use `kubectl`
```
kubectl delete argocdrole.rbac-operator.argoproj-labs.io/test-role
kubectl delete argocdrolebinding.rbac-operator.argoproj-labs.io/test-role-binding
```
After the Resource is deleted, the policy string will be also deleted from the RBAC-CM.

### Change the Scope, Default Role or Policy.CSV

To change the scope, default role or policy.csv you have to make changes in the `internal/controller/common/defaults.go` file.

### Deployment types

As for now only single Argo CD deployment type is supported. The default Argo CD namespace is defined as `argocd`, to change that you have to make a change in `internal/controller/common/values.go`.

## Roadmap

- extend the operator with functionality to manage Argo CD AppProject RBAC
- achieve test coverage of >= 80%
- allow management for multi-instances set-up of Argo CD