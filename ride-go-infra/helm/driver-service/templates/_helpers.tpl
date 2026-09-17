{{- define "driver-service.name" -}}
{{- .Chart.Name -}}
{{- end -}}

{{- define "driver-service.fullname" -}}
{{- .Release.Name -}}
{{- end -}}

{{- define "driver-service.labels" -}}
app: {{ include "driver-service.fullname" . }}
chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}
