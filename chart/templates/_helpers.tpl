{{/*
Expand the name of the chart.
*/}}
{{- define "language-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "language-operator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "language-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "language-operator.labels" -}}
helm.sh/chart: {{ include "language-operator.chart" . }}
{{ include "language-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "language-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "language-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "language-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "language-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the image name
*/}}
{{- define "language-operator.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
Create the metrics bind address
*/}}
{{- define "language-operator.metricsBindAddress" -}}
{{- printf ":%s" (toString .Values.service.metricsPort) }}
{{- end }}

{{/*
ServiceMonitor namespace
*/}}
{{- define "language-operator.serviceMonitor.namespace" -}}
{{- if .Values.monitoring.serviceMonitor.namespace }}
{{- .Values.monitoring.serviceMonitor.namespace }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
OpenTelemetry resource attributes
Builds a comma-separated list of key=value pairs for OTEL_RESOURCE_ATTRIBUTES
*/}}
{{- define "language-operator.otelResourceAttributes" -}}
{{- $attrs := list -}}
{{- if .Values.opentelemetry.resourceAttributes.environment -}}
{{- $attrs = append $attrs (printf "environment=%s" .Values.opentelemetry.resourceAttributes.environment) -}}
{{- end -}}
{{- if .Values.opentelemetry.resourceAttributes.cluster -}}
{{- $attrs = append $attrs (printf "k8s.cluster.name=%s" .Values.opentelemetry.resourceAttributes.cluster) -}}
{{- end -}}
{{- $attrs = append $attrs (printf "k8s.namespace.name=%s" .Release.Namespace) -}}
{{- range $key, $value := .Values.opentelemetry.resourceAttributes.custom -}}
{{- $attrs = append $attrs (printf "%s=%s" $key $value) -}}
{{- end -}}
{{- join "," $attrs -}}
{{- end }}
