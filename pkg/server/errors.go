package server

import (
	"fmt"
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/sirupsen/logrus"
)

func handleInternalServerError(rw http.ResponseWriter, err error, requestPath string, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig) {
	err2 := templateExecution(tplCfg.InternalServerError, logger, rw, struct {
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
		rw.Write([]byte(res))
	}
}

func handleNotFound(rw http.ResponseWriter, requestPath string, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig) {
	err := templateExecution(tplCfg.NotFound, logger, rw, struct{ Path string }{Path: requestPath}, http.StatusNotFound)
	if err != nil {
		(*logger).Errorln(err)
		handleInternalServerError(rw, err, requestPath, logger, tplCfg)
		return
	}
}

func handleUnauthorized(rw http.ResponseWriter, requestPath string, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig) {
	err := templateExecution(tplCfg.Unauthorized, logger, rw, struct{ Path string }{Path: requestPath}, http.StatusUnauthorized)
	if err != nil {
		(*logger).Errorln(err)
		handleInternalServerError(rw, err, requestPath, logger, tplCfg)
		return
	}
}

func handleBadRequest(rw http.ResponseWriter, requestPath string, err error, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig) {
	err2 := templateExecution(tplCfg.BadRequest, logger, rw, struct {
		Path  string
		Error error
	}{Path: requestPath, Error: err}, http.StatusBadRequest)
	if err2 != nil {
		(*logger).Errorln(err2)
		handleInternalServerError(rw, err2, requestPath, logger, tplCfg)
		return
	}
}

func handleForbidden(rw http.ResponseWriter, requestPath string, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig) {
	err := templateExecution(tplCfg.Forbidden, logger, rw, struct {
		Path string
	}{Path: requestPath}, http.StatusForbidden)
	if err != nil {
		(*logger).Errorln(err)
		handleInternalServerError(rw, err, requestPath, logger, tplCfg)
		return
	}
}
