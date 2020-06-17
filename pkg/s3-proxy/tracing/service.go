package tracing

import (
	"io"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"

	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerprom "github.com/uber/jaeger-lib/metrics/prometheus"
)

type service struct {
	closer     io.Closer
	tracer     opentracing.Tracer
	cfgManager config.Manager
	logger     log.Logger
}

func (s *service) GetTracer() opentracing.Tracer {
	return s.tracer
}

func (s *service) Reload() error {
	// Save closer
	cl := s.closer

	// Setup
	err := s.setup()
	if err != nil {
		return err
	}

	// Close old one
	err = cl.Close()
	if err != nil {
		return err
	}

	return nil
}

func (s *service) setup() error {
	cfg := s.cfgManager.GetConfig()
	// Initialize configuration
	jcfg := jaegercfg.Configuration{
		ServiceName: "s3-proxy",
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
	}

	// Check if configuration can be set
	if !cfg.Tracing.Enabled {
		jcfg.Disabled = true
	} else {
		// Add reporter configuration
		jcfg.Reporter = &jaegercfg.ReporterConfig{
			LogSpans:  cfg.Tracing.LogSpan,
			QueueSize: cfg.Tracing.QueueSize,
		}

		// Check if flush interval is customized
		if cfg.Tracing.FlushInterval != "" {
			// Try to parse duration for flush interval
			dur, err := time.ParseDuration(cfg.Tracing.FlushInterval)
			if err != nil {
				return err
			}

			jcfg.Reporter.BufferFlushInterval = dur
		}

		// Check if UDP is customized
		if cfg.Tracing.UDPHost != "" {
			jcfg.Reporter.LocalAgentHostPort = cfg.Tracing.UDPHost
		}
	}

	// Create prometheus metrics factory
	factory := jaegerprom.New()

	// Initialize tracer with a logger and a metrics factory
	tracer, closer, err := jcfg.NewTracer(
		jaegercfg.Logger(s.logger.GetTracingLogger()),
		jaegercfg.Metrics(factory),
	)
	// Check error
	if err != nil {
		return err
	}
	// Set the singleton opentracing.Tracer with the Jaeger tracer.
	opentracing.SetGlobalTracer(tracer)

	s.closer = closer
	s.tracer = tracer

	return nil
}

func newService(cfgManager config.Manager, logger log.Logger) (*service, error) {
	svc := &service{
		cfgManager: cfgManager,
		logger:     logger,
	}

	// Run setup
	err := svc.setup()
	if err != nil {
		return nil, err
	}

	return svc, nil
}
