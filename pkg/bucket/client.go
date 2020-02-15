package bucket

import (
	"io"
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3client"
	"github.com/sirupsen/logrus"
)

type Client interface {
	// Get allow to GET what's inside a request path
	Get(requestPath string)
	// Put will put a file following input
	Put(inp *PutInput)
	// Delete will delete file on request path
	Delete(requestPath string)
}

// PutInput represents Put input
type PutInput struct {
	RequestPath string
	Filename    string
	Body        io.Reader
	ContentType string
}

// NewClient will generate a new client to do GET,PUT or DELETE actions
// nolint:whitespace
func NewClient(
	tgt *config.TargetConfig, tplConfig *config.TemplateConfig, logger logrus.FieldLogger,
	mountPath string, httpRW http.ResponseWriter,
	handleNotFound func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig),
	handleInternalServerError func(rw http.ResponseWriter, err error, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig),
	handleForbidden func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig),
	metricsCtx metrics.Client,
) (Client, error) {
	s3ctx, err := s3client.NewS3Context(tgt, logger, metricsCtx)
	if err != nil {
		return nil, err
	}

	return &requestContext{
		s3Context:                 s3ctx,
		logger:                    logger,
		bucketInstance:            tgt,
		mountPath:                 mountPath,
		httpRW:                    httpRW,
		tplConfig:                 tplConfig,
		handleNotFound:            handleNotFound,
		handleForbidden:           handleForbidden,
		handleInternalServerError: handleInternalServerError,
	}, nil
}
