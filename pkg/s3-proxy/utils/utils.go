package utils

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/dustin/go-humanize"
)

func ExecuteTemplate(tplString string, data interface{}) (*bytes.Buffer, error) {
	// Load template from string
	tmpl, err := template.
		New("template-string-loaded").
		Funcs(sprig.TxtFuncMap()).
		Funcs(s3ProxyFuncMap()).
		Parse(tplString)
	// Check if error exists
	if err != nil {
		return nil, err
	}

	// Generate template in buffer
	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, data)
	// Check if error exists
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func s3ProxyFuncMap() template.FuncMap {
	// Result
	funcMap := map[string]interface{}{}
	// Add human size function
	funcMap["humanSize"] = func(fmt int64) string {
		return humanize.Bytes(uint64(fmt))
	}
	// Add request URI function
	funcMap["requestURI"] = GetRequestURI
	// Add request scheme function
	funcMap["requestScheme"] = GetRequestScheme
	// Add request host function
	funcMap["requestHost"] = GetRequestHost

	// Return result
	return template.FuncMap(funcMap)
}

// ClientIP will return client ip from request.
func ClientIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}

	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}

	return IPAddress
}

func GetRequestScheme(r *http.Request) string {
	// Default
	scheme := "http"

	// Get forwarded scheme
	fwdScheme := r.Header.Get("X-Forwarded-Proto")

	// Check if it is https
	if r.TLS != nil || fwdScheme == "https" {
		scheme = "https"
	}

	return scheme
}

func GetRequestURI(r *http.Request) string {
	scheme := GetRequestScheme(r)

	return fmt.Sprintf("%s://%s%s", scheme, GetRequestHost(r), r.URL.RequestURI())
}

func GetRequestHost(r *http.Request) string {
	// not standard, but most popular
	host := r.Header.Get("X-Forwarded-Host")
	if host != "" {
		return host
	}

	// RFC 7239
	host = r.Header.Get("Forwarded")
	_, _, host = parseForwarded(host)

	if host != "" {
		return host
	}

	// if all else fails fall back to request host
	host = r.Host

	return host
}

func parseForwarded(forwarded string) (addr, proto, host string) {
	if forwarded == "" {
		return
	}

	for _, forwardedPair := range strings.Split(forwarded, ";") {
		if tv := strings.SplitN(forwardedPair, "=", 2); len(tv) == 2 { // nolint: gomnd // No constant for that
			token, value := tv[0], tv[1]
			token = strings.TrimSpace(token)
			value = strings.TrimSpace(strings.Trim(value, `"`))

			switch strings.ToLower(token) {
			case "for":
				addr = value
			case "proto":
				proto = value
			case "host":
				host = value
			}
		}
	}

	return
}
