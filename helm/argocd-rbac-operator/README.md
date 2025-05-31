# Argo CD RBAC Operator

[![Go Report Card](https://goreportcard.com/badge/github.com/argoproj-labs/argocd-rbac-operator)](https://goreportcard.com/report/github.com/argoproj-labs/argocd-rbac-operator)
[![go.mod Go version](https://img.shields.io/github/go-mod/go-version/argoproj-labs/argocd-rbac-operator)](https://github.com/argoproj-labs/argocd-rbac-operator)
[![GitHub Release](https://img.shields.io/github/v/release/argoproj-labs/argocd-rbac-operator)](https://github.com/argoproj-labs/argocd-rbac-operator/releases/tag/v0.1.9)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/argocd-rbac-operator)](https://artifacthub.io/packages/search?repo=argocd-rbac-operator)

Kubernetes Operator for Argo CD RBAC Management.

## Introduction

The Argo CD RBAC Operator provides a CRD based API for the RBAC management of Argo CD. It provides a structured and easy to use way to define RBAC policies. The Operator uses the CRs as a single source of truth for RBAC management and converts them into a policy string that is patched into the Argo CD RBAC ConfigMap.

## Installation

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

```bash
kubectl delete argocdrole.rbac-operator.argoproj-labs.io/test-role
kubectl delete argocdrolebinding.rbac-operator.argoproj-labs.io/test-role-binding
```

After the Resource is deleted, the policy string will be also deleted from the RBAC-CM.

### Change the Policy.CSV

To change the policy.csv you have to make changes in the `internal/controller/common/defaults.go` file.

### Deployment types

As for now only single Argo CD deployment type is supported. The default Argo CD namespace is defined as `argocd`, to change that you have to make a change in `internal/controller/common/values.go`.

## General parameters

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| additionalLabels | object | `{}` |  |
| argocd.cmName | string | `"argocd-rbac-cm"` |  |
| argocd.namespace | string | `"argocd"` |  |
| containerSecurityContext.allowPrivilegeEscalation | bool | `false` |  |
| containerSecurityContext.capabilities.drop[0] | string | `"ALL"` |  |
| containerSecurityContext.readOnlyRootFilesystem | bool | `true` |  |
| containerSecurityContext.runAsNonRoot | bool | `true` |  |
| containerSecurityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"quay.io/argoprojlabs/argocd-rbac-operator"` |  |
| image.tag | string | `"v0.1.6"` |  |
| imagePullSecrets | list | `[]` |  |
| livenessProbe.httpGet.path | string | `"/healthz"` |  |
| livenessProbe.httpGet.port | int | `8081` |  |
| livenessProbe.initialDelaySeconds | int | `15` |  |
| livenessProbe.periodSeconds | int | `20` |  |
| namespace.create | bool | `true` |  |
| namespace.nameOverride | string | `""` |  |
| nodeSelector | object | `{}` |  |
| readinessProbe.httpGet.path | string | `"/readyz"` |  |
| readinessProbe.httpGet.port | int | `8081` |  |
| readinessProbe.initialDelaySeconds | int | `5` |  |
| readinessProbe.periodSeconds | int | `10` |  |
| replicaCount | int | `1` |  |
| resources.limits.cpu | string | `"500m"` |  |
| resources.limits.memory | string | `"128Mi"` |  |
| resources.requests.cpu | string | `"10m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| securityContext.runAsNonRoot | bool | `true` |  |
| securityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| serviceAccountAnnotations | list | `[]` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs](https://github.com/norwoodj/helm-docs)
