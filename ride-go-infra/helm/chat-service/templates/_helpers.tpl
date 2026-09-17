{{- define "chat-service.name" -}}
{{- .Chart.Name -}}
{{- end -}}

{{- define "chat-service.fullname" -}}
{{- .Release.Name -}}
{{- end -}}

{{- define "chat-service.labels" -}}
app: {{ include "chat-service.fullname" . }}
chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}
