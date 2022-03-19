
{{- /* This function will allow to get user identifier. */ -}}
{{- define "main.userIdentifier" -}}
{{- if .User -}}
{{- .User.GetIdentifier -}}
{{- end -}}
{{- end -}}


{{- /* This function will allow to get the content type header from "Accept" header */ -}}
{{- define "main.headers.contentType" -}}
{{- if contains "application/json" (.Request.Header.Get "Accept") -}}
application/json; charset=utf-8
{{- else -}}
text/html; charset=utf-8
{{- end -}}
{{- end -}}

{{- /* This will forge the json output of an error */ -}}
{{- define "main.body.errorJsonBody" -}}
{"error": {{ .Error.Error | toJson }}}
{{- end -}}
