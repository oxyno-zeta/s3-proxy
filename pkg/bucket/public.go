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
func NewRequestContext(
	tgt *config.Target, tplConfig *config.TemplateConfig, logger *logrus.FieldLogger,
	mountPath string, requestPath string, httpRW *http.ResponseWriter,
	handleNotFound func(rw http.ResponseWriter, requestPath string, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig),
) (*RequestContext, error) {
	s3ctx, err := s3client.NewS3Context(tgt, logger)
	if err != nil {
		return nil, err
	}

	return &RequestContext{
		s3Context:      s3ctx,
		logger:         logger,
		bucketInstance: tgt,
		mountPath:      mountPath,
		requestPath:    requestPath,
		httpRW:         httpRW,
		tplConfig:      tplConfig,
		handleNotFound: handleNotFound,
	}, nil
}

// Proxy proxy requests
func (rctx *RequestContext) Proxy() {
	bucketRootPrefixKey := rctx.bucketInstance.Bucket.GetRootPrefix()
	// Key must begin by bucket prefix
	key := bucketRootPrefixKey
	// Trim first / if exists
	key += strings.TrimPrefix(rctx.requestPath, "/")
	// Check that the path ends with a / for a directory listing or the main path special case (empty path)
	if strings.HasSuffix(rctx.requestPath, "/") || rctx.requestPath == "" {
		// Directory listing case
		s3Entries, err := rctx.s3Context.ListFilesAndDirectories(key)
		if err != nil {
			(*rctx.logger).Errorln(err)
			// ! TODO Need to manage internal server error
			return
		}

		// Transform entries in entry with path objects
		entries := transformS3Entries(s3Entries, rctx, bucketRootPrefixKey)

		// Check if index document is activated
		if rctx.bucketInstance.IndexDocument != "" {
			// Search if the file is present
			indexDocumentEntry := funk.Find(entries, func(ent *Entry) bool {
				return rctx.bucketInstance.IndexDocument == ent.Name
			})
			// Check if index document entry exists
			if indexDocumentEntry != nil {
				// Get data
				getFile(rctx, indexDocumentEntry.(*Entry).Key)
			}
		}

		// Load template
		tplFileName := filepath.Base(rctx.tplConfig.FolderList)
		tmpl, err := template.New(tplFileName).Funcs(sprig.HtmlFuncMap()).Funcs(s3ProxyFuncMap()).ParseFiles(rctx.tplConfig.FolderList)
		if err != nil {
			// ! TODO Need to manage internal server error
			(*rctx.logger).Errorln(err)
			return
		}
		// Create bucket list data for templating
		data := &bucketListingData{
			Entries:    entries,
			BucketName: rctx.bucketInstance.Bucket.Name,
			Name:       rctx.bucketInstance.Name,
			Path:       rctx.mountPath + "/" + rctx.requestPath,
		}
		err = tmpl.Execute(*rctx.httpRW, data)

		// ! TODO Need to manage internal server error
		fmt.Println(err)
		return
	}

	// Get object case
	err := getFile(rctx, key)
	if err != nil {
		// Check if error is a not found error
		if err == s3client.ErrNotFound {
			// Not found
			rctx.handleNotFound(*rctx.httpRW, rctx.requestPath, rctx.logger, rctx.tplConfig)
		}
		// ! TODO Need to manage internal server error
		(*rctx.logger).Errorln(err)
	}
}

func transformS3Entries(s3Entries []*s3client.Entry, rctx *RequestContext, bucketRootPrefixKey string) []*Entry {
	entries := make([]*Entry, 0)
	for _, item := range s3Entries {
		entries = append(entries, &Entry{
			Type:         item.Type,
			ETag:         item.ETag,
			Name:         item.Name,
			LastModified: item.LastModified,
			Size:         item.Size,
			Key:          item.Key,
			Path:         rctx.mountPath + "/" + strings.TrimPrefix(item.Key, bucketRootPrefixKey),
		})
	}
	return entries
}

func getFile(brctx *RequestContext, key string) error {
	objOutput, err := brctx.s3Context.GetObject(key)
	if err != nil {
		return err
	}
	setHeadersFromObjectOutput(*brctx.httpRW, objOutput)
	io.Copy(*brctx.httpRW, *objOutput.Body)
	return nil
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
