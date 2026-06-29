{{/*
Expand the name of the chart.
*/}}
{{- define "rootly-edge-connector.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "rootly-edge-connector.fullname" -}}
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
{{- define "rootly-edge-connector.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "rootly-edge-connector.labels" -}}
helm.sh/chart: {{ include "rootly-edge-connector.chart" . }}
{{ include "rootly-edge-connector.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "rootly-edge-connector.selectorLabels" -}}
app.kubernetes.io/name: {{ include "rootly-edge-connector.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use.
*/}}
{{- define "rootly-edge-connector.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "rootly-edge-connector.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Secret name for the API key.
*/}}
{{- define "rootly-edge-connector.secretName" -}}
{{- if .Values.rootly.existingSecret }}
{{- .Values.rootly.existingSecret }}
{{- else }}
{{- include "rootly-edge-connector.fullname" . }}
{{- end }}
{{- end }}

{{/*
Actions ConfigMap name.
*/}}
{{- define "rootly-edge-connector.actionsConfigMapName" -}}
{{- if .Values.existingActionsConfigMap }}
{{- .Values.existingActionsConfigMap }}
{{- else }}
{{- printf "%s-actions" (include "rootly-edge-connector.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Validate required values.
*/}}
{{- define "rootly-edge-connector.validateValues" -}}
{{- if and (not .Values.rootly.apiKey) (not .Values.rootly.existingSecret) }}
{{- fail "rootly.apiKey or rootly.existingSecret is required" }}
{{- end }}
{{- if and (not .Values.actionsYaml) (not .Values.existingActionsConfigMap) }}
{{- fail "actionsYaml or existingActionsConfigMap is required" }}
{{- end }}
{{- end }}
