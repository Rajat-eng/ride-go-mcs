{{- define "ws-gateway.name" -}}
{{- .Chart.Name -}}
{{- end -}}

{{- define "ws-gateway.fullname" -}}
{{- .Release.Name -}}
{{- end -}}

{{- define "ws-gateway.labels" -}}
app: {{ include "ws-gateway.fullname" . }}
chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}
