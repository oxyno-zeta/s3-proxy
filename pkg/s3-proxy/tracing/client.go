package tracing

import (
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
)

type Service interface {
	Reload() error
	GetTracer() opentracing.Tracer
}

type Trace interface {
	SetTag(key string, value interface{})
	GetChildTrace(operationName string) Trace
	Finish()
	GetTraceID() string
}

func New(cfgManager config.Manager, logger log.Logger) (Service, error) {
	return newService(cfgManager, logger)
}
