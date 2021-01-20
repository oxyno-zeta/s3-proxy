package metrics

import "net/http"

// Client Client metrics interface.
type Client interface {
	// Will return a middleware to instrument http routers.
	Instrument(serverLabel string) func(next http.Handler) http.Handler
	// Will return a handler to expose metrics over a http server.
	GetExposeHandler() http.Handler
	// Will increase counter of S3 operations done by service.
	IncS3Operations(targetName, bucketName, operation string)
	// Will increase counter of authenticated user.
	IncAuthenticated(providerType, providerName string)
	// Will increase counter of authorized user.
	IncAuthorized(providerType string)
}

// NewClient will generate a new client instance.
func NewClient() Client {
	client := &prometheusClient{}
	// Call register to create all prometheus instances objects
	client.register()

	return client
}
