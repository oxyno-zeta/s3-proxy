package log

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

// HTTPAddLoggerToContextMiddleware HTTP Middleware that will add request logger to request context.
func HTTPAddLoggerToContextMiddleware() func(next http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// Get logger from request
			logger := getLogEntry(r)
			// Add logger to request context in order to keep it
			ctx := context.WithValue(r.Context(), loggerContextKey, logger)
			// Create new request with new context
			r = r.WithContext(ctx)

			// Next
			h.ServeHTTP(rw, r)
		})
	}
}

// Copied and modified from https://github.com/go-chi/chi/blob/master/_examples/logging/main.go

// NewStructuredLogger Generate a new structured logger.
func NewStructuredLogger(
	logger Logger,
	getTraceID func(r *http.Request) string,
	getClientIP func(r *http.Request) string,
	getRequestURI func(r *http.Request) string,
) func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&StructuredLogger{
		Logger:        logger,
		GetTraceID:    getTraceID,
		GetClientIP:   getClientIP,
		GetRequestURI: getRequestURI,
	})
}

// StructuredLogger structured logger.
type StructuredLogger struct {
	Logger        Logger
	GetTraceID    func(r *http.Request) string
	GetClientIP   func(r *http.Request) string
	GetRequestURI func(r *http.Request) string
}

// NewLogEntry new log entry.
func (l *StructuredLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	entry := &StructuredLoggerEntry{Logger: l.Logger}
	logFields := map[string]interface{}{}

	// Get trace id
	traceIDStr := l.GetTraceID(r)
	if traceIDStr != "" {
		logFields["span_id"] = traceIDStr
	}

	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		logFields["req_id"] = reqID
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	logFields["http_scheme"] = scheme
	logFields["http_proto"] = r.Proto
	logFields["http_method"] = r.Method

	logFields["remote_addr"] = r.RemoteAddr
	logFields["user_agent"] = r.UserAgent()
	logFields["client_ip"] = l.GetClientIP(r)

	logFields["uri"] = l.GetRequestURI(r)

	entry.Logger = entry.Logger.WithFields(logFields)

	entry.Logger.Debug("request started")

	return entry
}

// StructuredLoggerEntry Structured logger entry.
type StructuredLoggerEntry struct {
	Logger Logger
}

// Write Write.
func (l *StructuredLoggerEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	l.Logger = l.Logger.WithFields(logrus.Fields{
		"resp_status":       status,
		"resp_bytes_length": bytes,
		"resp_elapsed_ms":   float64(elapsed.Nanoseconds()) / 1000000.0, // nolint: gomnd // No constant for that
	})
	logFunc := l.Logger.Infoln
	// Check status code for warn logger
	if status >= http.StatusMultipleChoices && status < http.StatusBadRequest {
		logFunc = l.Logger.Warnln
	}
	// Check status code for error logger
	if status >= http.StatusBadRequest {
		logFunc = l.Logger.Errorln
	}

	logFunc("request complete")
}

// Panic panic log.
func (l *StructuredLoggerEntry) Panic(v interface{}, stack []byte) {
	l.Logger = l.Logger.WithFields(logrus.Fields{
		"stack": string(stack),
		"panic": fmt.Sprintf("%+v", v),
	})
}

// Helper methods used by the application to get the request-scoped
// logger entry and set additional fields between handlers.
//
// This is a useful pattern to use to set state on the entry as it
// passes through the handler chain, which at any point can be logged
// with a call to .Print(), .Info(), etc.

// GetLogEntry get log entry.
func getLogEntry(r *http.Request) Logger {
	entry := middleware.GetLogEntry(r).(*StructuredLoggerEntry)

	return entry.Logger
}
