{{- $root := . -}}
{{- if contains "application/json" (.Request.Header.Get "Accept") -}}
[
  {{- $maxLen := len $root.Entries -}}
  {{- range $index, $entry := $root.Entries -}}
  {
    "name": "{{ js $entry.Name }}",
    "etag": "{{ js $entry.ETag }}",
    "type": "{{ js $entry.Type }}",
    "size": {{ $entry.Size }},
    "path": "{{ js $entry.Path }}",
    "lastModified": "{{ $entry.LastModified | date "2006-01-02T15:04:05Z07:00" }}"
  }{{- if ne $index (sub $maxLen 1) -}},{{- end -}}
  {{- end -}}
]
{{- else -}}
<!DOCTYPE html>
<html>
  <body>
    <h1>Index of {{ .Request.URL.Path }}</h1>
    <table style="width:100%">
        <thead>
            <tr>
                <th style="border-right:1px solid black;text-align:start">Entry</th>
                <th style="border-right:1px solid black;text-align:start">Size</th>
                <th style="border-right:1px solid black;text-align:start">Last modified</th>
            </tr>
        </thead>
        <tbody style="border-top:1px solid black">
          <tr>
            <td style="border-right:1px solid black;padding: 0 5px"><a href="..">..</a></td>
            <td style="border-right:1px solid black;padding: 0 5px"> - </td>
            <td style="padding: 0 5px"> - </td>
          </tr>
        {{- range .Entries }}
          <tr>
              <td style="border-right:1px solid black;padding: 0 5px"><a href="{{ .Path }}">{{ .Name }}</a></td>
              <td style="border-right:1px solid black;padding: 0 5px">{{- if eq .Type "FOLDER" -}} - {{- else -}}{{ .Size | humanSize }}{{- end -}}</td>
              <td style="padding: 0 5px">{{ .LastModified }}</td>
          </tr>
        {{- end }}
        </tbody>
    </table>
  </body>
</html>
{{- end -}}
