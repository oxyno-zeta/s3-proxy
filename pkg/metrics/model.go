package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

// Instance Instance metrics interface
type Instance interface {
	Instrument() func(next http.Handler) http.Handler
	GetPrometheusHandler() http.Handler
	IncS3Operations(operation string)
}

type instance struct {
	reqCnt            *prometheus.CounterVec
	resSz             prometheus.Summary
	reqDur            prometheus.Summary
	reqSz             prometheus.Summary
	up                prometheus.Gauge
	s3OperationsTotal *prometheus.CounterVec
}

// NewInstance will generate a new Instance
func NewInstance() Instance {
	ctx := &instance{}
	// Call register to create all prometheus instances objects
	ctx.register()

	return ctx
}
