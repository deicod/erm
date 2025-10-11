package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/deicod/erm/observability/tracing"
)

func TestBuildSelectSQL(t *testing.T) {
	spec := SelectSpec{
		Table:   "users",
		Columns: []string{"id", "email"},
		Predicates: []Predicate{
			{Column: "email", Operator: OpILike, Value: "%example.com"},
		},
		Orders: []Order{
			{Column: "created_at", Direction: SortDesc},
		},
		Limit:  25,
		Offset: 10,
	}

	sql, args := BuildSelectSQL(spec)
	expected := "SELECT id, email FROM users WHERE email ILIKE $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
	if sql != expected {
		t.Fatalf("unexpected SQL:\n got: %s\nwant: %s", sql, expected)
	}
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(args))
	}
	if args[0] != "%example.com" || args[1] != 25 || args[2] != 10 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildSelectSQLDefaults(t *testing.T) {
	spec := SelectSpec{Table: "users"}

	sql, args := BuildSelectSQL(spec)
	if sql != "SELECT * FROM users" {
		t.Fatalf("unexpected SQL: %s", sql)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args, got %d", len(args))
	}
}

func TestBuildAggregateSQL(t *testing.T) {
	spec := AggregateSpec{
		Table: "users",
		Predicates: []Predicate{
			{Column: "created_at", Operator: OpGTE, Value: "2024-01-01"},
		},
		Aggregate: Aggregate{Func: AggCount, Column: "id"},
	}

	sql, args := BuildAggregateSQL(spec)
	expected := "SELECT COUNT(id) FROM users WHERE created_at >= $1"
	if sql != expected {
		t.Fatalf("unexpected SQL:\n got: %s\nwant: %s", sql, expected)
	}
	if len(args) != 1 || args[0] != "2024-01-01" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildAggregateSQLDefaultColumn(t *testing.T) {
	spec := AggregateSpec{
		Table:     "users",
		Aggregate: Aggregate{Func: AggCount},
	}

	sql, args := BuildAggregateSQL(spec)
	if sql != "SELECT COUNT(*) FROM users" {
		t.Fatalf("unexpected SQL: %s", sql)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args, got %d", len(args))
	}
}

func TestQueryObserverEmitsTelemetry(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKey{}, "request-42")

	logger := &recordingLogger{}
	collector := &recordingCollector{}
	tracer := &recordingTracer{}

	observer := QueryObserver{
		Logger:    logger,
		Tracer:    tracer,
		Collector: collector,
		Correlator: CorrelationProviderFunc(func(ctx context.Context) string {
			v, _ := ctx.Value(ctxKey{}).(string)
			return v
		}),
	}

	args := []any{"tenant", 5}
	obs := observer.Observe(ctx, OperationSelect, "posts", "SELECT * FROM posts WHERE slug = $1", args,
		WithObservationAttributes(tracing.String("orm.target", "replica")))
	if obs.Context().Value(ctxKey{}) != "span-started" {
		t.Fatalf("expected tracer-provided context value, got %v", obs.Context().Value(ctxKey{}))
	}

	args[0] = "mutated"
	err := errors.New("boom")
	obs.End(err)

	if tracer.name != "orm.posts.select" {
		t.Fatalf("unexpected span name: %s", tracer.name)
	}
	if tracer.ended != err {
		t.Fatalf("span end error mismatch: got %v want %v", tracer.ended, err)
	}
	if !collector.recorded {
		t.Fatalf("expected metrics collector to record query")
	}
	if collector.table != "posts" || collector.operation != "select" {
		t.Fatalf("unexpected collector labels: %+v", collector)
	}
	if collector.err != err {
		t.Fatalf("collector error mismatch: got %v want %v", collector.err, err)
	}
	if collector.duration <= 0 {
		t.Fatalf("collector duration must be positive, got %v", collector.duration)
	}

	if len(logger.entries) != 1 {
		t.Fatalf("expected single log entry, got %d", len(logger.entries))
	}
	entry := logger.entries[0]
	if entry.Operation != OperationSelect || entry.Table != "posts" {
		t.Fatalf("unexpected log metadata: %+v", entry)
	}
	if entry.CorrelationID != "request-42" {
		t.Fatalf("unexpected correlation id: %q", entry.CorrelationID)
	}
	if entry.Err != err {
		t.Fatalf("log error mismatch: got %v want %v", entry.Err, err)
	}
	if len(entry.Args) != 2 || entry.Args[0] != "tenant" {
		t.Fatalf("expected immutable args copy, got %+v", entry.Args)
	}
	if entry.Duration <= 0 {
		t.Fatalf("log duration must be positive, got %v", entry.Duration)
	}
	if len(entry.Attributes) == 0 {
		t.Fatalf("expected attributes to be forwarded")
	}
}

func TestQueryObserverZeroValue(t *testing.T) {
	ctx := context.Background()
	obs := (QueryObserver{}).Observe(ctx, OperationAggregate, "users", "SELECT COUNT(*) FROM users", nil)
	if obs.Context() == nil {
		t.Fatalf("expected non-nil context")
	}
	obs.End(nil)
}

type ctxKey struct{}

type recordingLogger struct {
	entries []QueryLog
}

func (l *recordingLogger) LogQuery(_ context.Context, entry QueryLog) {
	l.entries = append(l.entries, entry)
}

type recordingCollector struct {
	recorded  bool
	table     string
	operation string
	duration  time.Duration
	err       error
}

func (c *recordingCollector) RecordDataloaderBatch(string, int, time.Duration) {}

func (c *recordingCollector) RecordQuery(table string, operation string, duration time.Duration, err error) {
	c.recorded = true
	c.table = table
	c.operation = operation
	c.duration = duration
	c.err = err
}

type recordingTracer struct {
	name  string
	attrs []tracing.Attribute
	ended error
}

func (r *recordingTracer) Start(ctx context.Context, name string, attrs ...tracing.Attribute) (context.Context, tracing.Span) {
	r.name = name
	r.attrs = attrs
	ctx = context.WithValue(ctx, ctxKey{}, "span-started")
	return ctx, recordingSpan{tracer: r}
}

type recordingSpan struct {
	tracer *recordingTracer
}

func (s recordingSpan) End(err error) {
	if s.tracer != nil {
		s.tracer.ended = err
	}
}
