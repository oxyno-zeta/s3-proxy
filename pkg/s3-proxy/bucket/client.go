package bucket

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
)

// ErrRemovalFolder will be raised when end user is trying to delete a folder and not a file.
var ErrRemovalFolder = errors.New("can't remove folder")

// Client represents a client in order to GET, PUT or DELETE file on a bucket with a html output.
type Client interface {
	// Get allow to GET what's inside a request path
	Get(input *GetInput)
	// Put will put a file following input
	Put(inp *PutInput)
	// Delete will delete file on request path
	Delete(requestPath string)
	// Handle not found errors with bucket configuration
	HandleNotFound(requestPath string)
	// Handle forbidden errors with bucket configuration
	HandleForbidden(requestPath string)
	// Handle bad request errors with bucket configuration
	HandleBadRequest(err error, requestPath string)
	// Handle internal server error errors with bucket configuration
	HandleInternalServerError(err error, requestPath string)
	// Handle unauthorized errors with bucket configuration
	HandleUnauthorized(requestPath string)
}

// GetInput represents Get input.
type GetInput struct {
	RequestPath       string
	IfModifiedSince   *time.Time
	IfMatch           string
	IfNoneMatch       string
	IfUnmodifiedSince *time.Time
}

// PutInput represents Put input.
type PutInput struct {
	RequestPath string
	Filename    string
	Body        io.ReadSeeker
	ContentType string
	ContentSize int64
}

// ErrorHandlers error handlers.
type ErrorHandlers struct {
	HandleNotFoundWithTemplate            func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string)            //nolint: lll // It is long
	HandleForbiddenWithTemplate           func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string)            //nolint: lll // It is long
	HandleUnauthorizedWithTemplate        func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string)            //nolint: lll // It is long
	HandleBadRequestWithTemplate          func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string, err error) //nolint: lll // It is long
	HandleInternalServerErrorWithTemplate func(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string, err error) //nolint: lll // It is long
}

// NewClient will generate a new client to do GET,PUT or DELETE actions.
func NewClient(
	tgt *config.TargetConfig, tplConfig *config.TemplateConfig, logger log.Logger,
	mountPath string, httpRW http.ResponseWriter, httpReq *http.Request,
	metricsCtx metrics.Client,
	errorHandlers *ErrorHandlers,
	parentTrace tracing.Trace,
) (Client, error) {
	s3ctx, err := s3client.NewS3Context(tgt, logger, metricsCtx, parentTrace)
	if err != nil {
		return nil, err
	}

	return &requestContext{
		s3Context:      s3ctx,
		logger:         logger,
		targetCfg:      tgt,
		mountPath:      mountPath,
		httpRW:         httpRW,
		httpReq:        httpReq,
		tplConfig:      tplConfig,
		errorsHandlers: errorHandlers,
	}, nil
}
