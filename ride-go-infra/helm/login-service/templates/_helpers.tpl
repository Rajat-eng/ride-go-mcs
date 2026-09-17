{{- define "login-service.name" -}}
{{- .Chart.Name -}}
{{- end -}}

{{- define "login-service.fullname" -}}
{{- .Release.Name -}}
{{- end -}}

{{- define "login-service.labels" -}}
app: {{ include "login-service.fullname" . }}
chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}
