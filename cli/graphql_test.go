package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGraphQLInitCmdGeneratesResolverConfig(t *testing.T) {
	tmpDir := t.TempDir()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module github.com/example/app\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cmd := newGraphQLInitCmd()
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("execute graphql init: %v", err)
	}

	graphqlDir := "graphql"
	if _, err := os.Stat(graphqlDir); err != nil {
		t.Fatalf("graphql directory missing after init: %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(graphqlDir, "gqlgen.yml"))
	if err != nil {
		t.Fatalf("read gqlgen.yml: %v", err)
	}

	cfg := string(contents)
	if !strings.Contains(cfg, "dir: graphql/resolvers") {
		t.Fatalf("resolver dir not set correctly in gqlgen.yml:\n%s", cfg)
	}
	if !strings.Contains(cfg, "package: resolvers") {
		t.Fatalf("resolver package not set correctly in gqlgen.yml:\n%s", cfg)
	}
	if !strings.Contains(cfg, "  BigInt:\n    model:\n      - int64\n") {
		t.Fatalf("gqlgen.yml missing BigInt scalar mapping:\n%s", cfg)
	}
	if !strings.Contains(cfg, "  JSONB:\n    model:\n      - encoding/json.RawMessage\n") {
		t.Fatalf("gqlgen.yml missing JSONB scalar mapping:\n%s", cfg)
	}
	if !strings.Contains(cfg, "  Timestamptz:\n    model:\n      - time.Time\n") {
		t.Fatalf("gqlgen.yml missing Timestamptz scalar mapping:\n%s", cfg)
	}
	if strings.Contains(cfg, "autobind:") {
		t.Fatalf("gqlgen.yml unexpectedly includes autobind block by default:\n%s", cfg)
	}
}

func TestGraphQLInitCmdCanIncludeAutobind(t *testing.T) {
	tmpDir := t.TempDir()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module github.com/example/app\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cmd := newGraphQLInitCmd()
	if err := cmd.Flags().Set("autobind", "true"); err != nil {
		t.Fatalf("set autobind flag: %v", err)
	}
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("execute graphql init with autobind: %v", err)
	}

	contents, err := os.ReadFile(filepath.Join("graphql", "gqlgen.yml"))
	if err != nil {
		t.Fatalf("read gqlgen.yml: %v", err)
	}
	cfg := string(contents)
	if !strings.Contains(cfg, "autobind:\n  - github.com/example/app/orm/gen\n") {
		t.Fatalf("gqlgen.yml missing autobind block when requested:\n%s", cfg)
	}
}
