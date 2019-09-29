package bucket

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/dustin/go-humanize"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3client"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

// NewRequestContext New request context
func NewRequestContext(binst *config.BucketInstance, tplConfig *config.TemplateConfig, logger *logrus.FieldLogger, mountPath string, requestPath string, httpRW *http.ResponseWriter) (*RequestContext, error) {
	s3ctx, err := s3client.NewS3Context(binst, logger)
	if err != nil {
		return nil, err
	}

	return &RequestContext{
		s3Context:      s3ctx,
		logger:         logger,
		bucketInstance: binst,
		mountPath:      mountPath,
		requestPath:    requestPath,
		httpRW:         httpRW,
		tplConfig:      tplConfig,
	}, nil
}

// Proxy proxy requests
func (brctx *RequestContext) Proxy() {
	// Trim first / if exists
	key := strings.TrimPrefix(brctx.requestPath, "/")
	// Check that the path ends with a / for a directory listing or the main path special case (empty path)
	if strings.HasSuffix(brctx.requestPath, "/") || brctx.requestPath == "" {
		// Directory listing case
		entries, err := brctx.s3Context.ListFilesAndDirectories(key)
		if err != nil {
			(*brctx.logger).Errorln(err)
			// ! TODO Need to manage internal server error
			return
		}

		// Check if index document is activated
		if brctx.bucketInstance.IndexDocument != "" {
			// Search if the file is present
			indexDocumentEntry := funk.Find(entries, func(ent *s3client.Entry) bool {
				return brctx.bucketInstance.IndexDocument == ent.Name
			})
			// Check if index document entry exists
			if indexDocumentEntry != nil {
				// Get data
				getFile(brctx, indexDocumentEntry.(*s3client.Entry).Key)
			}
		}

		// Load template
		tplFileName := filepath.Base(brctx.tplConfig.FolderList)
		tmpl, err := template.New(tplFileName).Funcs(sprig.HtmlFuncMap()).Funcs(s3ProxyFuncMap()).ParseFiles(brctx.tplConfig.FolderList)
		if err != nil {
			// ! TODO Need to manage internal server error
			(*brctx.logger).Errorln(err)
			return
		}
		// Transform entries in entry with path objects
		entryPathList := make([]*EntryPath, 0)
		for _, item := range entries {
			entryPathList = append(entryPathList, &EntryPath{
				Entry: item,
				Path:  brctx.mountPath + "/" + item.Key,
			})
		}
		// Create bucket list data for templating
		data := &bucketListingData{
			Entries:    entryPathList,
			BucketName: brctx.bucketInstance.Bucket.Name,
			Name:       brctx.bucketInstance.Name,
			Path:       brctx.mountPath + "/" + brctx.requestPath,
		}
		err = tmpl.Execute(*brctx.httpRW, data)

		// ! TODO Need to manage internal server error
		fmt.Println(err)
		return
	}

	// Get object case
	getFile(brctx, key)
}

func getFile(brctx *RequestContext, key string) {
	objOutput, err := brctx.s3Context.GetObject(key)
	if err != nil {
		// Check if it a not found error
		if err == s3client.ErrNotFound {
			// ! TODO Need to manage this via templates
			(*brctx.httpRW).WriteHeader(404)
			(*brctx.httpRW).Write([]byte("Not found"))
			return
		}
	}
	setHeadersFromObjectOutput(*brctx.httpRW, objOutput)
	io.Copy(*brctx.httpRW, *objOutput.Body)
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
	httpStatus := http.StatusOK
	contentRangeIsGiven := len(obj.ContentRange) > 0
	if contentRangeIsGiven {
		httpStatus = http.StatusPartialContent
		if totalFileSizeEqualToContentRange(obj) {
			httpStatus = http.StatusOK
		}
	}
	return httpStatus
}

func totalFileSizeEqualToContentRange(obj *s3client.ObjectOutput) bool {
	totalSizeIsEqualToContentRange := false
	totalSize, err := strconv.ParseInt(getFileSizeAsString(obj), 10, 64)
	if err == nil {
		if totalSize == (obj.ContentLength) {
			totalSizeIsEqualToContentRange = true
		}
	}
	return totalSizeIsEqualToContentRange
}

/**
See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Range
*/
func getFileSizeAsString(obj *s3client.ObjectOutput) string {
	s := strings.Split(obj.ContentRange, "/")
	totalSizeString := s[1]
	totalSizeString = strings.TrimSpace(totalSizeString)
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
	return template.FuncMap(funcMap)
}
