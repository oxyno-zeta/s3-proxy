<!DOCTYPE html>
<html>
  <body>
    <h1>Target buckets list</h1>
    <ul>
        {{- range $key, $target := .Targets }}
        <li>{{ $target.Name }}:
          {{- range $target.Mount.Path }}
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
