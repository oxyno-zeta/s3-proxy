<!DOCTYPE html>
<html>
  <body>
    <h1>Target buckets list</h1>
    <ul>
        {{- range .Targets }}
        <li><a href="/{{ .Name }}">{{ .Name }}</a></li>
        {{- end }}
    </ul>
  </body>
</html>
