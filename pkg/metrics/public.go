package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	return n, err
}

// Instrument will instrument gin routes
func (ctx *instance) Instrument() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Begin timer
			start := time.Now()
			// Calculate request size
			reqSz := computeApproximateRequestSize(r)

			// Next request with new response writer
			sw := statusWriter{ResponseWriter: w}
			next.ServeHTTP(&sw, r)

			// Get status as string
			status := strconv.Itoa(sw.status)
			// Calculate request time
			elapsed := float64(time.Since(start)) / float64(time.Second)
			// Get response size
			resSz := float64(sw.length)

			// Manage prometheus metrics
			ctx.reqDur.Observe(elapsed)
			ctx.reqCnt.WithLabelValues(status, r.Method, r.Host, r.URL.Path).Inc()
			ctx.reqSz.Observe(float64(reqSz))
			ctx.resSz.Observe(resSz)
		})
	}
}

// GetPrometheusHandler Get Prometheus handler for resquest
func (ctx *instance) GetPrometheusHandler() http.Handler {
	return promhttp.Handler()
}

// IncS3Operations Increment s3 operation counter
func (ctx *instance) IncS3Operations(operation string) {
	ctx.s3OperationsTotal.WithLabelValues(operation).Inc()
}
