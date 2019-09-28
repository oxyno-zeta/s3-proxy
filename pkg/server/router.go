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

	for i := 0; i < len(cfg.Buckets); i++ {
		bcfg := cfg.Buckets[i]
		mountPath := "/" + bcfg.Name
		r.Route(mountPath, func(r chi.Router) {
			r.Get("/*", func(rw http.ResponseWriter, req *http.Request) {
				requestPath := chi.URLParam(req, "*")
				logEntry := GetLogEntry(req)
				brctx, err := bucket.NewBucketRequestContext(bcfg, &logEntry, mountPath, requestPath, &rw)

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
