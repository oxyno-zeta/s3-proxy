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
func GenerateRouter(logger *logrus.Logger, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.DefaultCompress)
	r.Use(middleware.NoCache)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(NewStructuredLogger(logger))
	r.Use(middleware.Recoverer)

	if cfg.Auth != nil && cfg.Auth.Basic != nil {
		r.Use(basicAuthMiddleware(cfg.Auth.Basic, cfg.Templates))
	}

	r.NotFound(func(rw http.ResponseWriter, req *http.Request) {
		logEntry := GetLogEntry(req)
		path := req.URL.RequestURI()
		handleNotFound(rw, path, &logEntry, cfg.Templates)
	})

	// Load main route only if main bucket path support option isn't enabled
	if !cfg.MainBucketPathSupport {
		r.Route("/", func(r chi.Router) {
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

	return r
}

func generateTargetList(rw http.ResponseWriter, logger *logrus.FieldLogger, cfg *config.Config) {
	err := templateExecution(cfg.Templates.TargetList, logger, rw, struct{ Targets []*config.Target }{Targets: cfg.Targets}, 200)
	if err != nil {
		(*logger).Errorln(err)
		handleInternalServerError(rw, err, "/", logger, cfg.Templates)
		return
	}
}
