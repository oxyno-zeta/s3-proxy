package utils

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/Masterminds/sprig"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/sirupsen/logrus"
)

func HandleInternalServerError(rw http.ResponseWriter, err error, requestPath string, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig) {
	err2 := TemplateExecution(tplCfg.InternalServerError, logger, rw, struct {
		Path  string
		Error error
	}{Path: requestPath, Error: err}, http.StatusInternalServerError)
	if err2 != nil {
		// New error
		(*logger).Errorln(err2)
		// Template error
		res := fmt.Sprintf(`
<!DOCTYPE html>
<html>
  <body>
	<h1>Internal Server Error</h1>
	<p>%s</p>
  </body>
</html>
`, err2)

		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(res))
	}
}

func HandleNotFound(rw http.ResponseWriter, requestPath string, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig) {
	err := TemplateExecution(tplCfg.NotFound, logger, rw, struct{ Path string }{Path: requestPath}, http.StatusNotFound)
	if err != nil {
		(*logger).Errorln(err)
		HandleInternalServerError(rw, err, requestPath, logger, tplCfg)
	}
}

func HandleUnauthorized(rw http.ResponseWriter, requestPath string, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig) {
	err := TemplateExecution(tplCfg.Unauthorized, logger, rw, struct{ Path string }{Path: requestPath}, http.StatusUnauthorized)
	if err != nil {
		(*logger).Errorln(err)
		HandleInternalServerError(rw, err, requestPath, logger, tplCfg)
	}
}

func HandleBadRequest(rw http.ResponseWriter, requestPath string, err error, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig) {
	err2 := TemplateExecution(tplCfg.BadRequest, logger, rw, struct {
		Path  string
		Error error
	}{Path: requestPath, Error: err}, http.StatusBadRequest)
	if err2 != nil {
		(*logger).Errorln(err2)
		HandleInternalServerError(rw, err2, requestPath, logger, tplCfg)
	}
}

func HandleForbidden(rw http.ResponseWriter, requestPath string, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig) {
	err := TemplateExecution(tplCfg.Forbidden, logger, rw, struct {
		Path string
	}{Path: requestPath}, http.StatusForbidden)
	if err != nil {
		(*logger).Errorln(err)
		HandleInternalServerError(rw, err, requestPath, logger, tplCfg)
	}
}

func ClientIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}

	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}

	return IPAddress
}

func TemplateExecution(tplPath string, logger *logrus.FieldLogger, rw http.ResponseWriter, data interface{}, status int) error {
	// Load template
	tplFileName := filepath.Base(tplPath)
	tmpl, err := template.New(tplFileName).Funcs(sprig.HtmlFuncMap()).ParseFiles(tplPath)
	// Check if error exists
	if err != nil {
		return err
	}

	// Generate template in buffer
	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, data)
	// Check if error exists
	if err != nil {
		return err
	}
	// Set status code
	rw.WriteHeader(status)
	// Set the header and write the buffer to the http.ResponseWriter
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = buf.WriteTo(rw)
	// Check if error exists
	if err != nil {
		return err
	}

	return nil
}
