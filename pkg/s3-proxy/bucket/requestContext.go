package bucket

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
)

// ErrRemovalFolder will be raised when end user is trying to delete a folder and not a file
var ErrRemovalFolder = errors.New("can't remove folder")

// requestContext Bucket request context
type requestContext struct {
	s3Context      s3client.Client
	logger         log.Logger
	targetCfg      *config.TargetConfig
	tplConfig      *config.TemplateConfig
	mountPath      string
	httpRW         http.ResponseWriter
	errorsHandlers *ErrorHandlers
}

// Entry Entry with path for internal use (template)
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
	bucketRootPrefixKey := rctx.targetCfg.Bucket.GetRootPrefix()
	// Key must begin by bucket prefix
	key := bucketRootPrefixKey
	// Trim first / if exists
	key += strings.TrimPrefix(requestPath, "/")

	return key
}

func (rctx *requestContext) HandleInternalServerError(err error, requestPath string) {
	// Initialize content
	content := ""
	// Check if file is in bucket
	if rctx.targetCfg != nil &&
		rctx.targetCfg.Templates != nil &&
		rctx.targetCfg.Templates.InternalServerError != nil {
		// Put error err2 to avoid erase of err
		var err2 error
		content, err2 = rctx.loadTemplateContent(rctx.targetCfg.Templates.InternalServerError)
		// Check if error exists
		if err2 != nil {
			// This is a particular case. In this case, remove old error and manage new one
			err = err2
		}
	}

	rpath := path.Join(rctx.mountPath, requestPath)
	rctx.errorsHandlers.HandleInternalServerErrorWithTemplate(rctx.logger, rctx.httpRW, rctx.tplConfig, content, rpath, err)
}

func (rctx *requestContext) HandleNotFound(requestPath string) {
	// Initialize content
	content := ""
	// Check if file is in bucket
	if rctx.targetCfg != nil &&
		rctx.targetCfg.Templates != nil &&
		rctx.targetCfg.Templates.NotFound != nil {
		// Declare error
		var err error
		// Try to get file from bucket
		content, err = rctx.loadTemplateContent(rctx.targetCfg.Templates.NotFound)
		if err != nil {
			rctx.HandleInternalServerError(err, requestPath)
			return
		}
	}

	rpath := path.Join(rctx.mountPath, requestPath)
	rctx.errorsHandlers.HandleNotFoundWithTemplate(rctx.logger, rctx.httpRW, rctx.tplConfig, content, rpath)
}

func (rctx *requestContext) HandleForbidden(requestPath string) {
	// Initialize content
	content := ""
	// Check if file is in bucket
	if rctx.targetCfg != nil &&
		rctx.targetCfg.Templates != nil &&
		rctx.targetCfg.Templates.Forbidden != nil {
		// Declare error
		var err error
		// Try to get file from bucket
		content, err = rctx.loadTemplateContent(rctx.targetCfg.Templates.Forbidden)
		if err != nil {
			rctx.HandleInternalServerError(err, requestPath)
			return
		}
	}

	rpath := path.Join(rctx.mountPath, requestPath)
	rctx.errorsHandlers.HandleForbiddenWithTemplate(rctx.logger, rctx.httpRW, rctx.tplConfig, content, rpath)
}

func (rctx *requestContext) HandleBadRequest(err error, requestPath string) {
	// Initialize content
	content := ""
	// Check if file is in bucket
	if rctx.targetCfg != nil &&
		rctx.targetCfg.Templates != nil &&
		rctx.targetCfg.Templates.BadRequest != nil {
		// Declare error
		var err2 error
		// Try to get file from bucket
		content, err2 = rctx.loadTemplateContent(rctx.targetCfg.Templates.BadRequest)
		if err2 != nil {
			rctx.HandleInternalServerError(err2, requestPath)
			return
		}
	}

	rpath := path.Join(rctx.mountPath, requestPath)
	rctx.errorsHandlers.HandleBadRequestWithTemplate(rctx.logger, rctx.httpRW, rctx.tplConfig, content, rpath, err)
}

func (rctx *requestContext) HandleUnauthorized(requestPath string) {
	// Initialize content
	content := ""
	// Check if file is in bucket
	if rctx.targetCfg != nil &&
		rctx.targetCfg.Templates != nil &&
		rctx.targetCfg.Templates.Unauthorized != nil {
		// Declare error
		var err error
		// Try to get file from bucket
		content, err = rctx.loadTemplateContent(rctx.targetCfg.Templates.Unauthorized)
		if err != nil {
			rctx.HandleInternalServerError(err, requestPath)
			return
		}
	}

	rpath := path.Join(rctx.mountPath, requestPath)
	rctx.errorsHandlers.HandleUnauthorizedWithTemplate(rctx.logger, rctx.httpRW, rctx.tplConfig, content, rpath)
}

// Get proxy GET requests
func (rctx *requestContext) Get(requestPath string) {
	key := rctx.generateStartKey(requestPath)
	// Check that the path ends with a / for a directory listing or the main path special case (empty path)
	if strings.HasSuffix(requestPath, "/") || requestPath == "" {
		rctx.manageGetFolder(key, requestPath)
		// Stop
		return
	}

	// Get object case
	err := rctx.streamFileForResponse(key)
	if err != nil {
		// Check if error is a not found error
		if err == s3client.ErrNotFound {
			// Not found
			rctx.HandleNotFound(requestPath)
			// Stop
			return
		}
		// Log error
		rctx.logger.Error(err)
		// Manage error response
		rctx.HandleInternalServerError(err, requestPath)
		// Stop
		return
	}
}

func (rctx *requestContext) manageGetFolder(key, requestPath string) {
	// Check if index document is activated
	if rctx.targetCfg.IndexDocument != "" {
		// Create index key path
		indexKey := path.Join(key, rctx.targetCfg.IndexDocument)
		// Head index file in bucket
		headOutput, err := rctx.s3Context.HeadObject(indexKey)
		// Check if error exists and not a not found error
		if err != nil && err != s3client.ErrNotFound {
			// Log error
			rctx.logger.Error(err)
			// Manage error response
			rctx.HandleInternalServerError(err, requestPath)
			// Stop
			return
		}
		// Check that we found the file
		if headOutput != nil {
			// Get data
			err = rctx.streamFileForResponse(headOutput.Key)
			// Check if error exists
			if err != nil {
				// Check if error is a not found error
				if err == s3client.ErrNotFound {
					// Not found
					rctx.HandleNotFound(requestPath)
					return
				}
				// Log error
				rctx.logger.Error(err)
				// Response with error
				rctx.HandleInternalServerError(err, requestPath)
				// Stop
				return
			}
			// Stop here because no error are present
			return
		}
	}

	// Directory listing case
	s3Entries, err := rctx.s3Context.ListFilesAndDirectories(key)
	if err != nil {
		rctx.logger.Error(err)
		rctx.HandleInternalServerError(err, requestPath)
		// Stop
		return
	}

	// Transform entries in entry with path objects
	bucketRootPrefixKey := rctx.targetCfg.Bucket.GetRootPrefix()
	entries := transformS3Entries(s3Entries, rctx, bucketRootPrefixKey)

	var tmpl *template.Template
	// Check if per target template is declared
	if rctx.targetCfg != nil && rctx.targetCfg.Templates != nil &&
		rctx.targetCfg.Templates.FolderList != nil {
		// Load template file name
		tplFileName := filepath.Base(rctx.targetCfg.Templates.FolderList.Path)
		// Get template content
		var content string
		content, err = rctx.loadTemplateContent(rctx.targetCfg.Templates.FolderList)
		// Check if errors exists in load file content
		if err == nil {
			// Create template executor
			tmpl, err = template.New(tplFileName).Funcs(sprig.HtmlFuncMap()).Funcs(s3ProxyFuncMap()).Parse(content)
		}
	} else {
		// Load template file name
		tplFileName := filepath.Base(rctx.tplConfig.FolderList)
		// Create template executor
		tmpl, err = template.New(tplFileName).Funcs(sprig.HtmlFuncMap()).Funcs(s3ProxyFuncMap()).ParseFiles(rctx.tplConfig.FolderList)
	}

	// Check error
	if err != nil {
		rctx.logger.Error(err)
		rctx.HandleInternalServerError(err, requestPath)
		// Stop
		return
	}
	// Create bucket list data for templating
	data := &bucketListingData{
		Entries:    entries,
		BucketName: rctx.targetCfg.Bucket.Name,
		Name:       rctx.targetCfg.Name,
		Path:       rctx.mountPath + requestPath,
	}
	// Generate template in buffer
	buf := &bytes.Buffer{}
	// Execute template
	err = tmpl.Execute(buf, data)
	if err != nil {
		rctx.logger.Error(err)
		rctx.HandleInternalServerError(err, requestPath)
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
		rctx.logger.Error(err)
		rctx.HandleInternalServerError(err, requestPath)
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
		ContentSize: inp.ContentSize,
	}

	// Check if post actions configuration exists
	if rctx.targetCfg.Actions.PUT != nil &&
		rctx.targetCfg.Actions.PUT.Config != nil {
		// Check if metadata is configured in target configuration
		if rctx.targetCfg.Actions.PUT.Config.Metadata != nil {
			input.Metadata = rctx.targetCfg.Actions.PUT.Config.Metadata
		}

		// Check if storage class is present in target configuration
		if rctx.targetCfg.Actions.PUT.Config.StorageClass != "" {
			input.StorageClass = rctx.targetCfg.Actions.PUT.Config.StorageClass
		}

		// Check if allow override is enabled
		if !rctx.targetCfg.Actions.PUT.Config.AllowOverride {
			// Need to check if file already exists
			headOutput, err := rctx.s3Context.HeadObject(key)
			// Check if error is not found if exists
			if err != nil && err != s3client.ErrNotFound {
				rctx.logger.Error(err)
				rctx.HandleInternalServerError(err, inp.RequestPath)
				// Stop
				return
			}
			// Check if file exists
			if headOutput != nil {
				rctx.logger.Errorf("File detected on path %s for PUT request and override isn't allowed", key)
				rctx.HandleForbidden(inp.RequestPath)
				// Stop
				return
			}
		}
	}
	// Put file
	err := rctx.s3Context.PutObject(input)
	if err != nil {
		rctx.logger.Error(err)
		rctx.HandleInternalServerError(err, inp.RequestPath)
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
		rctx.HandleInternalServerError(ErrRemovalFolder, requestPath)
		// Stop
		return
	}
	// Delete object in S3
	err := rctx.s3Context.DeleteObject(key)
	// Check if error exists
	if err != nil {
		rctx.logger.Error(err)
		rctx.HandleInternalServerError(err, requestPath)
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

func (rctx *requestContext) loadTemplateContent(item *config.TargetTemplateConfigItem) (string, error) {
	// Check if it is in bucket
	if item.InBucket {
		// Try to get file from bucket
		return rctx.getFileContent(item.Path)
	}

	// Not in bucket, need to load from FS
	by, err := ioutil.ReadFile(item.Path)
	// Check if error exists
	if err != nil {
		return "", err
	}

	return string(by), nil
}

func (rctx *requestContext) getFileContent(path string) (string, error) {
	// Get object from s3
	objOutput, err := rctx.s3Context.GetObject(path)
	if err != nil {
		return "", err
	}

	// Read all body
	bb, err := ioutil.ReadAll(*objOutput.Body)
	if err != nil {
		return "", err
	}

	// Transform it to string and return
	return string(bb), nil
}

func (rctx *requestContext) streamFileForResponse(key string) error {
	// Get object from s3
	objOutput, err := rctx.s3Context.GetObject(key)
	if err != nil {
		return err
	}
	// Set headers from object
	setHeadersFromObjectOutput(rctx.httpRW, objOutput)
	// Copy data stream to output stream
	_, err = io.Copy(rctx.httpRW, *objOutput.Body)
	// Return potential error
	return err
}
