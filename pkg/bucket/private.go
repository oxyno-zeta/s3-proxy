package bucket

import (
	"html/template"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3client"
)

func transformS3Entries(s3Entries []*s3client.Entry, rctx *RequestContext, bucketRootPrefixKey string) []*Entry {
	// Prepare result
	entries := make([]*Entry, 0)
	// Loop over s3 entries
	for _, item := range s3Entries {
		entries = append(entries, &Entry{
			Type:         item.Type,
			ETag:         item.ETag,
			Name:         item.Name,
			LastModified: item.LastModified,
			Size:         item.Size,
			Key:          item.Key,
			Path:         rctx.mountPath + strings.TrimPrefix(item.Key, bucketRootPrefixKey),
		})
	}
	// Return result
	return entries
}

func getFile(brctx *RequestContext, key string) error {
	// Get object from s3
	objOutput, err := brctx.s3Context.GetObject(key)
	if err != nil {
		return err
	}
	// Set headers from object
	setHeadersFromObjectOutput(*brctx.httpRW, objOutput)
	// Copy data stream to output stream
	_, err = io.Copy(*brctx.httpRW, *objOutput.Body)
	// Return potential error
	return err
}

func setHeadersFromObjectOutput(w http.ResponseWriter, obj *s3client.ObjectOutput) {
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

func determineHTTPStatus(obj *s3client.ObjectOutput) int {
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

func totalFileSizeEqualToContentRange(obj *s3client.ObjectOutput) bool {
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
func getFileSizeAsString(obj *s3client.ObjectOutput) string {
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

func s3ProxyFuncMap() template.FuncMap {
	funcMap := make(map[string]interface{}, 1)
	funcMap["humanSize"] = func(fmt int64) string {
		return humanize.Bytes(uint64(fmt))
	}
	// Return result
	return template.FuncMap(funcMap)
}
