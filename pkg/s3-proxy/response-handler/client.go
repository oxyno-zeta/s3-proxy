package responsehandler

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

var responseHandlerCtxKey = &contextKey{name: "ResponseHandlerCtxKey"}

// Entry Entry with path for internal use (template).
type Entry struct {
	Type         string
	ETag         string
	Name         string
	LastModified time.Time
	Size         int64
	Key          string
	Path         string
}

// StreamInput represents a stream input file.
type StreamInput struct {
	Body               io.ReadCloser
	CacheControl       string
	Expires            string
	ContentDisposition string
	ContentEncoding    string
	ContentLanguage    string
	ContentLength      int64
	ContentRange       string
	ContentType        string
	ETag               string
	LastModified       time.Time
}

// ResponseHandler will handle responses.
type ResponseHandler interface {
	// TargetList will answer for the target list response.
	TargetList()
	// NoContent will answer with the NoContent status code.
	NoContent()
	// NotModified will answer with a Not Modified status code.
	NotModified()
	// PreconditionFailed will answer with a Precondition Failed status code.
	PreconditionFailed()
	// RedirectWithTrailingSlash will redirect with a trailing slash.
	RedirectWithTrailingSlash()
	// StreamFile will stream file in output.
	// Error will be managed outside of this function because of the workflow in the caller function.
	StreamFile(output *StreamInput) error
	// FoldersFilesList will answer with the folder list output coming from template.
	FoldersFilesList(
		loadFileContent func(ctx context.Context, path string) (string, error),
		entries []*Entry,
	)
	// NotFoundError will answer for not found error.
	NotFoundError(
		loadFileContent func(ctx context.Context, path string) (string, error),
	)
	// ForbiddenError will answer for forbidden error.
	ForbiddenError(
		loadFileContent func(ctx context.Context, path string) (string, error),
		err error,
	)
	// BadRequestError will answer for bad request error.
	BadRequestError(
		loadFileContent func(ctx context.Context, path string) (string, error),
		err error,
	)
	// UnauthorizedError will answer for unauthorized error.
	UnauthorizedError(
		loadFileContent func(ctx context.Context, path string) (string, error),
		err error,
	)
	// InternalServerError will answer for internal server error.
	InternalServerError(
		loadFileContent func(ctx context.Context, path string) (string, error),
		err error,
	)
	// UpdateRequestAndResponse will update request and response in object.
	// This will used to update request and response in order to have the latest context values.
	UpdateRequestAndResponse(req *http.Request, res http.ResponseWriter)
}

// NewHandler will return a new response handler object.
func NewHandler(req *http.Request, res http.ResponseWriter, cfgManager config.Manager, targetKey string) ResponseHandler {
	return &handler{
		req:        req,
		res:        res,
		cfgManager: cfgManager,
		targetKey:  targetKey,
	}
}

// GetResponseHandlerFromContext will return the response handler object from context.
func GetResponseHandlerFromContext(ctx context.Context) ResponseHandler {
	return ctx.Value(responseHandlerCtxKey).(ResponseHandler)
}
