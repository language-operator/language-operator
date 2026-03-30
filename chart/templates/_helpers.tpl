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
Dashboard helpers
*/}}
{{/*
Expand the name of the chart.
*/}}
{{- define "dashboard.name" -}}
{{- printf "%s-dashboard" (.Chart.Name) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "dashboard.fullname" -}}
{{- printf "%s-dashboard" .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "dashboard.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "dashboard.labels" -}}
helm.sh/chart: {{ include "dashboard.chart" . }}
{{ include "dashboard.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "dashboard.selectorLabels" -}}
app.kubernetes.io/name: {{ include "dashboard.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "dashboard.serviceAccountName" -}}
{{- if .Values.dashboard.serviceAccount.create }}
{{- default (include "dashboard.fullname" .) .Values.dashboard.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.dashboard.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Get PostgreSQL password
*/}}
{{- define "dashboard.postgresqlPassword" -}}
{{- if .Values.dashboard.postgresql.auth.password }}
{{- .Values.dashboard.postgresql.auth.password }}
{{- else }}
{{- $secretName := printf "%s-postgresql-password" (include "dashboard.fullname" .) }}
{{- $secret := lookup "v1" "Secret" .Release.Namespace $secretName }}
{{- if $secret }}
{{- index $secret.data "password" | b64dec }}
{{- else }}
{{- /* Use a deterministic password based on release name and namespace during initial install */ -}}
{{- $seed := printf "%s-%s-postgres" .Release.Name .Release.Namespace }}
{{- $seed | sha256sum | trunc 32 }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Get the database connection URL
*/}}
{{- define "dashboard.databaseUrl" -}}
{{- if .Values.dashboard.externalDatabase.enabled }}
{{- printf "postgresql://%s:%s@%s:%d/%s?schema=public" .Values.dashboard.externalDatabase.user .Values.dashboard.externalDatabase.password .Values.dashboard.externalDatabase.host (.Values.dashboard.externalDatabase.port | int) .Values.dashboard.externalDatabase.database }}
{{- else }}
{{- printf "postgresql://%s:%s@%s-postgresql:5432/%s?schema=public" .Values.dashboard.postgresql.auth.username (include "dashboard.postgresqlPassword" .) (include "dashboard.fullname" .) .Values.dashboard.postgresql.auth.database }}
{{- end }}
{{- end }}
