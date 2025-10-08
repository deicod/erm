package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// NewOTelTracer wires the provided provider (or global fallback) into the tracing interface.
func NewOTelTracer(provider trace.TracerProvider, instrumentationName string) Tracer {
	if instrumentationName == "" {
		instrumentationName = "github.com/deicod/erm"
	}
	if provider == nil {
		provider = otel.GetTracerProvider()
	}
	return &otelTracer{tracer: provider.Tracer(instrumentationName)}
}

type otelTracer struct {
	tracer trace.Tracer
}

func (t *otelTracer) Start(ctx context.Context, name string, attrs ...Attribute) (context.Context, Span) {
	if t == nil || t.tracer == nil {
		return ctx, noopSpan{}
	}
	ctx, span := t.tracer.Start(ctx, name, trace.WithAttributes(toOTelAttrs(attrs)...))
	return ctx, otelSpan{span: span}
}

type otelSpan struct {
	span trace.Span
}

func (s otelSpan) End(err error) {
	if err != nil {
		s.span.RecordError(err)
	}
	s.span.End()
}

func toOTelAttrs(attrs []Attribute) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}
	out := make([]attribute.KeyValue, 0, len(attrs))
	for _, attr := range attrs {
		if attr.Key == "" {
			continue
		}
		switch v := attr.Value.(type) {
		case string:
			out = append(out, attribute.String(attr.Key, v))
		case fmt.Stringer:
			out = append(out, attribute.String(attr.Key, v.String()))
		case bool:
			out = append(out, attribute.Bool(attr.Key, v))
		case int:
			out = append(out, attribute.Int(attr.Key, v))
		case int64:
			out = append(out, attribute.Int64(attr.Key, v))
		case float64:
			out = append(out, attribute.Float64(attr.Key, v))
		case float32:
			out = append(out, attribute.Float64(attr.Key, float64(v)))
		default:
			out = append(out, attribute.String(attr.Key, fmt.Sprint(v)))
		}
	}
	return out
}
