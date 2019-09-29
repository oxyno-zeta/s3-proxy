package bucket

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Masterminds/sprig"
	"github.com/dustin/go-humanize"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3client"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

func NewBucketRequestContext(binst *config.BucketInstance, tplConfig *config.TemplateConfig, logger *logrus.FieldLogger, mountPath string, requestPath string, httpRW *http.ResponseWriter) (*BucketRequestContext, error) {
	s3ctx, err := s3client.NewS3Context(binst, logger)
	if err != nil {
		return nil, err
	}

	return &BucketRequestContext{
		s3Context:      s3ctx,
		logger:         logger,
		bucketInstance: binst,
		mountPath:      mountPath,
		requestPath:    requestPath,
		httpRW:         httpRW,
		tplConfig:      tplConfig,
	}, nil
}

func (brctx *BucketRequestContext) Proxy() {
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
		data := &BucketListingData{
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

func getFile(brctx *BucketRequestContext, key string) {
	objectIOReadCloser, err := brctx.s3Context.GetObject(key)
	if err != nil {
		// Check if it a not found error
		if err == s3client.ErrNotFound {
			(*brctx.httpRW).WriteHeader(404)
			(*brctx.httpRW).Write([]byte("Not found"))
			return
		}
	}
	io.Copy(*brctx.httpRW, *objectIOReadCloser)
}

func s3ProxyFuncMap() template.FuncMap {
	funcMap := make(map[string]interface{}, 1)
	funcMap["humanSize"] = func(fmt int64) string {
		return humanize.Bytes(uint64(fmt))
	}
	return template.FuncMap(funcMap)
}
