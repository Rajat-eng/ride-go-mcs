{{- define "dlq-worker.name" -}}
{{- .Chart.Name -}}
{{- end -}}

{{- define "dlq-worker.fullname" -}}
{{- .Release.Name -}}
{{- end -}}

{{- define "dlq-worker.labels" -}}
app: {{ include "dlq-worker.fullname" . }}
chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}
