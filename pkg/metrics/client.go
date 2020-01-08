package metrics

import "net/http"

// Client Client metrics interface
type Client interface {
	Instrument() func(next http.Handler) http.Handler
	GetExposeHandler() http.Handler
	IncS3Operations(operation string)
}

// NewClient will generate a new client instance
func NewClient() Client {
	client := &prometheusClient{}
	// Call register to create all prometheus instances objects
	client.register()

	return client
}
