package bucket

import (
	"bytes"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3client"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

// requestContext Bucket request context
type requestContext struct {
	s3Context                 s3client.Client
	logger                    logrus.FieldLogger
	bucketInstance            *config.Target
	tplConfig                 *config.TemplateConfig
	mountPath                 string
	requestPath               string
	httpRW                    http.ResponseWriter
	handleNotFound            func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig)
	handleInternalServerError func(rw http.ResponseWriter, err error, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig)
}

// Entry Entry with path
type Entry struct {
	Type         string
	ETag         string
	Name         string
	LastModified time.Time
	Size         int64
	Key          string
	Path         string
}

// bucketListingData Bucket listing data for templating
type bucketListingData struct {
	Entries    []*Entry
	BucketName string
	Name       string
	Path       string
}

// Proxy proxy requests
func (rctx *requestContext) Proxy() {
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
			rctx.logger.Errorln(err)
			rctx.handleInternalServerError(rctx.httpRW, err, rctx.requestPath, rctx.logger, rctx.tplConfig)
			// Stop
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
				err = getFile(rctx, indexDocumentEntry.(*Entry).Key)
				// Check if error is a not found error
				if err == s3client.ErrNotFound {
					// Not found
					rctx.handleNotFound(rctx.httpRW, rctx.requestPath, rctx.logger, rctx.tplConfig)
					return
				}
				// Log error
				rctx.logger.Errorln(err)
				// Response with error
				rctx.handleInternalServerError(rctx.httpRW, err, rctx.requestPath, rctx.logger, rctx.tplConfig)
				// Stop
				return
			}
		}

		// Load template
		tplFileName := filepath.Base(rctx.tplConfig.FolderList)
		// Create template
		tmpl, err := template.New(tplFileName).Funcs(sprig.HtmlFuncMap()).Funcs(s3ProxyFuncMap()).ParseFiles(rctx.tplConfig.FolderList)
		if err != nil {
			rctx.logger.Errorln(err)
			rctx.handleInternalServerError(rctx.httpRW, err, rctx.requestPath, rctx.logger, rctx.tplConfig)
			// Stop
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
		// Execute template
		err = tmpl.Execute(buf, data)
		if err != nil {
			rctx.logger.Errorln(err)
			rctx.handleInternalServerError(rctx.httpRW, err, rctx.requestPath, rctx.logger, rctx.tplConfig)
			// Stop
			return
		}
		// Set status code
		rctx.httpRW.WriteHeader(200)
		// Set the header and write the buffer to the http.ResponseWriter
		rctx.httpRW.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Write buffer content to output
		_, err = buf.WriteTo(rctx.httpRW)
		if err != nil {
			rctx.logger.Errorln(err)
			rctx.handleInternalServerError(rctx.httpRW, err, rctx.requestPath, rctx.logger, rctx.tplConfig)
			// Stop
			return
		}
		// Stop
		return
	}

	// Get object case
	err := getFile(rctx, key)
	if err != nil {
		// Check if error is a not found error
		if err == s3client.ErrNotFound {
			// Not found
			rctx.handleNotFound(rctx.httpRW, rctx.requestPath, rctx.logger, rctx.tplConfig)
			// Stop
			return
		}
		// Log error
		rctx.logger.Errorln(err)
		// Manage error response
		rctx.handleInternalServerError(rctx.httpRW, err, rctx.requestPath, rctx.logger, rctx.tplConfig)
		// Stop
		return
	}
}

func transformS3Entries(s3Entries []*s3client.Entry, rctx *requestContext, bucketRootPrefixKey string) []*Entry {
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

func getFile(brctx *requestContext, key string) error {
	// Get object from s3
	objOutput, err := brctx.s3Context.GetObject(key)
	if err != nil {
		return err
	}
	// Set headers from object
	setHeadersFromObjectOutput(brctx.httpRW, objOutput)
	// Copy data stream to output stream
	_, err = io.Copy(brctx.httpRW, *objOutput.Body)
	// Return potential error
	return err
}
