package bucket

import (
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3client"
	"github.com/sirupsen/logrus"
)

type Client interface {
	Proxy()
}

// NewClient New Client
// nolint:whitespace
func NewClient(
	tgt *config.Target, tplConfig *config.TemplateConfig, logger logrus.FieldLogger,
	mountPath string, requestPath string, httpRW http.ResponseWriter,
	handleNotFound func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig),
	handleInternalServerError func(rw http.ResponseWriter, err error, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig),
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
		requestPath:               requestPath,
		httpRW:                    httpRW,
		tplConfig:                 tplConfig,
		handleNotFound:            handleNotFound,
		handleInternalServerError: handleInternalServerError,
	}, nil
}
