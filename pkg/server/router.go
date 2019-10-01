package server

import (
	"bytes"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/Masterminds/sprig"
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

func generateTargetList(rw http.ResponseWriter, logger *logrus.FieldLogger, cfg *config.Config) {
	// Load template
	tplFileName := filepath.Base(cfg.Templates.TargetList)
	tmpl, err := template.New(tplFileName).Funcs(sprig.HtmlFuncMap()).ParseFiles(cfg.Templates.TargetList)
	if err != nil {
		// ! TODO Need to manage internal server error
		(*logger).Errorln(err)
		return
	}

	// Generate template in buffer
	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, struct{ Targets []*config.Target }{Targets: cfg.Targets})
	if err != nil {
		// ! TODO Need to manage internal server error
		(*logger).Errorln(err)
		return
	}
	// Set the header and write the buffer to the http.ResponseWriter
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(rw)
}
