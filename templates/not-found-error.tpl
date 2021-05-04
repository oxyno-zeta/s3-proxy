{{- if contains "application/json" (.Request.Header.Get "Accept") -}}
{{ template "main.body.errorJsonBody" . }}
{{- else -}}
<!DOCTYPE html>
<html>
  <body>
    <h1>Not Found {{ .Request.URL.Path }}</h1>
  </body>
</html>
{{- end -}}
