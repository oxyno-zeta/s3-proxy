package server

import (
	"net/http"

	"github.com/dimiro1/health"
	"github.com/go-chi/chi"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// GenerateInternalRouter Generate internal router
func GenerateInternalRouter(logger *logrus.Logger, cfg *config.Config) http.Handler {
	r := chi.NewRouter()
	healthHandler := health.NewHandler()
	// Listen path
	r.Handle("/metrics", promhttp.Handler())
	r.Handle("/health", healthHandler)
	return r
}
