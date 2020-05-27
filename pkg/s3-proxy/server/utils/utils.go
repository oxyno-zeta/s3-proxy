package utils

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/Masterminds/sprig"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
)

// HandleInternalServerErrorWithTemplate Handle internal server error following response template given in parameter
// nolint:whitespace
func HandleInternalServerErrorWithTemplate(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string, err error) {
	err2 := TemplateExecution(tplCfg.InternalServerError, tplString, logger, rw, struct {
		Path  string
		Error error
	}{Path: requestPath, Error: err}, http.StatusInternalServerError)
	if err2 != nil {
		// New error
		logger.Error(err2)
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

		// Set the header and write the buffer to the http.ResponseWriter
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(res))
	}
}

// HandleInternalServerError Handle internal server error following response template
// nolint:whitespace
func HandleInternalServerError(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, requestPath string, err error) {
	HandleInternalServerErrorWithTemplate(logger, rw, tplCfg, "", requestPath, err)
}

// HandleNotFoundWithTemplate Handle not found error following response template with given template in parameter
func HandleNotFoundWithTemplate(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string) {
	err := TemplateExecution(tplCfg.NotFound, tplString, logger, rw, struct{ Path string }{Path: requestPath}, http.StatusNotFound)
	if err != nil {
		logger.Error(err)
		HandleInternalServerError(logger, rw, tplCfg, requestPath, err)
	}
}

// HandleNotFound Handle not found error following response template
func HandleNotFound(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, requestPath string) {
	HandleNotFoundWithTemplate(logger, rw, tplCfg, "", requestPath)
}

// HandleUnauthorized Handle unauthorized error following response template with given template in parameter
func HandleUnauthorizedWithTemplate(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string) {
	err := TemplateExecution(tplCfg.Unauthorized, tplString, logger, rw, struct{ Path string }{Path: requestPath}, http.StatusUnauthorized)
	if err != nil {
		logger.Error(err)
		HandleInternalServerError(logger, rw, tplCfg, requestPath, err)
	}
}

// HandleUnauthorized Handle unauthorized error following response template
func HandleUnauthorized(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, requestPath string) {
	HandleUnauthorizedWithTemplate(logger, rw, tplCfg, "", requestPath)
}

// HandleBadRequest Handle bad request error following response template with given template in parameter
// nolint:whitespace
func HandleBadRequestWithTemplate(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string, err error) {
	err2 := TemplateExecution(tplCfg.BadRequest, "", logger, rw, struct {
		Path  string
		Error error
	}{Path: requestPath, Error: err}, http.StatusBadRequest)
	if err2 != nil {
		logger.Error(err2)
		HandleInternalServerError(logger, rw, tplCfg, requestPath, err2)
	}
}

// HandleBadRequest Handle bad request error following response template
func HandleBadRequest(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, requestPath string, err error) {
	HandleBadRequestWithTemplate(logger, rw, tplCfg, "", requestPath, err)
}

// HandleForbiddenWithTemplate Handle forbidden error following response template given in parameters
func HandleForbiddenWithTemplate(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, tplString string, requestPath string) {
	err := TemplateExecution(tplCfg.Forbidden, tplString, logger, rw, struct {
		Path string
	}{Path: requestPath}, http.StatusForbidden)
	if err != nil {
		logger.Error(err)
		HandleInternalServerError(logger, rw, tplCfg, requestPath, err)
	}
}

// HandleForbidden Handle forbidden error following response template
func HandleForbidden(logger log.Logger, rw http.ResponseWriter, tplCfg *config.TemplateConfig, requestPath string) {
	HandleForbiddenWithTemplate(logger, rw, tplCfg, "", requestPath)
}

// ClientIP will return client ip from request
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

// TemplateExecution will execute template with values and interpret response as html content
func TemplateExecution(tplPath, tplString string, logger log.Logger, rw http.ResponseWriter, data interface{}, status int) error {
	// Set status code
	rw.WriteHeader(status)
	// Set the header and write the buffer to the http.ResponseWriter
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Initialize variables
	var err error

	var tmpl *template.Template

	// Check if it a template file
	if tplString != "" {
		// Load template from string
		tmpl, err = template.New("template-string-loaded").Funcs(sprig.HtmlFuncMap()).Parse(tplString)
	} else {
		// Load template from file
		tplFileName := filepath.Base(tplPath)
		tmpl, err = template.New(tplFileName).Funcs(sprig.HtmlFuncMap()).ParseFiles(tplPath)
	}

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

	_, err = buf.WriteTo(rw)
	// Check if error exists
	if err != nil {
		return err
	}

	return nil
}

func GetRequestURI(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)
}
