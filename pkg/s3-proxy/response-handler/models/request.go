package models

import (
	"net/http"
	"net/url"
)

type LightSanitizedRequest struct {
	URL              *url.URL
	Header           http.Header
	Trailer          http.Header
	RemoteAddr       string
	Method           string
	Proto            string
	Pattern          string
	RequestURI       string
	Host             string
	TransferEncoding []string
	ProtoMajor       int
	ContentLength    int64
	ProtoMinor       int
}
