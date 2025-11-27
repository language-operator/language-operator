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
{{- if .Values.opentelemetry.collector.resourceAttributes.environment -}}
{{- $attrs = append $attrs (printf "environment=%s" .Values.opentelemetry.collector.resourceAttributes.environment) -}}
{{- end -}}
{{- if .Values.opentelemetry.collector.resourceAttributes.cluster -}}
{{- $attrs = append $attrs (printf "k8s.cluster.name=%s" .Values.opentelemetry.collector.resourceAttributes.cluster) -}}
{{- end -}}
{{- $attrs = append $attrs (printf "k8s.namespace.name=%s" .Release.Namespace) -}}
{{- range $key, $value := .Values.opentelemetry.collector.resourceAttributes.custom -}}
{{- $attrs = append $attrs (printf "%s=%s" $key $value) -}}
{{- end -}}
{{- join "," $attrs -}}
{{- end }}

{{/*
Telemetry adapter enabled check
Returns true if telemetry adapter is enabled and properly configured
*/}}
{{- define "language-operator.telemetryAdapter.enabled" -}}
{{- if and .Values.telemetry.queryBackend.enabled .Values.telemetry.queryBackend.endpoint -}}
{{- true -}}
{{- else -}}
{{- false -}}
{{- end -}}
{{- end }}

{{/*
Telemetry adapter API key value
Returns the API key value, either direct or from secret reference
*/}}
{{- define "language-operator.telemetryAdapter.apiKey" -}}
{{- if .Values.telemetry.queryBackend.auth.apiKeySecret.name -}}
{{- printf "secretKeyRef:\n        name: %s\n        key: %s" .Values.telemetry.queryBackend.auth.apiKeySecret.name .Values.telemetry.queryBackend.auth.apiKeySecret.key -}}
{{- else if .Values.telemetry.queryBackend.auth.apiKey -}}
{{- .Values.telemetry.queryBackend.auth.apiKey | quote -}}
{{- else -}}
{{- "" -}}
{{- end -}}
{{- end }}

{{/*
Telemetry adapter type validation
Returns the adapter type if valid, otherwise "noop"
*/}}
{{- define "language-operator.telemetryAdapter.type" -}}
{{- $validTypes := list "signoz" "jaeger" "tempo" "noop" -}}
{{- if has .Values.telemetry.queryBackend.type $validTypes -}}
{{- .Values.telemetry.queryBackend.type -}}
{{- else -}}
{{- "noop" -}}
{{- end -}}
{{- end }}
