package metrics

import (
	"net/http"

	"github.com/pkg/errors"
)

type statusWriter struct {
	http.ResponseWriter
	status int
	length int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		// Set status if doesn't exists
		w.status = http.StatusOK
	}
	// Write with real response writer
	n, err := w.ResponseWriter.Write(b)
	// Increase length
	w.length += n
	// Return result
	return n, errors.WithStack(err)
}
