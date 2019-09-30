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
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(NewStructuredLogger(logger))
	r.Use(middleware.Recoverer)

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
				brctx, err := bucket.NewRequestContext(tgt, cfg.Templates, &logEntry, mountPath, requestPath, &rw)

				if err != nil {
					// ! TODO Need to manage errors
					logger.Errorln(err)
				} else {
					brctx.Proxy()
				}
			})
		})
	}

	return r
}
