package testkit

import (
	stdtesting "testing"

	"github.com/deicod/erm/orm/gen"
	"github.com/deicod/erm/orm/pg"
)

// NewORMClient constructs a generated ORM client backed by the provided database handle.
func NewORMClient(tb stdtesting.TB, db *pg.DB) *gen.Client {
	tb.Helper()
	if db == nil {
		tb.Fatalf("pg.DB is required")
	}
	return gen.NewClient(db)
}

// ORM lazily constructs (and memoises) the generated ORM client for the sandbox.
func (s *Sandbox) ORM(tb stdtesting.TB) *gen.Client {
	tb.Helper()
	if s == nil {
		tb.Fatalf("sandbox is nil")
	}
	if s.db == nil {
		tb.Fatalf("sandbox database is not initialised")
	}
	if s.orm == nil {
		s.orm = gen.NewClient(s.db)
	}
	return s.orm
}
