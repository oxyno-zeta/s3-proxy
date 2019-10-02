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
	r.Use(middleware.DefaultCompress)
	r.Use(middleware.NoCache)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(NewStructuredLogger(logger))
	r.Use(middleware.Recoverer)

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
				brctx, err := bucket.NewRequestContext(tgt, cfg.Templates, &logEntry, mountPath, requestPath, &rw, handleNotFound)

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

func handleNotFound(rw http.ResponseWriter, requestPath string, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig) {
	err := templateExecution(tplCfg.NotFound, logger, rw, struct{ Path string }{Path: requestPath})
	if err != nil {
		// ! TODO Need to manage internal server error
		(*logger).Errorln(err)
		return
	}
}

func generateTargetList(rw http.ResponseWriter, logger *logrus.FieldLogger, cfg *config.Config) {
	err := templateExecution(cfg.Templates.TargetList, logger, rw, struct{ Targets []*config.Target }{Targets: cfg.Targets})
	if err != nil {
		// ! TODO Need to manage internal server error
		(*logger).Errorln(err)
		return
	}
}

func templateExecution(tplPath string, logger *logrus.FieldLogger, rw http.ResponseWriter, data interface{}) error {
	// Load template
	tplFileName := filepath.Base(tplPath)
	tmpl, err := template.New(tplFileName).Funcs(sprig.HtmlFuncMap()).ParseFiles(tplPath)
	if err != nil {
		return err
	}

	// Generate template in buffer
	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, data)
	if err != nil {
		return err
	}
	// Set the header and write the buffer to the http.ResponseWriter
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(rw)
	return nil
}
