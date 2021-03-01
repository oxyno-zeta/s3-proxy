package middlewares

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/utils"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
	"github.com/sirupsen/logrus"
)

// Copied and modified from https://github.com/go-chi/chi/blob/master/_examples/logging/main.go

// NewStructuredLogger Generate a new structured logger.
func NewStructuredLogger(logger log.Logger) func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&StructuredLogger{logger})
}

// StructuredLogger structured logger.
type StructuredLogger struct {
	Logger log.Logger
}

// NewLogEntry new log entry.
func (l *StructuredLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	entry := &StructuredLoggerEntry{Logger: l.Logger}
	logFields := map[string]interface{}{}

	// Get request trace
	trace := tracing.GetTraceFromRequest(r)
	if trace != nil {
		traceIDStr := trace.GetTraceID()
		if traceIDStr != "" {
			logFields["span_id"] = traceIDStr
		}
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
	logFields["client_ip"] = utils.ClientIP(r)

	logFields["uri"] = utils.GetRequestURI(r)

	entry.Logger = entry.Logger.WithFields(logFields)

	entry.Logger.Debug("request started")

	return entry
}

// StructuredLoggerEntry Structured logger entry.
type StructuredLoggerEntry struct {
	Logger log.Logger
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
func GetLogEntry(r *http.Request) log.Logger {
	entry := middleware.GetLogEntry(r).(*StructuredLoggerEntry)

	return entry.Logger
}

// LogEntrySetField Log entry set field.
func LogEntrySetField(r *http.Request, key string, value interface{}) {
	if entry, ok := r.Context().Value(middleware.LogEntryCtxKey).(*StructuredLoggerEntry); ok {
		entry.Logger = entry.Logger.WithField(key, value)
	}
}

// LogEntrySetFields Log entry set fields.
func LogEntrySetFields(r *http.Request, fields map[string]interface{}) {
	if entry, ok := r.Context().Value(middleware.LogEntryCtxKey).(*StructuredLoggerEntry); ok {
		entry.Logger = entry.Logger.WithFields(fields)
	}
}
