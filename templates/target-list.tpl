{{- $root := . -}}
{{- if contains "application/json" (.Request.Header.Get "Accept") -}}
[
  {{- $mapKeys := keys .Targets -}}
  {{- $lastMapKey := last $mapKeys -}}
  {{- range $index, $key := $mapKeys -}}
  {{ $target := get $root.Targets $key }}
  {"name": "{{ js $key }}", "links": [
    {{- $pathLen := len $target.Mount.Path -}}
    {{- range $index2, $value2 := $target.Mount.Path -}}
      {{- if eq $target.Mount.Host "" -}}
      "{{ requestScheme $root.Request }}://{{ requestHost $root.Request }}{{ $value2 }}"
      {{- else -}}
      "{{ requestScheme $root.Request }}://{{ $target.Mount.Host }}{{ $value2 }}"
      {{- end -}}{{- if ne $index2 (sub $pathLen 1) -}},{{- end -}}
    {{- end -}}
  ]}{{- if ne $lastMapKey $key -}},{{- end -}}
  {{- end -}}
]
{{- else -}}
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
              <a href="{{ requestScheme $root.Request }}://{{ requestHost $root.Request }}{{ . }}">{{ requestScheme $root.Request }}://{{ $root.Request.Host }}{{ . }}</a>
              {{- else -}}
              <a href="{{ requestScheme $root.Request }}://{{ $target.Mount.Host }}{{ . }}">{{ requestScheme $root.Request }}://{{ $target.Mount.Host }}{{ . }}</a>
              {{- end -}}
            </li>
          </ul>
          {{- end }}
        </li>
        {{- end }}
    </ul>
  </body>
</html>
{{- end -}}
