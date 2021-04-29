package responsehandler

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/utils"
)

func (h *handler) manageHeaders(helpersContent string, headersTpl map[string]string) (map[string]string, error) {
	// Create regex to remove all new lines
	reg := regexp.MustCompile(`\r?\n`)
	// Create header data
	hData := &headerData{
		Request: h.req,
		User:    models.GetAuthenticatedUserFromContext(h.req.Context()),
	}

	// Store result
	res := map[string]string{}

	// Loop over all headers asked
	for k, htpl := range headersTpl {
		// Concat helpers to header template
		tpl := helpersContent + "\n" + htpl
		// Execute template
		buf, err := utils.ExecuteTemplate(tpl, hData)
		// Check error
		if err != nil {
			return nil, err
		}
		// Get string from buffer
		str := buf.String()
		// Remove all new lines
		str = reg.ReplaceAllString(str, "")
		// Save data only if the header isn't empty
		if str != "" {
			// Save
			res[k] = str
		}
	}

	// Return
	return res, nil
}

func (h *handler) loadAllHelpersContent(
	loadS3FileContent func(ctx context.Context, path string) (string, error),
	items []*config.TargetHelperConfigItem,
	pathList []string,
) (string, error) {
	// Initialize template content
	tplContent := ""

	// Check if there is a list of config items
	if len(items) != 0 {
		// Loop over items
		for _, item := range items {
			// Load template content
			tpl, err := h.loadHelperContent(
				loadS3FileContent,
				item,
			)
			// Check error
			if err != nil {
				return "", err
			}
			// Concat
			tplContent = tplContent + "\n" + tpl
		}
	} else {
		// Load from local files
		// Loop over local path
		for _, item := range pathList {
			// Load template content
			tpl, err := loadLocalFileContent(item)
			// Check error
			if err != nil {
				return "", err
			}
			// Concat
			tplContent = tplContent + "\n" + tpl
		}
	}

	// Return
	return tplContent, nil
}

func (h *handler) loadHelperContent(
	loadS3FileContent func(ctx context.Context, path string) (string, error),
	item *config.TargetHelperConfigItem,
) (string, error) {
	// Check if it is in bucket and if load from S3 function exists
	if item.InBucket && loadS3FileContent != nil {
		// Try to get file from bucket
		return loadS3FileContent(h.req.Context(), item.Path)
	}

	// Not in bucket, need to load from FS
	return loadLocalFileContent(item.Path)
}

func (h *handler) loadTemplateContent(
	loadS3FileContent func(ctx context.Context, path string) (string, error),
	item *config.TargetTemplateConfigItem,
) (string, error) {
	// Check if it is in bucket and if load from S3 function exists
	if item.InBucket && loadS3FileContent != nil {
		// Try to get file from bucket
		return loadS3FileContent(h.req.Context(), item.Path)
	}

	// Not in bucket, need to load from FS
	return loadLocalFileContent(item.Path)
}

// send will send the response.
func (h *handler) send(bodyBuf io.WriterTo, headers map[string]string, status int) error {
	// Loop over headers
	for k, v := range headers {
		// Set header
		h.res.Header().Set(k, v)
	}

	// Set status code
	h.res.WriteHeader(status)

	// Write to response
	_, err := bodyBuf.WriteTo(h.res)
	// Check if error exists
	if err != nil {
		return err
	}

	return nil
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
