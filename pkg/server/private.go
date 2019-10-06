package server

import (
	"bytes"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/Masterminds/sprig"
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
