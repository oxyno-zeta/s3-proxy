package bucket

import (
	"bytes"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Masterminds/sprig"
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
	handleInternalServerError func(rw http.ResponseWriter, err error, requestPath string, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig),
) (*RequestContext, error) {
	s3ctx, err := s3client.NewS3Context(tgt, logger)
	if err != nil {
		return nil, err
	}

	return &RequestContext{
		s3Context:                 s3ctx,
		logger:                    logger,
		bucketInstance:            tgt,
		mountPath:                 mountPath,
		requestPath:               requestPath,
		httpRW:                    httpRW,
		tplConfig:                 tplConfig,
		handleNotFound:            handleNotFound,
		handleInternalServerError: handleInternalServerError,
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
			rctx.handleInternalServerError(*rctx.httpRW, err, rctx.requestPath, rctx.logger, rctx.tplConfig)
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
				err := getFile(rctx, indexDocumentEntry.(*Entry).Key)
				// Check if error is a not found error
				if err == s3client.ErrNotFound {
					// Not found
					rctx.handleNotFound(*rctx.httpRW, rctx.requestPath, rctx.logger, rctx.tplConfig)
					return
				}
				(*rctx.logger).Errorln(err)
				rctx.handleInternalServerError(*rctx.httpRW, err, rctx.requestPath, rctx.logger, rctx.tplConfig)
				return
			}
		}

		// Load template
		tplFileName := filepath.Base(rctx.tplConfig.FolderList)
		tmpl, err := template.New(tplFileName).Funcs(sprig.HtmlFuncMap()).Funcs(s3ProxyFuncMap()).ParseFiles(rctx.tplConfig.FolderList)
		if err != nil {
			(*rctx.logger).Errorln(err)
			rctx.handleInternalServerError(*rctx.httpRW, err, rctx.requestPath, rctx.logger, rctx.tplConfig)
			return
		}
		// Create bucket list data for templating
		data := &bucketListingData{
			Entries:    entries,
			BucketName: rctx.bucketInstance.Bucket.Name,
			Name:       rctx.bucketInstance.Name,
			Path:       rctx.mountPath + rctx.requestPath,
		}
		// Generate template in buffer
		buf := &bytes.Buffer{}
		err = tmpl.Execute(buf, data)
		if err != nil {
			(*rctx.logger).Errorln(err)
			rctx.handleInternalServerError(*rctx.httpRW, err, rctx.requestPath, rctx.logger, rctx.tplConfig)
			return
		}
		(*rctx.httpRW).WriteHeader(200)
		// Set the header and write the buffer to the http.ResponseWriter
		(*rctx.httpRW).Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err = buf.WriteTo((*rctx.httpRW))
		if err != nil {
			(*rctx.logger).Errorln(err)
			rctx.handleInternalServerError(*rctx.httpRW, err, rctx.requestPath, rctx.logger, rctx.tplConfig)
			return
		}
		return
	}

	// Get object case
	err := getFile(rctx, key)
	if err != nil {
		// Check if error is a not found error
		if err == s3client.ErrNotFound {
			// Not found
			rctx.handleNotFound(*rctx.httpRW, rctx.requestPath, rctx.logger, rctx.tplConfig)
			return
		}
		(*rctx.logger).Errorln(err)
		rctx.handleInternalServerError(*rctx.httpRW, err, rctx.requestPath, rctx.logger, rctx.tplConfig)
		return
	}
}
