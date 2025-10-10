package cli

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deicod/erm/internal/generator"
	"github.com/deicod/erm/internal/orm/migrate"
	"github.com/jackc/pgx/v5"
)

type stubConn struct {
	closed bool
}

func (s *stubConn) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return nil, errors.New("not implemented")
}

func (s *stubConn) Close(ctx context.Context) error {
	s.closed = true
	return nil
}

func TestGenCmdForwardsOptions(t *testing.T) {
	original := runGenerator
	defer func() { runGenerator = original }()

	var capturedOpts generator.GenerateOptions
	runGenerator = func(root string, opts generator.GenerateOptions) (generator.MigrationResult, error) {
		capturedOpts = opts
		return generator.MigrationResult{}, nil
	}

	cmd := newGenCmd()
	if err := cmd.Flags().Set("dry-run", "true"); err != nil {
		t.Fatalf("set dry-run: %v", err)
	}
	if err := cmd.Flags().Set("force", "true"); err != nil {
		t.Fatalf("set force: %v", err)
	}
	if err := cmd.Flags().Set("name", "custom_slug"); err != nil {
		t.Fatalf("set name: %v", err)
	}
	if err := cmd.Flags().Set("only", "orm,graphql"); err != nil {
		t.Fatalf("set only: %v", err)
	}

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("run gen: %v", err)
	}

	if !capturedOpts.DryRun {
		t.Fatalf("expected DryRun to be true")
	}
	if !capturedOpts.Force {
		t.Fatalf("expected Force to be true")
	}
	if capturedOpts.MigrationName != "custom_slug" {
		t.Fatalf("expected MigrationName to be custom_slug, got %q", capturedOpts.MigrationName)
	}
	want := []string{"graphql", "orm"}
	if len(capturedOpts.Components) != len(want) {
		t.Fatalf("expected %d components, got %d", len(want), len(capturedOpts.Components))
	}
	for i, comp := range want {
		if capturedOpts.Components[i] != comp {
			t.Fatalf("component[%d] = %q, want %q", i, capturedOpts.Components[i], comp)
		}
	}
}

func TestGenCmdDryRunPrintsSQL(t *testing.T) {
	original := runGenerator
	defer func() { runGenerator = original }()

	runGenerator = func(root string, opts generator.GenerateOptions) (generator.MigrationResult, error) {
		return generator.MigrationResult{
			Operations: []generator.Operation{{Kind: generator.OpCreateTable, SQL: "CREATE TABLE users (...)"}},
			SQL:        "-- migration preview\nCREATE TABLE users (...);",
		}, nil
	}

	cmd := newGenCmd()
	if err := cmd.Flags().Set("dry-run", "true"); err != nil {
		t.Fatalf("set dry-run: %v", err)
	}

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("run gen: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "generator: dry-run - no files were written") {
		t.Fatalf("expected dry-run message, got:\n%s", output)
	}
	if !strings.Contains(output, "generator: application artifacts would be refreshed") {
		t.Fatalf("expected application artifact message, got:\n%s", output)
	}
	if !strings.Contains(output, "generator: migration dry-run preview") {
		t.Fatalf("expected preview header, got:\n%s", output)
	}
	if !strings.Contains(output, "CREATE TABLE users (...);") {
		t.Fatalf("expected SQL preview, got:\n%s", output)
	}
}

func TestGenCmdDryRunDiffSummary(t *testing.T) {
	original := runGenerator
	defer func() { runGenerator = original }()

	runGenerator = func(root string, opts generator.GenerateOptions) (generator.MigrationResult, error) {
		return generator.MigrationResult{
			Operations: []generator.Operation{{Kind: generator.OpCreateTable, Target: "users", SQL: "CREATE TABLE users (...);"}},
			SQL:        "-- preview\nCREATE TABLE users (...);",
		}, nil
	}

	cmd := newGenCmd()
	if err := cmd.Flags().Set("dry-run", "true"); err != nil {
		t.Fatalf("set dry-run: %v", err)
	}
	if err := cmd.Flags().Set("diff", "true"); err != nil {
		t.Fatalf("set diff: %v", err)
	}

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("run gen: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "generator: diff summary") {
		t.Fatalf("expected diff summary header, got:\n%s", output)
	}
	if !strings.Contains(output, "+ create_table users") {
		t.Fatalf("expected diff entry, got:\n%s", output)
	}
}

func TestGenCmdRejectsUnknownComponent(t *testing.T) {
	cmd := newGenCmd()
	if err := cmd.Flags().Set("only", "foo"); err != nil {
		t.Fatalf("set only: %v", err)
	}

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatalf("expected error for unknown component")
	}
	var cerr CommandError
	if !errors.As(err, &cerr) {
		t.Fatalf("expected CommandError, got %T", err)
	}
	if !strings.Contains(cerr.Message, "unknown component") {
		t.Fatalf("expected unknown component message, got %q", cerr.Message)
	}
	if cerr.Suggestion == "" {
		t.Fatalf("expected suggestion to be populated")
	}
	if cerr.ExitStatus() != 2 {
		t.Fatalf("expected exit code 2, got %d", cerr.ExitStatus())
	}
}

func TestMigrateCmdAppliesMigrations(t *testing.T) {
	originalOpen := openMigrationConn
	originalApply := applyMigrations
	defer func() {
		openMigrationConn = originalOpen
		applyMigrations = originalApply
	}()

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "erm.yaml"), []byte("module: test\ndatabase:\n  url: postgres://localhost/db"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var capturedDSN string
	conn := &stubConn{}
	openMigrationConn = func(ctx context.Context, url string) (migrationConn, error) {
		capturedDSN = url
		return conn, nil
	}

	var applyCalled bool
	applyMigrations = func(ctx context.Context, c migrate.TxStarter, fsys fs.FS, opts ...migrate.Option) error {
		applyCalled = true
		if c != conn {
			t.Fatalf("expected connection stub, got %T", c)
		}
		return nil
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cmd := newMigrateCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("run migrate: %v", err)
	}

	if capturedDSN != "postgres://localhost/db" {
		t.Fatalf("expected DSN to be captured, got %q", capturedDSN)
	}
	if !applyCalled {
		t.Fatalf("expected applyMigrations to be invoked")
	}
	if !conn.closed {
		t.Fatalf("expected connection to be closed")
	}

	output := buf.String()
	if !strings.Contains(output, "migrate: applying migrations") {
		t.Fatalf("expected applying message, got:\n%s", output)
	}
	if !strings.Contains(output, "migrate: completed successfully") {
		t.Fatalf("expected success message, got:\n%s", output)
	}
}

func TestMigrateCmdRequiresDSN(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "erm.yaml"), []byte("module: test"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cmd := newMigrateCmd()
	err = cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatalf("expected error when DSN missing")
	}
	var cerr CommandError
	if !errors.As(err, &cerr) {
		t.Fatalf("expected CommandError, got %T", err)
	}
	if !strings.Contains(cerr.Message, "database.url") {
		t.Fatalf("expected error to mention database.url, got %q", cerr.Message)
	}
	if cerr.Suggestion == "" {
		t.Fatalf("expected suggestion to be present")
	}
	if cerr.ExitStatus() != 2 {
		t.Fatalf("expected exit status 2, got %d", cerr.ExitStatus())
	}
}

func TestRootRegistersGenAndMigrate(t *testing.T) {
	cmd := NewRootCmd()
	names := make(map[string]struct{})
	for _, child := range cmd.Commands() {
		names[child.Name()] = struct{}{}
	}
	for _, name := range []string{"gen", "migrate"} {
		if _, ok := names[name]; !ok {
			t.Fatalf("expected %s command to be registered", name)
		}
	}
	if flag := cmd.PersistentFlags().Lookup("verbose"); flag == nil {
		t.Fatalf("expected verbose flag to be registered")
	}
}
