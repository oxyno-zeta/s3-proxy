package tracing

import (
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

func GetTraceFromRequest(r *http.Request) Trace {
	sp := opentracing.SpanFromContext(r.Context())
	if sp == nil {
		return nil
	}

	return &trace{
		span: sp,
	}
}
