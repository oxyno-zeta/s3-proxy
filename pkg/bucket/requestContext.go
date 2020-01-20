package bucket

import (
	"bytes"
	"errors"
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

var ErrRemovalFolder = errors.New("can't remove folder")

// requestContext Bucket request context
type requestContext struct {
	s3Context                 s3client.Client
	logger                    logrus.FieldLogger
	bucketInstance            *config.TargetConfig
	tplConfig                 *config.TemplateConfig
	mountPath                 string
	httpRW                    http.ResponseWriter
	handleNotFound            func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig)
	handleInternalServerError func(rw http.ResponseWriter, err error, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig)
	handleForbidden           func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig)
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

// generateStartKey will generate start key used in all functions
func (rctx *requestContext) generateStartKey(requestPath string) string {
	bucketRootPrefixKey := rctx.bucketInstance.Bucket.GetRootPrefix()
	// Key must begin by bucket prefix
	key := bucketRootPrefixKey
	// Trim first / if exists
	key += strings.TrimPrefix(requestPath, "/")

	return key
}

// Get proxy GET requests
func (rctx *requestContext) Get(requestPath string) {
	key := rctx.generateStartKey(requestPath)
	// Check that the path ends with a / for a directory listing or the main path special case (empty path)
	if strings.HasSuffix(requestPath, "/") || requestPath == "" {
		// Directory listing case
		s3Entries, err := rctx.s3Context.ListFilesAndDirectories(key)
		if err != nil {
			rctx.logger.Errorln(err)
			rctx.handleInternalServerError(rctx.httpRW, err, requestPath, rctx.logger, rctx.tplConfig)
			// Stop
			return
		}

		// Transform entries in entry with path objects
		bucketRootPrefixKey := rctx.bucketInstance.Bucket.GetRootPrefix()
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
					rctx.handleNotFound(rctx.httpRW, requestPath, rctx.logger, rctx.tplConfig)
					return
				}
				// Log error
				rctx.logger.Errorln(err)
				// Response with error
				rctx.handleInternalServerError(rctx.httpRW, err, requestPath, rctx.logger, rctx.tplConfig)
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
			rctx.handleInternalServerError(rctx.httpRW, err, requestPath, rctx.logger, rctx.tplConfig)
			// Stop
			return
		}
		// Create bucket list data for templating
		data := &bucketListingData{
			Entries:    entries,
			BucketName: rctx.bucketInstance.Bucket.Name,
			Name:       rctx.bucketInstance.Name,
			Path:       rctx.mountPath + requestPath,
		}
		// Generate template in buffer
		buf := &bytes.Buffer{}
		// Execute template
		err = tmpl.Execute(buf, data)
		if err != nil {
			rctx.logger.Errorln(err)
			rctx.handleInternalServerError(rctx.httpRW, err, requestPath, rctx.logger, rctx.tplConfig)
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
			rctx.handleInternalServerError(rctx.httpRW, err, requestPath, rctx.logger, rctx.tplConfig)
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
			rctx.handleNotFound(rctx.httpRW, requestPath, rctx.logger, rctx.tplConfig)
			// Stop
			return
		}
		// Log error
		rctx.logger.Errorln(err)
		// Manage error response
		rctx.handleInternalServerError(rctx.httpRW, err, requestPath, rctx.logger, rctx.tplConfig)
		// Stop
		return
	}
}

// Put proxy PUT requests
func (rctx *requestContext) Put(inp *PutInput) {
	key := rctx.generateStartKey(inp.RequestPath)
	// Add / at the end if not present
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}
	// Add filename at the end of key
	key += inp.Filename
	// Create input
	input := &s3client.PutInput{
		Key:         key,
		Body:        inp.Body,
		ContentType: inp.ContentType,
	}

	// Check if post actions configuration exists
	if rctx.bucketInstance.Actions.PUT != nil &&
		rctx.bucketInstance.Actions.PUT.Config != nil {
		// Check if metadata is configured in target configuration
		if rctx.bucketInstance.Actions.PUT.Config.Metadata != nil {
			input.Metadata = rctx.bucketInstance.Actions.PUT.Config.Metadata
		}

		// Check if storage class is present in target configuration
		if rctx.bucketInstance.Actions.PUT.Config.StorageClass != "" {
			input.StorageClass = rctx.bucketInstance.Actions.PUT.Config.StorageClass
		}

		// Check if allow override is enabled
		if !rctx.bucketInstance.Actions.PUT.Config.AllowOverride {
			// Need to check if file already exists
			headOutput, err := rctx.s3Context.HeadObject(key)
			// Check if error is not found if exists
			if err != nil && err != s3client.ErrNotFound {
				rctx.logger.Error(err)
				rctx.handleInternalServerError(rctx.httpRW, err, inp.RequestPath, rctx.logger, rctx.tplConfig)
				// Stop
				return
			}
			// Check if file exists
			if headOutput != nil {
				rctx.logger.Errorf("File detected on path %s for PUT request and override isn't allowed", key)
				rctx.handleForbidden(rctx.httpRW, inp.RequestPath, rctx.logger, rctx.tplConfig)
				// Stop
				return
			}
		}
	}
	// Put file
	err := rctx.s3Context.PutObject(input)
	if err != nil {
		rctx.logger.Error(err)
		rctx.handleInternalServerError(rctx.httpRW, err, inp.RequestPath, rctx.logger, rctx.tplConfig)
		// Stop
		return
	}
	// Set status code
	rctx.httpRW.WriteHeader(http.StatusNoContent)
}

// Delete will delete object in S3
func (rctx *requestContext) Delete(requestPath string) {
	key := rctx.generateStartKey(requestPath)
	// Check that the path ends with a / for a directory or the main path special case (empty path)
	if strings.HasSuffix(requestPath, "/") || requestPath == "" {
		rctx.logger.Error(ErrRemovalFolder)
		rctx.handleInternalServerError(rctx.httpRW, ErrRemovalFolder, requestPath, rctx.logger, rctx.tplConfig)
		// Stop
		return
	}
	// Delete object in S3
	err := rctx.s3Context.DeleteObject(key)
	// Check if error exists
	if err != nil {
		rctx.logger.Error(err)
		rctx.handleInternalServerError(rctx.httpRW, err, requestPath, rctx.logger, rctx.tplConfig)
		// Stop
		return
	}
	// Set status code
	rctx.httpRW.WriteHeader(http.StatusNoContent)
}

func transformS3Entries(s3Entries []*s3client.ListElementOutput, rctx *requestContext, bucketRootPrefixKey string) []*Entry {
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
