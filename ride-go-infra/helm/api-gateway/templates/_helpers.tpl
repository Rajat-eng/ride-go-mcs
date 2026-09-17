{{- define "api-gateway.name" -}}
{{- .Chart.Name -}}
{{- end -}}

{{- define "api-gateway.fullname" -}}
{{- .Release.Name -}}
{{- end -}}

{{- define "api-gateway.labels" -}}
app: {{ include "api-gateway.fullname" . }}
chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}
