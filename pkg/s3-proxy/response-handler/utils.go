package responsehandler

import (
	"bytes"
	"context"
	"html/template"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/dustin/go-humanize"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

func (h *handler) loadTemplateContent(
	ctx context.Context,
	loadS3FileContent func(ctx context.Context, path string) (string, error),
	item *config.TargetTemplateConfigItem,
) (string, error) {
	// Check if it is in bucket and if load from S3 function exists
	if item.InBucket && loadS3FileContent != nil {
		// Try to get file from bucket
		return loadS3FileContent(ctx, item.Path)
	}

	// Not in bucket, need to load from FS
	return loadLocalFileContent(item.Path)
}

// templateExecution will execute template with values and interpret response as html content.
func (h *handler) templateExecution(tplString string, data interface{}, status int) error {
	// Set status code
	h.res.WriteHeader(status)
	// Set the header and write the buffer to the http.ResponseWriter
	h.res.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Load template from string
	tmpl, err := template.
		New("template-string-loaded").
		Funcs(sprig.HtmlFuncMap()).
		Funcs(s3ProxyFuncMap()).
		Parse(tplString)
	// Check if error exists
	if err != nil {
		return err
	}

	// Generate template in buffer
	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, data)
	// Check if error exists
	if err != nil {
		return err
	}

	_, err = buf.WriteTo(h.res)
	// Check if error exists
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) generateRequestData() *requestData {
	// Create url data
	urlDt := &urlData{
		Scheme:          h.req.URL.Scheme,
		Opaque:          h.req.URL.Opaque,
		Host:            h.req.URL.Host,
		Path:            h.req.URL.Path,
		RawPath:         h.req.URL.RawPath,
		ForceQuery:      h.req.URL.ForceQuery,
		RawQuery:        h.req.URL.RawQuery,
		Fragment:        h.req.URL.Fragment,
		RawFragment:     h.req.URL.RawFragment,
		IsAbs:           h.req.URL.IsAbs(),
		EscapedFragment: h.req.URL.EscapedFragment(),
		EscapedPath:     h.req.URL.EscapedPath(),
		Port:            h.req.URL.Port(),
		RequestURI:      h.req.URL.RequestURI(),
		String:          h.req.URL.String(),
	}

	// Create query values structure
	qs := map[string]interface{}{}

	// Loop over query values
	for k, v := range h.req.URL.Query() {
		// Store
		qs[k] = v
	}

	// Store query values
	urlDt.QueryParams = qs

	// Create result data
	res := &requestData{
		Method:        h.req.Method,
		Proto:         h.req.Proto,
		ProtoMajor:    h.req.ProtoMajor,
		ProtoMinor:    h.req.ProtoMinor,
		ContentLength: h.req.ContentLength,
		Close:         h.req.Close,
		Host:          h.req.Host,
		RemoteAddr:    h.req.RemoteAddr,
		RequestURI:    h.req.RequestURI,
		URL:           urlDt,
	}

	// Create header structure
	hds := map[string]interface{}{}

	// Loop over headers
	for k, v := range h.req.Header {
		// Store
		hds[k] = v
	}

	// Store headers
	res.Headers = hds

	// Return
	return res
}

func loadLocalFileContent(path string) (string, error) {
	// Read file from file path
	by, err := ioutil.ReadFile(path)
	// Check if error exists
	if err != nil {
		return "", err
	}

	return string(by), nil
}

func s3ProxyFuncMap() template.FuncMap {
	funcMap := make(map[string]interface{}, 1)
	funcMap["humanSize"] = func(fmt int64) string {
		return humanize.Bytes(uint64(fmt))
	}
	// Return result
	return template.FuncMap(funcMap)
}

func setHeadersFromObjectOutput(w http.ResponseWriter, obj *StreamInput) {
	setStrHeader(w, "Cache-Control", obj.CacheControl)
	setStrHeader(w, "Expires", obj.Expires)
	setStrHeader(w, "Content-Disposition", obj.ContentDisposition)
	setStrHeader(w, "Content-Encoding", obj.ContentEncoding)
	setStrHeader(w, "Content-Language", obj.ContentLanguage)
	setIntHeader(w, "Content-Length", obj.ContentLength)
	setStrHeader(w, "Content-Range", obj.ContentRange)
	setStrHeader(w, "Content-Type", obj.ContentType)
	setStrHeader(w, "ETag", obj.ETag)
	setTimeHeader(w, "Last-Modified", obj.LastModified)

	httpStatus := determineHTTPStatus(obj)
	w.WriteHeader(httpStatus)
}

func determineHTTPStatus(obj *StreamInput) int {
	// Set default http status to 200 OK
	httpStatus := http.StatusOK
	contentRangeIsGiven := len(obj.ContentRange) > 0
	// Check if content will be partial
	if contentRangeIsGiven {
		httpStatus = http.StatusPartialContent
		if totalFileSizeEqualToContentRange(obj) {
			httpStatus = http.StatusOK
		}
	}
	// Return status code
	return httpStatus
}

func totalFileSizeEqualToContentRange(obj *StreamInput) bool {
	totalSizeIsEqualToContentRange := false
	// Calculate total file size
	totalSize, err := strconv.ParseInt(getFileSizeAsString(obj), 10, 64)
	if err == nil {
		if totalSize == (obj.ContentLength) {
			totalSizeIsEqualToContentRange = true
		}
	}
	// Return result
	return totalSizeIsEqualToContentRange
}

/**
See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Range
*/
func getFileSizeAsString(obj *StreamInput) string {
	s := strings.Split(obj.ContentRange, "/")
	totalSizeString := s[1]
	totalSizeString = strings.TrimSpace(totalSizeString)
	// Return result
	return totalSizeString
}

func setStrHeader(w http.ResponseWriter, key string, value string) {
	if len(value) > 0 {
		w.Header().Add(key, value)
	}
}

func setIntHeader(w http.ResponseWriter, key string, value int64) {
	if value > 0 {
		w.Header().Add(key, strconv.FormatInt(value, 10))
	}
}

func setTimeHeader(w http.ResponseWriter, key string, value time.Time) {
	if !reflect.DeepEqual(value, time.Time{}) {
		w.Header().Add(key, value.UTC().Format(http.TimeFormat))
	}
}
