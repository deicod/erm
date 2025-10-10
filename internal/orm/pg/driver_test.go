package pg

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/deicod/erm/internal/observability/tracing"
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

type instrumentTracer struct{}

func (instrumentTracer) Start(ctx context.Context, name string, attrs ...tracing.Attribute) (context.Context, tracing.Span) {
	return ctx, instrumentSpan{}
}

type instrumentSpan struct{}

func (instrumentSpan) End(err error) {}

// ensure Options type signature compiles with custom functions
var _ Option = func(cfg *pgxpool.Config) {}
