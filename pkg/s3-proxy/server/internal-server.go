package server

import (
	"net/http"
	"strconv"

	"github.com/dimiro1/health"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/middlewares"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
	"github.com/pkg/errors"
)

type InternalServer struct {
	logger     log.Logger
	cfgManager config.Manager
	metricsCl  metrics.Client
	server     *http.Server
}

func NewInternalServer(logger log.Logger, cfgManager config.Manager, metricsCl metrics.Client) *InternalServer {
	return &InternalServer{
		logger:     logger,
		cfgManager: cfgManager,
		metricsCl:  metricsCl,
	}
}

func (svr *InternalServer) Listen() error {
	svr.logger.Infof("Internal server listening on %s", svr.server.Addr)
	// Listen
	err := svr.server.ListenAndServe()
	// Check error
	if err != nil {
		return errors.WithStack(err)
	}

	// Default
	return nil
}

func (svr *InternalServer) GenerateServer() {
	// Get configuration
	cfg := svr.cfgManager.GetConfig()
	// Generate internal router
	r := svr.generateInternalRouter()
	// Create server
	addr := cfg.InternalServer.ListenAddr + ":" + strconv.Itoa(cfg.InternalServer.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Get the TLS configuration (if necessary)
	tlsConfig, err := generateTLSConfig(cfg.InternalServer.SSL, svr.logger)
	if err != nil {
		svr.logger.Fatal(err)
	}

	server.TLSConfig = tlsConfig

	// Store server
	svr.server = server
}

func (svr *InternalServer) generateInternalRouter() http.Handler {
	r := chi.NewRouter()

	// Get configuration
	cfg := svr.cfgManager.GetConfig()

	// Check if we need to enabled the compress middleware
	if *cfg.InternalServer.Compress.Enabled {
		r.Use(middleware.Compress(
			cfg.InternalServer.Compress.Level,
			cfg.InternalServer.Compress.Types...,
		))
	}

	// Check if no cache is disabled or not
	if cfg.InternalServer.Cache == nil || cfg.InternalServer.Cache.NoCacheEnabled {
		// Apply no cache
		r.Use(middleware.NoCache)
	} else {
		// Apply S3 proxy cache management middleware
		r.Use(middlewares.CacheManagement(cfg.InternalServer.Cache))
	}

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(log.NewStructuredLogger(
		svr.logger,
		tracing.GetTraceIDFromRequest,
	))
	r.Use(log.HTTPAddLoggerToContextMiddleware())
	r.Use(svr.metricsCl.Instrument("internal"))
	r.Use(middleware.Recoverer)

	healthHandler := health.NewHandler()
	// Listen path
	r.Handle("/metrics", svr.metricsCl.GetExposeHandler())
	r.Handle("/health", healthHandler)

	return r
}
