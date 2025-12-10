{{/*
Expand the name of the chart.
*/}}
{{- define "dashboard.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "dashboard.fullname" -}}
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
{{- if .Values.serviceAccount.create }}
{{- default (include "dashboard.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Get PostgreSQL password
*/}}
{{- define "dashboard.postgresqlPassword" -}}
{{- if .Values.postgresql.auth.password }}
{{- .Values.postgresql.auth.password }}
{{- else }}
{{- $secret := lookup "v1" "Secret" .Release.Namespace (printf "%s-postgresql" (include "dashboard.fullname" .)) }}
{{- if $secret }}
{{- index $secret.data "password" | b64dec }}
{{- else }}
{{- randAlphaNum 32 }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Get the database connection URL
*/}}
{{- define "dashboard.databaseUrl" -}}
{{- if .Values.externalDatabase.enabled }}
{{- printf "postgresql://%s:%s@%s:%d/%s?schema=public" .Values.externalDatabase.user .Values.externalDatabase.password .Values.externalDatabase.host (.Values.externalDatabase.port | int) .Values.externalDatabase.database }}
{{- else }}
{{- printf "postgresql://%s:%s@%s-postgresql:5432/%s?schema=public" .Values.postgresql.auth.username (include "dashboard.postgresqlPassword" .) (include "dashboard.fullname" .) .Values.postgresql.auth.database }}
{{- end }}
{{- end }}
