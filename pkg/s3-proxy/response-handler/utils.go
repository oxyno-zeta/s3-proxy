package responsehandler

import (
	"io"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"emperror.dev/errors"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler/models"
	utils "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/utils/generalutils"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/utils/templateutils"
)

func (*handler) manageStatus(
	helpersContent string,
	tplConfigItem *config.TargetTemplateConfigItem,
	defaultTpl string,
	data any,
) (int, error) {
	// Create main status content
	statusContent := helpersContent

	// Check if per target template is declared
	if tplConfigItem != nil && tplConfigItem.Status != "" {
		// Concat
		statusContent = statusContent + "\n" + tplConfigItem.Status
	} else {
		// Concat
		statusContent = statusContent + "\n" + defaultTpl
	}

	// Execute status main template
	buf, err := templateutils.ExecuteTemplate(statusContent, data)
	// Check error
	if err != nil {
		return 0, err
	}

	// Get string from buffer
	str := buf.String()
	// Remove all new lines
	str = utils.NewLineMatcherRegex.ReplaceAllString(str, "")

	// Try to parse int from string
	return strconv.Atoi(str)
}

func (*handler) manageHeaders(helpersContent string, headersTpl map[string]string, hData any) (map[string]string, error) {
	// Store result
	res := map[string]string{}

	// Loop over all headers asked
	for k, htpl := range headersTpl {
		// Concat helpers to header template
		tpl := helpersContent + "\n" + htpl
		// Execute template
		buf, err := templateutils.ExecuteTemplate(tpl, hData)
		// Check error
		if err != nil {
			return nil, err
		}
		// Get string from buffer
		str := buf.String()
		// Remove all new lines
		str = utils.NewLineMatcherRegex.ReplaceAllString(str, "")
		// Save data only if the header isn't empty
		if str != "" {
			// Save
			res[k] = str
		}
	}

	// Return
	return res, nil
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

	// Check if we aren't in head answer
	if !h.headAnswerMode {
		// Write to response
		_, err := bodyBuf.WriteTo(h.res)
		// Check if error exists
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func setHeadersFromObjectOutput(w http.ResponseWriter, obj *models.StreamInput) {
	setStrHeader(w, "Cache-Control", obj.CacheControl)
	setStrHeader(w, "Expires", obj.Expires)
	setStrHeader(w, "Content-Digest", obj.ContentDigest)
	setStrHeader(w, "Content-Disposition", obj.ContentDisposition)
	setStrHeader(w, "Content-Encoding", obj.ContentEncoding)
	setStrHeader(w, "Content-Language", obj.ContentLanguage)
	setIntHeader(w, "Content-Length", obj.ContentLength)
	setStrHeader(w, "Content-Range", obj.ContentRange)
	setStrHeader(w, "Content-Type", obj.ContentType)
	setStrHeader(w, "ETag", obj.ETag)
	setStrHeader(w, "Accept-Ranges", "bytes")
	setTimeHeader(w, "Last-Modified", obj.LastModified)

	httpStatus := determineHTTPStatus(obj)
	w.WriteHeader(httpStatus)
}

func determineHTTPStatus(obj *models.StreamInput) int {
	// Content-Range is only set when S3 itself answered 206. Mirror that status
	// even when the range covers the whole object: RFC 9110 forbids Content-Range
	// on a 200 response, and clients like Safari treat 200 as "ranges unsupported".
	if len(obj.ContentRange) > 0 {
		// Return partial content
		return http.StatusPartialContent
	}

	// Return ok
	return http.StatusOK
}

func setStrHeader(w http.ResponseWriter, key, value string) {
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
