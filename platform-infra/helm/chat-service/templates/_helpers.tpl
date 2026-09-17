{{- define "chat-service.fullname" -}}
{{- .Release.Name -}}
{{- end -}}

{{- define "chat-service.labels" -}}
app: {{ include "chat-service.fullname" . }}
{{- end -}}
