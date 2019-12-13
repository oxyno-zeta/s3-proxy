package server

import (
	"bytes"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/Masterminds/sprig"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/sirupsen/logrus"
)

func templateExecution(tplPath string, logger *logrus.FieldLogger, rw http.ResponseWriter, data interface{}, status int) error {
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
	rw.WriteHeader(status)
	// Set the header and write the buffer to the http.ResponseWriter
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = buf.WriteTo(rw)
	if err != nil {
		return err
	}
	return nil
}

func clientIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
}

func generateTargetList(rw http.ResponseWriter, logger *logrus.FieldLogger, cfg *config.Config) {
	// TODO Manage new host and path list
	err := templateExecution(cfg.Templates.TargetList, logger, rw, struct{ Targets []*config.Target }{Targets: cfg.Targets}, 200)
	if err != nil {
		(*logger).Errorln(err)
		handleInternalServerError(rw, err, "/", logger, cfg.Templates)
		return
	}
}
