package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/deicod/erm/internal/observability/metrics"
	"github.com/deicod/erm/internal/observability/tracing"
)

// QueryOperation identifies the runtime operation being executed.
type QueryOperation string

const (
	// OperationSelect captures SELECT statements constructed from SelectSpec.
	OperationSelect QueryOperation = "select"
	// OperationAggregate captures aggregate queries constructed from AggregateSpec.
	OperationAggregate QueryOperation = "aggregate"
)

// QueryLog describes the structured payload emitted for each ORM query.
type QueryLog struct {
	Operation     QueryOperation
	Table         string
	SQL           string
	Args          []any
	Duration      time.Duration
	Err           error
	CorrelationID string
}

// QueryLogger receives structured query events.
type QueryLogger interface {
	LogQuery(ctx context.Context, entry QueryLog)
}

// QueryLoggerFunc adapts plain functions to QueryLogger.
type QueryLoggerFunc func(context.Context, QueryLog)

// LogQuery implements QueryLogger.
func (fn QueryLoggerFunc) LogQuery(ctx context.Context, entry QueryLog) {
	if fn == nil {
		return
	}
	fn(ctx, entry)
}

// CorrelationProvider extracts correlation IDs from the request context.
type CorrelationProvider interface {
	CorrelationID(context.Context) string
}

// CorrelationProviderFunc adapts functions into CorrelationProvider implementations.
type CorrelationProviderFunc func(context.Context) string

// CorrelationID implements CorrelationProvider.
func (fn CorrelationProviderFunc) CorrelationID(ctx context.Context) string {
	if fn == nil {
		return ""
	}
	return fn(ctx)
}

// QueryObserver coordinates logging, metrics, and tracing for ORM queries.
type QueryObserver struct {
	Logger     QueryLogger
	Tracer     tracing.Tracer
	Collector  metrics.Collector
	Correlator CorrelationProvider
}

// Observe prepares an observation handle for the provided query. Call End on the
// returned observation once the driver call completes.
func (o QueryObserver) Observe(ctx context.Context, op QueryOperation, table, sql string, args []any) QueryObservation {
	if ctx == nil {
		ctx = context.Background()
	}
	tracer := o.Tracer
	if tracer == nil {
		tracer = tracing.NoopTracer{}
	}

	correlation := ""
	if o.Correlator != nil {
		correlation = o.Correlator.CorrelationID(ctx)
	}

	attrs := []tracing.Attribute{
		tracing.String("orm.table", table),
		tracing.String("orm.operation", string(op)),
		tracing.Int("orm.arg_count", len(args)),
	}
	if correlation != "" {
		attrs = append(attrs, tracing.String("erm.correlation_id", correlation))
	}

	spanName := fmt.Sprintf("orm.%s.%s", table, op)
	spanCtx, span := tracer.Start(ctx, spanName, attrs...)

	obs := QueryObservation{
		ctx:           spanCtx,
		baseCtx:       ctx,
		start:         time.Now(),
		op:            op,
		table:         table,
		sql:           sql,
		span:          span,
		collector:     o.Collector,
		logger:        o.Logger,
		correlationID: correlation,
	}
	if o.Logger != nil {
		obs.args = append([]any(nil), args...)
	}
	return obs
}

// QueryObservation tracks a single in-flight ORM query.
type QueryObservation struct {
	ctx           context.Context
	baseCtx       context.Context
	start         time.Time
	op            QueryOperation
	table         string
	sql           string
	args          []any
	span          tracing.Span
	collector     metrics.Collector
	logger        QueryLogger
	correlationID string
}

// Context returns the context propagated to the driver call.
func (obs QueryObservation) Context() context.Context {
	if obs.ctx != nil {
		return obs.ctx
	}
	if obs.baseCtx != nil {
		return obs.baseCtx
	}
	return context.Background()
}

// End finalises the observation, emitting logs, metrics, and span completion.
func (obs QueryObservation) End(err error) {
	if obs.span != nil {
		obs.span.End(err)
	}
	duration := time.Since(obs.start)

	if obs.collector != nil {
		obs.collector.RecordQuery(obs.table, string(obs.op), duration, err)
	}
	if obs.logger != nil {
		obs.logger.LogQuery(obs.Context(), QueryLog{
			Operation:     obs.op,
			Table:         obs.table,
			SQL:           obs.sql,
			Args:          obs.args,
			Duration:      duration,
			Err:           err,
			CorrelationID: obs.correlationID,
		})
	}
}
