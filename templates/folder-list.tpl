<!DOCTYPE html>
<html>
  <body>
    <h1>Index of {{ .Path }}</h1>
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
