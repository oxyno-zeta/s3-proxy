package server

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/hostrouter"
	"github.com/oxyno-zeta/s3-proxy/pkg/bucket"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/server/middlewares"
	"github.com/oxyno-zeta/s3-proxy/pkg/server/utils"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

// GenerateRouter Generate router
func GenerateRouter(logger logrus.FieldLogger, cfg *config.Config, metricsCtx metrics.Instance) (http.Handler, error) {
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.DefaultCompress)
	r.Use(middleware.NoCache)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middlewares.NewStructuredLogger(logger))
	r.Use(metricsCtx.Instrument())
	r.Use(middleware.Recoverer)
	// Check if auth if enabled and oidc enabled
	if cfg.AuthProviders != nil && cfg.AuthProviders.OIDC != nil {
		for _, v := range cfg.AuthProviders.OIDC {
			err := middlewares.OIDCEndpoints(v, cfg.Templates, r)
			if err != nil {
				return nil, err
			}
		}
	}

	r.NotFound(func(rw http.ResponseWriter, req *http.Request) {
		logEntry := middlewares.GetLogEntry(req)
		path := req.URL.RequestURI()
		utils.HandleNotFound(rw, path, logEntry, cfg.Templates)
	})

	// Create host router
	hr := hostrouter.New()

	// Load main route only if main bucket path support option isn't enabled
	if cfg.ListTargets.Enabled {
		// Create new router
		rt := chi.NewRouter()
		// Make list of resources from resource
		resources := make([]*config.Resource, 0)
		if cfg.ListTargets.Resource != nil {
			resources = append(resources, cfg.ListTargets.Resource)
		}
		// Manage path for list targets feature
		// Loop over path list
		funk.ForEach(cfg.ListTargets.Mount.Path, func(path string) {
			rt.Route(path, func(rt2 chi.Router) {
				rt2 = rt2.With(middlewares.AuthMiddleware(cfg, resources))
				rt2.Get("/", func(rw http.ResponseWriter, req *http.Request) {
					logEntry := middlewares.GetLogEntry(req)
					generateTargetList(rw, logEntry, cfg)
				})
			})
		})
		// Create domain
		domain := cfg.ListTargets.Mount.Host
		if domain == "" {
			domain = "*"
		}
		// Mount domain from configuration
		hr.Map(domain, rt)
	}

	// Load all targets routes
	funk.ForEach(cfg.Targets, func(tgt *config.Target) {
		// Manage domain
		domain := tgt.Mount.Host
		if domain == "" {
			domain = "*"
		}
		// Get router from hostrouter if exists
		rt := hr[domain]
		if rt == nil {
			// Create a new router
			rt = chi.NewRouter()
		}
		// Loop over path list
		funk.ForEach(tgt.Mount.Path, func(path string) {
			rt.Route(path, func(rt2 chi.Router) {
				rt2 = rt2.With(middlewares.AuthMiddleware(cfg, tgt.Resources))
				rt2.Get("/*", func(rw http.ResponseWriter, req *http.Request) {
					requestPath := chi.URLParam(req, "*")
					logEntry := middlewares.GetLogEntry(req)
					brctx, err := bucket.NewRequestContext(tgt, cfg.Templates, logEntry,
						path, requestPath, rw, utils.HandleNotFound,
						utils.HandleInternalServerError, metricsCtx)

					if err != nil {
						logger.Errorln(err)
						utils.HandleInternalServerError(rw, err, requestPath, logEntry, cfg.Templates)
					} else {
						brctx.Proxy()
					}
				})
			})
		})
		// Mount domain from target
		hr.Map(domain, rt)
	})

	// Mount host router
	r.Mount("/", hr)

	return r, nil
}
