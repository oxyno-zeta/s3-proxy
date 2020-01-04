package server

import (
	"net/http"

	"github.com/dimiro1/health"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/server/middlewares"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// GenerateInternalRouter Generate internal router
func GenerateInternalRouter(logger logrus.FieldLogger, cfg *config.Config, metricsCtx metrics.Instance) http.Handler {
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.DefaultCompress)
	r.Use(middleware.NoCache)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middlewares.NewStructuredLogger(logger))
	r.Use(middleware.Recoverer)

	healthHandler := health.NewHandler()
	// Listen path
	r.Handle("/metrics", promhttp.Handler())
	r.Handle("/health", healthHandler)

	return r
}
