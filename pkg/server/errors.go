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
	}{Path: requestPath, Error: err}, 500)
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
		rw.WriteHeader(500)
		rw.Write([]byte(res))
	}
}

func handleNotFound(rw http.ResponseWriter, requestPath string, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig) {
	err := templateExecution(tplCfg.NotFound, logger, rw, struct{ Path string }{Path: requestPath}, 404)
	if err != nil {
		(*logger).Errorln(err)
		handleInternalServerError(rw, err, requestPath, logger, tplCfg)
		return
	}
}

func handleUnauthorized(rw http.ResponseWriter, requestPath string, logger *logrus.FieldLogger, tplCfg *config.TemplateConfig) {
	err := templateExecution(tplCfg.Unauthorized, logger, rw, struct{ Path string }{Path: requestPath}, 401)
	if err != nil {
		(*logger).Errorln(err)
		handleInternalServerError(rw, err, requestPath, logger, tplCfg)
		return
	}
}
