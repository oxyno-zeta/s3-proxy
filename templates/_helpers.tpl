
{{- /* This function will allow to get user identifier. */ -}}
{{ define "main.userIdentifier" }}
{{- if .User }}
{{ .User.GetIdentifier }}
{{- end -}}
{{ end }}


{{ define "main.headers.contentType" }}
{{- if contains "application/json" (.Request.Header.Get "Accept") }}
application/json; charset=utf-8
{{- else }}
text/html; charset=utf-8
{{- end }}
{{ end }}
