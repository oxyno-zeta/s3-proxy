{{- if contains "application/json" (.Request.Header.Get "Accept") -}}
{{ template "main.body.jsonBody" . }}
{{- else -}}
<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>{{ .Error }}</p>
  </body>
</html>
{{- end -}}
