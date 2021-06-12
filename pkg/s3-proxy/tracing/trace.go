package tracing

import (
	"context"
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
)

type trace struct {
	span opentracing.Span
}

func (t *trace) SetTag(key string, value interface{}) {
	t.span.SetTag(key, value)
}

func (t *trace) GetChildTrace(operationName string) Trace {
	tracer := opentracing.GlobalTracer()

	childSpan := tracer.StartSpan(
		operationName,
		opentracing.ChildOf(t.span.Context()),
	)

	return &trace{span: childSpan}
}

func (t *trace) Finish() {
	t.span.Finish()
}

func (t *trace) GetTraceID() string {
	if sc, ok := t.span.Context().(jaeger.SpanContext); ok {
		return sc.TraceID().String()
	}

	return ""
}

func GetTraceFromContext(ctx context.Context) Trace {
	sp := opentracing.SpanFromContext(ctx)
	if sp == nil {
		return nil
	}

	return &trace{
		span: sp,
	}
}

func GetTraceIDFromRequest(r *http.Request) string {
	// Get request trace
	trace := GetTraceFromContext(r.Context())
	if trace != nil {
		return trace.GetTraceID()
	}

	return ""
}
