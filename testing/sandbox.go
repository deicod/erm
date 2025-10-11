package testkit

import (
	"context"
	stdtesting "testing"

	pgxmock "github.com/pashagolub/pgxmock/v4"

	"github.com/deicod/erm/orm/gen"
	"github.com/deicod/erm/orm/pg"
)

type mockPool struct {
	pgxmock.PgxConnIface
}

func (m *mockPool) Close() {
	_ = m.PgxConnIface.Close(context.Background())
}

// Sandbox encapsulates a mocked Postgres connection and cancellable context for tests.
type Sandbox struct {
	ctx    context.Context
	cancel context.CancelFunc
	mock   pgxmock.PgxConnIface
	db     *pg.DB
	orm    *gen.Client
}

// NewPostgresSandbox returns a sandbox backed by pgxmock with QueryMatcherEqual semantics.
//
// Tests can configure expectations directly on the returned sandbox via the Mock method and
// obtain a runtime client through ORM().
func NewPostgresSandbox(tb stdtesting.TB) *Sandbox {
	tb.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	mock, err := pgxmock.NewConn(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		tb.Fatalf("pgxmock.NewConn: %v", err)
	}
	sandbox := &Sandbox{
		ctx:    ctx,
		cancel: cancel,
		mock:   mock,
		db:     &pg.DB{Pool: &mockPool{PgxConnIface: mock}},
	}
	tb.Cleanup(sandbox.Close)
	return sandbox
}

// Context returns the sandbox context for use in ORM and GraphQL helpers.
func (s *Sandbox) Context() context.Context {
	if s == nil {
		return context.Background()
	}
	return s.ctx
}

// Mock exposes the underlying pgxmock connection for expectation management.
func (s *Sandbox) Mock() pgxmock.PgxConnIface {
	if s == nil {
		return nil
	}
	return s.mock
}

// DB returns the pg.DB wrapper bound to the sandbox connection.
func (s *Sandbox) DB() *pg.DB {
	if s == nil {
		return nil
	}
	return s.db
}

// Close releases sandbox resources. Tests typically rely on the registered cleanup to invoke it.
func (s *Sandbox) Close() {
	if s == nil {
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.mock != nil {
		_ = s.mock.Close(context.Background())
	}
}

// ExpectationsWereMet fails the supplied test if outstanding pgxmock expectations remain.
func (s *Sandbox) ExpectationsWereMet(tb stdtesting.TB) {
	if s == nil {
		return
	}
	tb.Helper()
	if err := s.mock.ExpectationsWereMet(); err != nil {
		tb.Fatalf("pgx expectations: %v", err)
	}
}
