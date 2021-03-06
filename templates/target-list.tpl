<!DOCTYPE html>
<html>
  <body>
    <h1>Target buckets list</h1>
    <ul>
        {{- range .Targets }}
        <li>{{ .Name }}:
          {{- $target := . -}}
          {{- range .Mount.Path }}
          <ul>
            <li>
              {{- if eq $target.Mount.Host "" -}}
              <a href="{{ . }}">{{ . }}</a>
              {{- else -}}
              <a href="http://{{ $target.Mount.Host }}{{ . }}">http://{{ $target.Mount.Host }}{{ . }}</a>
              {{- end -}}
            </li>
          </ul>
          {{- end }}
        </li>
        {{- end }}
    </ul>
  </body>
</html>
