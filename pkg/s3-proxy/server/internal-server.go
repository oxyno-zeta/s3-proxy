package server

import (
	"net/http"
	"strconv"

	"github.com/dimiro1/health"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/middlewares"
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
	// Store server
	svr.server = server

	svr.logger.Infof("Internal server listening on %s", addr)

	err := server.ListenAndServe()

	return err
}

func (svr *InternalServer) generateInternalRouter() http.Handler {
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.Compress(
		5,
		"text/html",
		"text/css",
		"text/plain",
		"text/javascript",
		"application/javascript",
		"application/x-javascript",
		"application/json",
		"application/atom+xml",
		"application/rss+xml",
		"image/svg+xml",
	))
	r.Use(middleware.NoCache)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middlewares.NewStructuredLogger(svr.logger))
	r.Use(svr.metricsCl.Instrument())
	r.Use(middleware.Recoverer)

	healthHandler := health.NewHandler()
	// Listen path
	r.Handle("/metrics", svr.metricsCl.GetExposeHandler())
	r.Handle("/health", healthHandler)

	return r
}
