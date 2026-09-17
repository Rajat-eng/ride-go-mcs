{{- define "payment-service.name" -}}
{{- .Chart.Name -}}
{{- end -}}

{{- define "payment-service.fullname" -}}
{{- .Release.Name -}}
{{- end -}}

{{- define "payment-service.labels" -}}
app: {{ include "payment-service.fullname" . }}
chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}
