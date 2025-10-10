package pg

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/deicod/erm/internal/observability/tracing"
	"github.com/deicod/erm/internal/orm/runtime"
)

// Pool exposes the subset of pgxpool behaviour required by generated clients.
type Pool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Close()
}

type DB struct {
	Pool     Pool
	Observer runtime.QueryObserver
}

// Option configures pgx connections.
type Option func(*pgxpool.Config)

// Connect initialises a pgx pool with optional configuration overrides.
func Connect(ctx context.Context, url string, opts ...Option) (*DB, error) {
	cfg, err := newPoolConfig(url, opts...)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

func (db *DB) Select(ctx context.Context, spec runtime.SelectSpec) (pgx.Rows, error) {
	sql, args := runtime.BuildSelectSQL(spec)
	obs := db.Observer.Observe(ctx, runtime.OperationSelect, spec.Table, sql, args)
	rows, err := db.Pool.Query(obs.Context(), sql, args...)
	obs.End(err)
	return rows, err
}

func (db *DB) Aggregate(ctx context.Context, spec runtime.AggregateSpec) pgx.Row {
	sql, args := runtime.BuildAggregateSQL(spec)
	obs := db.Observer.Observe(ctx, runtime.OperationAggregate, spec.Table, sql, args)
	row := db.Pool.QueryRow(obs.Context(), sql, args...)
	obs.End(nil)
	return row
}

// UseObserver attaches a query observer to the database handle.
func (db *DB) UseObserver(observer runtime.QueryObserver) {
	if db == nil {
		return
	}
	db.Observer = observer
}

func newPoolConfig(url string, opts ...Option) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	applyDefaults(cfg)
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(cfg)
	}
	return cfg, nil
}

func applyDefaults(cfg *pgxpool.Config) {
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
}

// WithMaxConns sets the maximum pool size.
func WithMaxConns(n int32) Option {
	return func(cfg *pgxpool.Config) {
		cfg.MaxConns = n
	}
}

// WithMinConns sets the minimum pool size.
func WithMinConns(n int32) Option {
	return func(cfg *pgxpool.Config) {
		cfg.MinConns = n
	}
}

// WithMaxConnLifetime configures the maximum connection lifetime.
func WithMaxConnLifetime(d time.Duration) Option {
	return func(cfg *pgxpool.Config) { cfg.MaxConnLifetime = d }
}

// WithMaxConnIdleTime configures how long an idle connection may remain in the pool.
func WithMaxConnIdleTime(d time.Duration) Option {
	return func(cfg *pgxpool.Config) { cfg.MaxConnIdleTime = d }
}

// WithHealthCheckPeriod configures the background health check period.
func WithHealthCheckPeriod(d time.Duration) Option {
	return func(cfg *pgxpool.Config) { cfg.HealthCheckPeriod = d }
}

// WithTracer enables pgx tracing using the provided tracer abstraction.
func WithTracer(tracer tracing.Tracer) Option {
	return func(cfg *pgxpool.Config) {
		if tracer == nil {
			cfg.ConnConfig.Tracer = nil
			return
		}
		cfg.ConnConfig.Tracer = newPGXTracer(tracer)
	}
}
