{{- define "trip-service.name" -}}
{{- .Chart.Name -}}
{{- end -}}

{{- define "trip-service.fullname" -}}
{{- .Release.Name -}}
{{- end -}}

{{- define "trip-service.labels" -}}
app: {{ include "trip-service.fullname" . }}
chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}
