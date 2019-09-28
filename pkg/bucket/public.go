package bucket

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/Masterminds/sprig"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3client"
	"github.com/sirupsen/logrus"
)

func NewBucketRequestContext(binst *config.BucketInstance, logger *logrus.FieldLogger, mountPath string, requestPath string, httpRW *http.ResponseWriter) (*BucketRequestContext, error) {
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
			return
		}
		// Load template
		tmpl, err := template.New("list.tpl").Funcs(sprig.HtmlFuncMap()).ParseFiles("templates/list.tpl")
		if err != nil {
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

		fmt.Println(err)
		fmt.Println(entries)
		return
	}

	// Get object case
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
