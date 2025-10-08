package pg

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/deicod/erm/internal/observability/tracing"
)

type recordingTracer struct {
	starts int
	ends   int
}

func (r *recordingTracer) Start(ctx context.Context, name string, attrs ...tracing.Attribute) (context.Context, tracing.Span) {
	r.starts++
	return ctx, recordingSpan{rec: r}
}

type recordingSpan struct {
	rec *recordingTracer
}

func (r recordingSpan) End(err error) {
	if r.rec != nil {
		r.rec.ends++
	}
}

func TestPGXTracerQueryLifecycle(t *testing.T) {
	rec := &recordingTracer{}
	tracer := newPGXTracer(rec)
	ctx := context.Background()
	startData := pgx.TraceQueryStartData{SQL: "select 1", Args: []any{1}}
	ctx = tracer.TraceQueryStart(ctx, nil, startData)
	if rec.starts != 1 {
		t.Fatalf("expected tracer start counter, got %d", rec.starts)
	}
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{Err: errors.New("boom")})
	if rec.ends != 1 {
		t.Fatalf("expected tracer end counter, got %d", rec.ends)
	}
}
