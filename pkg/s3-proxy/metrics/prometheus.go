package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type prometheusClient struct {
	reqCnt             *prometheus.CounterVec
	resSz              *prometheus.SummaryVec
	reqDur             *prometheus.SummaryVec
	reqSz              *prometheus.SummaryVec
	up                 *prometheus.GaugeVec
	s3OperationsTotal  *prometheus.CounterVec
	authenticatedTotal *prometheus.CounterVec
	authorizedTotal    *prometheus.CounterVec
}

// Instrument will instrument gin routes.
func (ctx *prometheusClient) Instrument(serverLabel string) func(next http.Handler) http.Handler {
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
			ctx.reqDur.WithLabelValues(serverLabel, status, r.Method, r.Host, r.URL.Path).Observe(elapsed)
			ctx.reqCnt.WithLabelValues(serverLabel, status, r.Method, r.Host, r.URL.Path).Inc()
			ctx.reqSz.WithLabelValues(serverLabel, status, r.Method, r.Host, r.URL.Path).Observe(float64(reqSz))
			ctx.resSz.WithLabelValues(serverLabel, status, r.Method, r.Host, r.URL.Path).Observe(resSz)
		})
	}
}

// GetExposeHandler Get handler to expose metrics for resquest.
func (ctx *prometheusClient) GetExposeHandler() http.Handler {
	return promhttp.Handler()
}

// IncS3Operations Increment s3 operation counter.
func (ctx *prometheusClient) IncS3Operations(targetName, bucketName, operation string) {
	ctx.s3OperationsTotal.WithLabelValues(targetName, bucketName, operation).Inc()
}

// Will increase counter of authenticated user.
func (ctx *prometheusClient) IncAuthenticated(providerType, providerName string) {
	ctx.authenticatedTotal.WithLabelValues(providerType, providerName).Inc()
}

// Will increase counter of authorized user.
func (ctx *prometheusClient) IncAuthorized(providerType string) {
	ctx.authorizedTotal.WithLabelValues(providerType).Inc()
}

func (ctx *prometheusClient) register() {
	ctx.reqCnt = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "How many HTTP requests have been processed ?",
		},
		[]string{"server", "status_code", "method", "host", "path"},
	)
	prometheus.MustRegister(ctx.reqCnt)

	ctx.reqDur = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_request_duration_seconds",
			Help: "The HTTP request latencies in seconds.",
		},
		[]string{"server", "status_code", "method", "host", "path"},
	)
	prometheus.MustRegister(ctx.reqDur)

	ctx.reqSz = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_request_size_bytes",
			Help: "The HTTP request sizes in bytes.",
		},
		[]string{"server", "status_code", "method", "host", "path"},
	)
	prometheus.MustRegister(ctx.reqSz)

	ctx.resSz = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_response_size_bytes",
			Help: "The HTTP response sizes in bytes.",
		},
		[]string{"server", "status_code", "method", "host", "path"},
	)
	prometheus.MustRegister(ctx.resSz)

	ctx.up = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "up",
			Help: "1 = up, 0 = down",
		},
		[]string{"component"},
	)
	ctx.up.WithLabelValues("s3-proxy").Set(1)
	prometheus.MustRegister(ctx.up)

	ctx.s3OperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "s3_operations_total",
			Help: "How many operations are generated to s3 in total ?",
		},
		[]string{"target_name", "bucket_name", "operation"},
	)
	prometheus.MustRegister(ctx.s3OperationsTotal)

	ctx.authenticatedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authenticated_total",
			Help: "How many users have been authenticated ?",
		},
		[]string{"provider_type", "provider_name"},
	)
	prometheus.MustRegister(ctx.authenticatedTotal)

	ctx.authorizedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authorized_total",
			Help: "How many users have been authorized ?",
		},
		[]string{"provider_type"},
	)
	prometheus.MustRegister(ctx.authorizedTotal)
}
