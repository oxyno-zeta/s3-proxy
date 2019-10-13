package server

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/oxyno-zeta/s3-proxy/pkg/bucket"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/sirupsen/logrus"
)

// GenerateRouter Generate router
func GenerateRouter(logger *logrus.Logger, cfg *config.Config) (http.Handler, error) {
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.DefaultCompress)
	r.Use(middleware.NoCache)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(NewStructuredLogger(logger))
	r.Use(middleware.Recoverer)
	// Check if auth if enabled and oidc enabled
	if cfg.Auth != nil && cfg.Auth.OIDC != nil {
		err := oidcEndpoints(cfg.Auth.OIDC, cfg.Templates, r)
		if err != nil {
			return nil, err
		}
	}

	r.NotFound(func(rw http.ResponseWriter, req *http.Request) {
		logEntry := GetLogEntry(req)
		path := req.URL.RequestURI()
		handleNotFound(rw, path, &logEntry, cfg.Templates)
	})

	// Load main route only if main bucket path support option isn't enabled
	if !cfg.MainBucketPathSupport {
		r.Route("/", func(r chi.Router) {
			r = putAuthMiddlewares(cfg, r)
			r.Get("/", func(rw http.ResponseWriter, req *http.Request) {
				logEntry := GetLogEntry(req)
				generateTargetList(rw, &logEntry, cfg)
			})
		})
	}

	// Load all targets routes
	for i := 0; i < len(cfg.Targets); i++ {
		tgt := cfg.Targets[i]
		mountPath := "/" + tgt.Name
		requestMountPath := mountPath
		if cfg.MainBucketPathSupport {
			mountPath = ""
			requestMountPath = "/"
		}
		r.Route(requestMountPath, func(r chi.Router) {
			r = putAuthMiddlewares(cfg, r)
			r.Get("/*", func(rw http.ResponseWriter, req *http.Request) {
				requestPath := chi.URLParam(req, "*")
				logEntry := GetLogEntry(req)
				brctx, err := bucket.NewRequestContext(tgt, cfg.Templates, &logEntry, mountPath, requestPath, &rw, handleNotFound, handleInternalServerError)

				if err != nil {
					logger.Errorln(err)
					handleInternalServerError(rw, err, requestPath, &logEntry, cfg.Templates)
				} else {
					brctx.Proxy()
				}
			})
		})
	}

	return r, nil
}

func generateTargetList(rw http.ResponseWriter, logger *logrus.FieldLogger, cfg *config.Config) {
	err := templateExecution(cfg.Templates.TargetList, logger, rw, struct{ Targets []*config.Target }{Targets: cfg.Targets}, 200)
	if err != nil {
		(*logger).Errorln(err)
		handleInternalServerError(rw, err, "/", logger, cfg.Templates)
		return
	}
}

func putAuthMiddlewares(cfg *config.Config, r chi.Router) chi.Router {
	// Check if oidc is enabled
	if cfg.Auth != nil && cfg.Auth.OIDC != nil {
		return r.With(oidcAuthorizationMiddleware(cfg.Auth.OIDC, cfg.Templates))
	}
	// Check if basic auth is enabled
	if cfg.Auth != nil && cfg.Auth.Basic != nil {
		return r.With(basicAuthMiddleware(cfg.Auth.Basic, cfg.Templates))
	}
	return r
}
