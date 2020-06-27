package tracing

import (
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
)

// Service interface
type Service interface {
	// Reload service (useful for configuration change)
	Reload() error
	// Get global tracer object
	GetTracer() opentracing.Tracer
}

// Trace object interface
type Trace interface {
	// Set tag on trace
	SetTag(key string, value interface{})
	// Get child trace with an operation name
	GetChildTrace(operationName string) Trace
	// Will finish the trace
	Finish()
	// Get trace id as a string (useful for logs)
	GetTraceID() string
}

func New(cfgManager config.Manager, logger log.Logger) (Service, error) {
	return newService(cfgManager, logger)
}
