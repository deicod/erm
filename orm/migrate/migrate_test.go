package migrate

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestParseVersion(t *testing.T) {
	cases := map[string]string{
		"0001_init.sql":        "0001",
		"20240101__create.sql": "20240101",
		"feature-a.sql":        "feature",
		"plain.sql":            "plain",
	}
	for name, want := range cases {
		got, err := ParseVersion(name)
		if err != nil {
			t.Fatalf("ParseVersion(%q) unexpected error: %v", name, err)
		}
		if got != want {
			t.Fatalf("ParseVersion(%q) = %q, want %q", name, got, want)
		}
	}

	if _, err := ParseVersion(""); err == nil {
		t.Fatal("ParseVersion with empty string should error")
	}
}

func TestDiscoverOrdersMigrations(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/002_second.sql": &fstest.MapFile{Mode: 0o644, Data: []byte("-- noop")},
		"migrations/001_first.sql":  &fstest.MapFile{Mode: 0o644, Data: []byte("-- noop")},
		"migrations/readme.txt":     &fstest.MapFile{Mode: 0o644, Data: []byte("ignore")},
	}

	migs, err := Discover(context.Background(), fsys, "migrations")
	if err != nil {
		t.Fatalf("Discover error: %v", err)
	}
	if len(migs) != 2 {
		t.Fatalf("Discover returned %d migrations, want 2", len(migs))
	}
	if migs[0].Name != "001_first.sql" || migs[1].Name != "002_second.sql" {
		t.Fatalf("Discover order wrong: %+v", migs)
	}

	fsys["migrations/003_dup.sql"] = &fstest.MapFile{Mode: 0o644, Data: []byte("-- noop")}
	fsys["migrations/003_dup_again.sql"] = &fstest.MapFile{Mode: 0o644, Data: []byte("-- noop")}

	if _, err := Discover(context.Background(), fsys, "migrations"); err == nil {
		t.Fatal("expected duplicate version error")
	}
}

func TestPlanIdentifiesPendingMigrations(t *testing.T) {
	mock, err := pgxmock.NewConn(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("pgxmock.NewConn: %v", err)
	}
	defer mock.Close(context.Background())

	fsys := fstest.MapFS{
		"migrations/001_first.sql": &fstest.MapFile{Mode: 0o644, Data: []byte("create table first;")},
	}

	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectExec("SELECT pg_advisory_xact_lock($1)").WithArgs(defaultAdvisoryLock).WillReturnResult(pgxmock.NewResult("SELECT", 1))
	pgErr := &pgconn.PgError{Code: "42P01"}
	mock.ExpectQuery("SELECT version FROM erm_schema_migrations ORDER BY applied_at").WillReturnError(pgErr)

	plan, err := Plan(context.Background(), mock, fsys)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Pending) != 1 || plan.Pending[0].Version != "001" {
		t.Fatalf("unexpected plan: %+v", plan)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestPlanDetectsSchemaDrift(t *testing.T) {
	mock, err := pgxmock.NewConn(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("pgxmock.NewConn: %v", err)
	}
	defer mock.Close(context.Background())

	fsys := fstest.MapFS{
		"migrations/001_first.sql": &fstest.MapFile{Mode: 0o644, Data: []byte("-- noop")},
	}

	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectExec("SELECT pg_advisory_xact_lock($1)").WithArgs(defaultAdvisoryLock).WillReturnResult(pgxmock.NewResult("SELECT", 1))
	rows := mock.NewRows([]string{"version"}).AddRow("999")
	mock.ExpectQuery("SELECT version FROM erm_schema_migrations ORDER BY applied_at").WillReturnRows(rows)

	_, err = Plan(context.Background(), mock, fsys)
	if err == nil {
		t.Fatal("expected schema drift error")
	}
	var driftErr SchemaDriftError
	if !errors.As(err, &driftErr) {
		t.Fatalf("expected SchemaDriftError, got %T", err)
	}
	if len(driftErr.Missing) != 1 || driftErr.Missing[0] != "999" {
		t.Fatalf("unexpected drift details: %+v", driftErr.Missing)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestRollbackExecutesDownMigration(t *testing.T) {
	mock, err := pgxmock.NewConn(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("pgxmock.NewConn: %v", err)
	}
	defer mock.Close(context.Background())

	fsys := fstest.MapFS{
		"migrations/001_first.sql":      &fstest.MapFile{Mode: 0o644, Data: []byte("create table first;")},
		"migrations/001_first_down.sql": &fstest.MapFile{Mode: 0o644, Data: []byte("drop table first;")},
	}

	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectExec("SELECT pg_advisory_xact_lock($1)").WithArgs(defaultAdvisoryLock).WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS erm_schema_migrations (\n        version    text PRIMARY KEY,\n        applied_at timestamptz NOT NULL DEFAULT now()\n    )").WillReturnResult(pgxmock.NewResult("CREATE", 0))
	rows := mock.NewRows([]string{"version"}).AddRow("001")
	mock.ExpectQuery("SELECT version FROM erm_schema_migrations ORDER BY applied_at DESC LIMIT 1").WillReturnRows(rows)
	mock.ExpectExec("drop table first;").WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectExec("DELETE FROM erm_schema_migrations WHERE version = $1").WithArgs("001").WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectCommit()

	rolled, err := Rollback(context.Background(), mock, fsys)
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if rolled.Type != MigrationTypeDown || rolled.Version != "001" {
		t.Fatalf("unexpected rolled back migration: %+v", rolled)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestApplyExecutesPendingMigrations(t *testing.T) {
	mock, err := pgxmock.NewConn(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("pgxmock.NewConn: %v", err)
	}
	defer mock.Close(context.Background())

	fsys := fstest.MapFS{
		"migrations/001_first.sql":  &fstest.MapFile{Mode: 0o644, Data: []byte("create table first;")},
		"migrations/010_second.sql": &fstest.MapFile{Mode: 0o644, Data: []byte("create table second;")},
	}

	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectExec("SELECT pg_advisory_xact_lock($1)").WithArgs(defaultAdvisoryLock).WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS erm_schema_migrations (\n        version    text PRIMARY KEY,\n        applied_at timestamptz NOT NULL DEFAULT now()\n    )").WillReturnResult(pgxmock.NewResult("CREATE", 0))
	mock.ExpectQuery("SELECT version FROM erm_schema_migrations").WillReturnRows(mock.NewRows([]string{"version"}))
	mock.ExpectExec("create table first;").WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectExec("INSERT INTO erm_schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING").WithArgs("001").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec("create table second;").WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectExec("INSERT INTO erm_schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING").WithArgs("010").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	if err := Apply(context.Background(), mock, fsys); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestApplyIsIdempotent(t *testing.T) {
	mock, err := pgxmock.NewConn(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("pgxmock.NewConn: %v", err)
	}
	defer mock.Close(context.Background())

	fsys := fstest.MapFS{
		"migrations/001_first.sql": &fstest.MapFile{Mode: 0o644, Data: []byte("create table first;")},
	}

	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectExec("SELECT pg_advisory_xact_lock($1)").WithArgs(defaultAdvisoryLock).WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS erm_schema_migrations (\n        version    text PRIMARY KEY,\n        applied_at timestamptz NOT NULL DEFAULT now()\n    )").WillReturnResult(pgxmock.NewResult("CREATE", 0))
	rows := mock.NewRows([]string{"version"}).AddRow("001")
	mock.ExpectQuery("SELECT version FROM erm_schema_migrations").WillReturnRows(rows)
	mock.ExpectCommit()

	if err := Apply(context.Background(), mock, fsys); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestApplyHonorsBatchSize(t *testing.T) {
	mock, err := pgxmock.NewConn(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("pgxmock.NewConn: %v", err)
	}
	defer mock.Close(context.Background())

	fsys := fstest.MapFS{
		"migrations/001_first.sql":  &fstest.MapFile{Mode: 0o644, Data: []byte("create table first;")},
		"migrations/002_second.sql": &fstest.MapFile{Mode: 0o644, Data: []byte("create table second;")},
	}

	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectExec("SELECT pg_advisory_xact_lock($1)").WithArgs(defaultAdvisoryLock).WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS erm_schema_migrations (\n        version    text PRIMARY KEY,\n        applied_at timestamptz NOT NULL DEFAULT now()\n    )").WillReturnResult(pgxmock.NewResult("CREATE", 0))
	mock.ExpectQuery("SELECT version FROM erm_schema_migrations").WillReturnRows(mock.NewRows([]string{"version"}))
	mock.ExpectExec("create table first;").WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectExec("INSERT INTO erm_schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING").WithArgs("001").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	if err := Apply(context.Background(), mock, fsys, WithBatchSize(1)); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestWrapExecErrorFormatsPosition(t *testing.T) {
	pgErr := &pgconn.PgError{Position: 5}
	err := wrapExecError("migrations/001_first.sql", "line1\nline2", pgErr)
	if !errors.Is(err, pgErr) {
		t.Fatalf("wrapExecError should retain underlying error")
	}
	if got := err.Error(); !strings.Contains(got, "migrations/001_first.sql:1:5") {
		t.Fatalf("unexpected error message: %s", got)
	}
}
