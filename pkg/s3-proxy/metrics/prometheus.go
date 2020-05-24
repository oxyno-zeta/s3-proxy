package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type prometheusClient struct {
	reqCnt            *prometheus.CounterVec
	resSz             prometheus.Summary
	reqDur            prometheus.Summary
	reqSz             prometheus.Summary
	up                prometheus.Gauge
	s3OperationsTotal *prometheus.CounterVec
}

// Instrument will instrument gin routes
func (ctx *prometheusClient) Instrument() func(next http.Handler) http.Handler {
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

// GetExposeHandler Get handler to expose metrics for resquest
func (ctx *prometheusClient) GetExposeHandler() http.Handler {
	return promhttp.Handler()
}

// IncS3Operations Increment s3 operation counter
func (ctx *prometheusClient) IncS3Operations(operation string) {
	ctx.s3OperationsTotal.WithLabelValues(operation).Inc()
}

func (ctx *prometheusClient) register() {
	ctx.reqCnt = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "How many HTTP requests processed, partitioned by status code and HTTP method.",
		},
		[]string{"status_code", "method", "host", "path"},
	)
	prometheus.MustRegister(ctx.reqCnt)

	ctx.reqDur = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name: "http_request_duration_seconds",
			Help: "The HTTP request latencies in seconds.",
		},
	)
	prometheus.MustRegister(ctx.reqDur)

	ctx.reqSz = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name: "http_request_size_bytes",
			Help: "The HTTP request sizes in bytes.",
		},
	)
	prometheus.MustRegister(ctx.reqSz)

	ctx.resSz = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name: "http_response_size_bytes",
			Help: "The HTTP response sizes in bytes.",
		},
	)
	prometheus.MustRegister(ctx.resSz)

	ctx.up = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "up",
			Help: "1 = up, 0 = down",
		},
	)
	ctx.up.Set(1)
	prometheus.MustRegister(ctx.up)

	ctx.s3OperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "s3_operations_total",
			Help: "How many operations are generated to s3 in total ?",
		},
		[]string{"operation"},
	)
	prometheus.MustRegister(ctx.s3OperationsTotal)
}
