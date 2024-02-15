package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
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
	succeedWebhooks    *prometheus.CounterVec
	failedWebhooks     *prometheus.CounterVec
}

// Instrument will instrument gin routes.
func (cl *prometheusClient) Instrument(serverLabel string, metricsCfg *config.MetricsConfig) func(next http.Handler) http.Handler {
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

			// Init path
			path := r.URL.Path
			// Check if router path metrics is disabled
			if metricsCfg != nil && metricsCfg.DisableRouterPath {
				path = ""
			}

			// Manage prometheus metrics
			cl.reqDur.WithLabelValues(serverLabel, status, r.Method, r.Host, path).Observe(elapsed)
			cl.reqCnt.WithLabelValues(serverLabel, status, r.Method, r.Host, path).Inc()
			cl.reqSz.WithLabelValues(serverLabel, status, r.Method, r.Host, path).Observe(float64(reqSz))
			cl.resSz.WithLabelValues(serverLabel, status, r.Method, r.Host, path).Observe(resSz)
		})
	}
}

// GetExposeHandler Get handler to expose metrics for resquest.
func (*prometheusClient) GetExposeHandler() http.Handler {
	return promhttp.Handler()
}

// IncS3Operations Increment s3 operation counter.
func (cl *prometheusClient) IncS3Operations(targetName, bucketName, operation string) {
	cl.s3OperationsTotal.WithLabelValues(targetName, bucketName, operation).Inc()
}

// Will increase counter of authenticated user.
func (cl *prometheusClient) IncAuthenticated(providerType, providerName string) {
	cl.authenticatedTotal.WithLabelValues(providerType, providerName).Inc()
}

// Will increase counter of authorized user.
func (cl *prometheusClient) IncAuthorized(providerType string) {
	cl.authorizedTotal.WithLabelValues(providerType).Inc()
}

func (cl *prometheusClient) IncSucceedWebhooks(targetName, actionName string) {
	cl.succeedWebhooks.WithLabelValues(targetName, actionName).Inc()
}

func (cl *prometheusClient) IncFailedWebhooks(targetName, actionName string) {
	cl.failedWebhooks.WithLabelValues(targetName, actionName).Inc()
}

func (cl *prometheusClient) register() {
	cl.reqCnt = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "How many HTTP requests have been processed ?",
		},
		[]string{"server", "status_code", "method", "host", "path"},
	)
	prometheus.MustRegister(cl.reqCnt)

	cl.reqDur = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_request_duration_seconds",
			Help: "The HTTP request latencies in seconds.",
		},
		[]string{"server", "status_code", "method", "host", "path"},
	)
	prometheus.MustRegister(cl.reqDur)

	cl.reqSz = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_request_size_bytes",
			Help: "The HTTP request sizes in bytes.",
		},
		[]string{"server", "status_code", "method", "host", "path"},
	)
	prometheus.MustRegister(cl.reqSz)

	cl.resSz = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_response_size_bytes",
			Help: "The HTTP response sizes in bytes.",
		},
		[]string{"server", "status_code", "method", "host", "path"},
	)
	prometheus.MustRegister(cl.resSz)

	cl.up = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "up",
			Help: "1 = up, 0 = down",
		},
		[]string{"component"},
	)
	cl.up.WithLabelValues("s3-proxy").Set(1)
	prometheus.MustRegister(cl.up)

	cl.s3OperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "s3_operations_total",
			Help: "How many operations are generated to s3 in total ?",
		},
		[]string{"target_name", "bucket_name", "operation"},
	)
	prometheus.MustRegister(cl.s3OperationsTotal)

	cl.authenticatedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authenticated_total",
			Help: "How many users have been authenticated ?",
		},
		[]string{"provider_type", "provider_name"},
	)
	prometheus.MustRegister(cl.authenticatedTotal)

	cl.authorizedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authorized_total",
			Help: "How many users have been authorized ?",
		},
		[]string{"provider_type"},
	)
	prometheus.MustRegister(cl.authorizedTotal)

	cl.succeedWebhooks = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "succeed_webhooks_total",
			Help: "How many webhooks have been succeed ?",
		},
		[]string{"target_name", "action_name"},
	)
	prometheus.MustRegister(cl.succeedWebhooks)

	cl.failedWebhooks = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "failed_webhooks_total",
			Help: "How many webhooks have been failed ?",
		},
		[]string{"target_name", "action_name"},
	)
	prometheus.MustRegister(cl.failedWebhooks)
}
