{{- if contains "application/json" (.Request.Header.Get "Accept") -}}
{{ template "main.body.jsonBody" . }}
{{- else -}}
<!DOCTYPE html>
<html>
  <body>
    <h1>Bad Request</h1>
    <p>{{ .Error }}</p>
  </body>
</html>
{{- end -}}
