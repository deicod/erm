package tracing

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type testSpan struct {
	endCount int
}

func (s *testSpan) End(err error) {
	s.endCount++
}

type testTracer struct {
	starts int
}

func (t *testTracer) Start(ctx context.Context, name string, attrs ...Attribute) (context.Context, Span) {
	t.starts++
	return ctx, &testSpan{}
}

func TestWithTracerFanout(t *testing.T) {
	primary := &testTracer{}
	secondary := &testTracer{}
	tracer := WithTracer(primary, secondary)
	ctx, span := tracer.Start(context.Background(), "operation")
	if ctx == nil {
		t.Fatalf("expected context")
	}
	span.End(nil)
	if primary.starts != 1 || secondary.starts != 1 {
		t.Fatalf("expected both tracers to start once, got %d and %d", primary.starts, secondary.starts)
	}
}

func TestOTelTracer(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))
	tracer := NewOTelTracer(tp, "test")
	ctx, span := tracer.Start(context.Background(), "db.query", String("sql", "select 1"))
	if ctx == nil {
		t.Fatalf("expected context from tracer")
	}
	span.End(errors.New("boom"))
	spans := spanRecorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected one span recorded, got %d", len(spans))
	}
	if spans[0].Name() != "db.query" {
		t.Fatalf("unexpected span name %q", spans[0].Name())
	}
}
