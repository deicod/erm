package pg

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/deicod/erm/internal/observability/tracing"
	"github.com/deicod/erm/internal/orm/runtime"
)

func TestNewPoolConfigDefaults(t *testing.T) {
	cfg, err := newPoolConfig("postgres://postgres@localhost:5432/app?sslmode=disable")
	if err != nil {
		t.Fatalf("newPoolConfig: %v", err)
	}
	if cfg.MaxConns != 10 {
		t.Fatalf("expected default max conns 10, got %d", cfg.MaxConns)
	}
	if cfg.MinConns != 2 {
		t.Fatalf("expected default min conns 2, got %d", cfg.MinConns)
	}
	if cfg.MaxConnLifetime != time.Hour {
		t.Fatalf("expected default max lifetime %s, got %s", time.Hour, cfg.MaxConnLifetime)
	}
}

func TestOptionsOverrideConfig(t *testing.T) {
	tTracer := &instrumentTracer{}
	cfg, err := newPoolConfig("postgres://postgres@localhost:5432/app?sslmode=disable",
		WithMaxConns(42),
		WithMinConns(5),
		WithMaxConnLifetime(2*time.Hour),
		WithMaxConnIdleTime(15*time.Minute),
		WithHealthCheckPeriod(time.Minute),
		WithPoolConfig(PoolConfig{MaxConns: 80, MinConns: 40, MaxConnLifetime: 5 * time.Hour, MaxConnIdleTime: time.Hour, HealthCheckPeriod: 2 * time.Minute}),
		WithTracer(tTracer),
	)
	if err != nil {
		t.Fatalf("newPoolConfig: %v", err)
	}
	if cfg.MaxConns != 80 {
		t.Fatalf("expected max conns override, got %d", cfg.MaxConns)
	}
	if cfg.MinConns != 40 {
		t.Fatalf("expected min conns override, got %d", cfg.MinConns)
	}
	if cfg.MaxConnLifetime != 5*time.Hour {
		t.Fatalf("expected max lifetime override, got %s", cfg.MaxConnLifetime)
	}
	if cfg.MaxConnIdleTime != time.Hour {
		t.Fatalf("expected max idle override, got %s", cfg.MaxConnIdleTime)
	}
	if cfg.HealthCheckPeriod != 2*time.Minute {
		t.Fatalf("expected health check override, got %s", cfg.HealthCheckPeriod)
	}
	if cfg.ConnConfig.Tracer == nil {
		t.Fatalf("expected tracer to be set")
	}
	if _, ok := cfg.ConnConfig.Tracer.(*pgxTracer); !ok {
		t.Fatalf("expected tracer adapter, got %T", cfg.ConnConfig.Tracer)
	}
}

func TestQueryRoutesToReplicaAndFallsBack(t *testing.T) {
	primaryRows := &stubRows{name: "primary"}
	primary := &fakePool{
		queryFunc: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return primaryRows, nil
		},
	}
	replica := &fakePool{
		queryFunc: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return nil, errors.New("replica offline")
		},
	}

	db := &DB{
		Pool:     primary,
		writer:   primary,
		replicas: []*replicaPool{{name: "replica-a", pool: replica, config: ReplicaConfig{ReadOnly: true}}},
		healthCheck: func(context.Context, Pool) (ReplicaHealthReport, error) {
			return ReplicaHealthReport{ReadOnly: true}, nil
		},
		healthProbeSQL: "SELECT 1",
		healthInterval: time.Hour,
	}

	rows, err := db.Query(WithReplicaRead(context.Background(), ReplicaReadOptions{}), "users", "SELECT 1", 1)
	if err != nil {
		t.Fatalf("query returned error: %v", err)
	}
	if primary.queryCount != 1 {
		t.Fatalf("expected primary to handle fallback, got %d", primary.queryCount)
	}
	if replica.queryCount != 1 {
		t.Fatalf("expected replica to be attempted once, got %d", replica.queryCount)
	}
	fallbackRows, ok := rows.(*stubRows)
	if !ok {
		t.Fatalf("expected stub rows, got %T", rows)
	}
	if fallbackRows.name != "primary" {
		t.Fatalf("expected rows from primary fallback, got %s", fallbackRows.name)
	}
}

func TestAggregateReplicaFallback(t *testing.T) {
	var primaryScans int
	primary := &fakePool{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return stubRow{scanFn: func(dest ...any) error {
				primaryScans++
				if len(dest) > 0 {
					if v, ok := dest[0].(*int); ok {
						*v = 42
					}
				}
				return nil
			}}
		},
	}
	replica := &fakePool{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return stubRow{err: errors.New("replica failure")}
		},
	}

	db := &DB{
		Pool:     primary,
		writer:   primary,
		replicas: []*replicaPool{{name: "replica-a", pool: replica, config: ReplicaConfig{ReadOnly: true}}},
		healthCheck: func(context.Context, Pool) (ReplicaHealthReport, error) {
			return ReplicaHealthReport{ReadOnly: true}, nil
		},
		healthProbeSQL: "SELECT 1",
		healthInterval: time.Hour,
	}

	row := db.Aggregate(WithReplicaRead(context.Background(), ReplicaReadOptions{}), runtime.AggregateSpec{Table: "users"})
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("aggregate scan error: %v", err)
	}
	if count != 42 {
		t.Fatalf("expected fallback scan to populate value, got %d", count)
	}
	if primary.queryRowCount != 1 {
		t.Fatalf("expected primary query row once, got %d", primary.queryRowCount)
	}
	if replica.queryRowCount != 1 {
		t.Fatalf("expected replica query row once, got %d", replica.queryRowCount)
	}
	if primaryScans != 1 {
		t.Fatalf("expected primary scan once, got %d", primaryScans)
	}
}

func TestReplicaHealthLagAvoidsReplica(t *testing.T) {
	primary := &fakePool{
		queryFunc: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &stubRows{name: "primary"}, nil
		},
	}
	replica := &fakePool{
		queryFunc: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &stubRows{name: "replica"}, nil
		},
	}

	db := &DB{
		Pool:   primary,
		writer: primary,
		replicas: []*replicaPool{{
			name:   "replica-laggy",
			pool:   replica,
			config: ReplicaConfig{ReadOnly: true, MaxFollowerLag: 2 * time.Second},
		}},
		healthCheck: func(context.Context, Pool) (ReplicaHealthReport, error) {
			return ReplicaHealthReport{ReadOnly: true, Lag: 10 * time.Second}, nil
		},
		healthProbeSQL: "SELECT 1",
		healthInterval: time.Hour,
	}

	if _, err := db.Query(WithReplicaRead(context.Background(), ReplicaReadOptions{}), "users", "SELECT 1"); err != nil {
		t.Fatalf("query returned error: %v", err)
	}
	if replica.queryCount != 0 {
		t.Fatalf("expected laggy replica to be skipped, got %d queries", replica.queryCount)
	}
	if primary.queryCount != 1 {
		t.Fatalf("expected primary to serve query, got %d", primary.queryCount)
	}
}

func TestReplicaPolicyRouting(t *testing.T) {
	replicaRows := &stubRows{name: "replica"}
	primary := &fakePool{
		queryFunc: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return &stubRows{name: "primary"}, nil
		},
	}
	replica := &fakePool{
		queryFunc: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return replicaRows, nil
		},
	}

	db := &DB{
		Pool:     primary,
		writer:   primary,
		replicas: []*replicaPool{{name: "reporting", pool: replica, config: ReplicaConfig{ReadOnly: true}}},
		healthCheck: func(context.Context, Pool) (ReplicaHealthReport, error) {
			return ReplicaHealthReport{ReadOnly: true}, nil
		},
		healthProbeSQL: "SELECT 1",
		healthInterval: time.Hour,
	}
	db.UseReplicaPolicies("reporting", map[string]ReplicaReadOptions{"reporting": {}})

	rows, err := db.Query(context.Background(), "users", "SELECT 1")
	if err != nil {
		t.Fatalf("query returned error: %v", err)
	}
	if replica.queryCount != 1 {
		t.Fatalf("expected replica to serve query, got %d", replica.queryCount)
	}
	if primary.queryCount != 0 {
		t.Fatalf("expected primary unused, got %d", primary.queryCount)
	}
	routedRows, ok := rows.(*stubRows)
	if !ok {
		t.Fatalf("expected stub rows, got %T", rows)
	}
	if routedRows.name != "replica" {
		t.Fatalf("expected replica rows, got %s", routedRows.name)
	}

	if _, err := db.Query(WithPrimary(context.Background()), "users", "SELECT 1"); err != nil {
		t.Fatalf("query with primary override error: %v", err)
	}
	if primary.queryCount != 1 {
		t.Fatalf("expected primary override to query primary, got %d", primary.queryCount)
	}
}

type instrumentTracer struct{}

func (instrumentTracer) Start(ctx context.Context, name string, attrs ...tracing.Attribute) (context.Context, tracing.Span) {
	return ctx, instrumentSpan{}
}

type instrumentSpan struct{}

func (instrumentSpan) End(err error) {}

// ensure Options type signature compiles with custom functions
var _ Option = func(cfg *pgxpool.Config) {}

type fakePool struct {
	queryFunc    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	queryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
	execFunc     func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)

	queryCount    int
	queryRowCount int
	execCount     int
	closed        bool
}

func (p *fakePool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	p.queryCount++
	if p.queryFunc != nil {
		return p.queryFunc(ctx, sql, args...)
	}
	return &stubRows{}, nil
}

func (p *fakePool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	p.queryRowCount++
	if p.queryRowFunc != nil {
		return p.queryRowFunc(ctx, sql, args...)
	}
	return stubRow{}
}

func (p *fakePool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	p.execCount++
	if p.execFunc != nil {
		return p.execFunc(ctx, sql, args...)
	}
	return pgconn.CommandTag{}, nil
}

func (p *fakePool) Close() { p.closed = true }

type stubRows struct {
	name string
	err  error
}

func (s *stubRows) Close()                                       {}
func (s *stubRows) Err() error                                   { return s.err }
func (s *stubRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (s *stubRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (s *stubRows) Next() bool                                   { return false }
func (s *stubRows) RawValues() [][]byte                          { return nil }
func (s *stubRows) Values() ([]any, error)                       { return nil, s.err }
func (s *stubRows) Scan(dest ...any) error                       { return s.err }
func (s *stubRows) Conn() *pgx.Conn                              { return nil }

type stubRow struct {
	err    error
	scanFn func(dest ...any) error
}

func (r stubRow) Scan(dest ...any) error {
	if r.scanFn != nil {
		return r.scanFn(dest...)
	}
	return r.err
}
