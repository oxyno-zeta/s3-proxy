package metrics

import (
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

// Client Client metrics interface.
//
//go:generate mockgen -destination=./mocks/mock_Client.go -package=mocks github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics Client
type Client interface {
	// Will return a middleware to instrument http routers.
	Instrument(serverLabel string, metricsCfg *config.MetricsConfig) func(next http.Handler) http.Handler
	// Will return a handler to expose metrics over a http server.
	GetExposeHandler() http.Handler
	// Will increase counter of S3 operations done by service.
	IncS3Operations(targetName, bucketName, operation string)
	// Will increase counter of authenticated user.
	IncAuthenticated(providerType, providerName string)
	// Will increase counter of authorized user.
	IncAuthorized(providerType string)
	// Will increase counter of succeed webhooks
	IncSucceedWebhooks(targetName, actionName string)
	// Will increase counter of failed webhooks
	IncFailedWebhooks(targetName, actionName string)
}

// NewClient will generate a new client instance.
func NewClient() Client {
	client := &prometheusClient{}
	// Call register to create all prometheus instances objects
	client.register()

	return client
}
