{{/*
Expand the name of the chart.
*/}}
{{- define "argocd-rbac-operator.name" -}}
{{- default .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "argocd-rbac-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "argocd-rbac-operator.labels" -}}
helm.sh/chart: {{ include "argocd-rbac-operator.chart" . }}
{{ include "argocd-rbac-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- range $key, $val :=  .Values.additionalLabels }}
{{ $key }}: {{ $val | quote }}
{{- end }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "argocd-rbac-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "argocd-rbac-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the namespace
*/}}
{{- define "argocd-rbac-operator.namespace" -}}
{{- if .Values.namespace.nameOverride }}
{{- .Values.namespace.nameOverride | trimSuffix "-" }}
{{- else }}
{{- printf "%s-system" .Chart.Name | trimSuffix "-" }}
{{- end }}
{{- end }}


{{/*
Create the name of the service account
*/}}
{{- define "argocd-rbac-operator.serviceAccountName" -}}
{{- printf "%s-controller-manager" .Chart.Name | trimSuffix "-" }}
{{- end }}

