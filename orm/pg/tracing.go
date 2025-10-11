package pg

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/deicod/erm/observability/tracing"
)

func newPGXTracer(tracer tracing.Tracer) pgx.QueryTracer {
	if tracer == nil {
		return nil
	}
	return &pgxTracer{tracer: tracer}
}

type pgxTracer struct {
	tracer tracing.Tracer
}

type spanCtxKey struct{}

func (t *pgxTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if t == nil || t.tracer == nil {
		return ctx
	}
	ctx, span := t.tracer.Start(ctx, "pgx.query",
		tracing.String("sql", data.SQL),
		tracing.Int("arg_count", len(data.Args)),
	)
	return context.WithValue(ctx, spanCtxKey{}, span)
}

func (t *pgxTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if span, ok := ctx.Value(spanCtxKey{}).(tracing.Span); ok && span != nil {
		span.End(data.Err)
	}
}
