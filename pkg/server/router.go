package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3client"
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
		bucket := cfg.Buckets[i]
		r.Route("/"+bucket.Name, func(r chi.Router) {
			r.Get("/*", func(rw http.ResponseWriter, req *http.Request) {
				logEntry := GetLogEntry(req)
				s3ctx, err := s3client.NewS3Context(bucket, &logEntry)
				path := chi.URLParam(req, "*")
				if err != nil {
					logger.Errorln((err))
				} else {
					res, _ := s3ctx.ListFilesAndDirectories(path)
					for _, item := range res {
						fmt.Println(item)
					}
				}

				rw.Write([]byte(path))
			})
		})
	}

	return r
}
