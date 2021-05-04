{{- if contains "application/json" (.Request.Header.Get "Accept") -}}
{{ template "main.body.errorJsonBody" . }}
{{- else -}}
<!DOCTYPE html>
<html>
  <body>
    <h1>Unauthorized</h1>
    <p>{{ .Error }}</p>
  </body>
</html>
{{- end -}}
