
{{- /* This function will allow to get user identifier. */ -}}
{{ define "userIdentifier" }}
{{- if .User }}
{{ .User.GetIdentifier }}
{{- end -}}
{{ end }}
