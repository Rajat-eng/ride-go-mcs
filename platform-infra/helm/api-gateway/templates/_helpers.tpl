{{- define "api-gateway.name" -}}
{{- .Chart.Name -}}
{{- end -}}

{{- define "api-gateway.fullname" -}}
{{- .Release.Name -}}
{{- end -}}
