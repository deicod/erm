package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorCmdSuccess(t *testing.T) {
	tmp := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	writeFile(t, "go.mod", `module example.com/app

go 1.21

require (
	github.com/99designs/gqlgen v0.17.80
	github.com/jackc/pgx/v5 v5.7.6
)
`)
	writeFile(t, "erm.yaml", "module: example.com/app\n")

	if err := os.MkdirAll("schema", 0o755); err != nil {
		t.Fatalf("mkdir schema: %v", err)
	}
	writeFile(t, filepath.Join("schema", "User.schema.go"), "package schema\n")

	graphqlDir := filepath.Join("internal", "graphql")
	if err := os.MkdirAll(graphqlDir, 0o755); err != nil {
		t.Fatalf("mkdir graphql: %v", err)
	}
	writeFile(t, filepath.Join(graphqlDir, "gqlgen.yml"), "schema: []\n")
	writeFile(t, filepath.Join(graphqlDir, "schema.graphqls"), "schema { query: Query }\n")

	cmd := newDoctorCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("doctor cmd: %v\noutput: %s", err, buf.String())
	}

	out := buf.String()
	if !containsAll(out, "Go toolchain", "schema directory", "GraphQL assets") {
		t.Fatalf("unexpected doctor output:\n%s", out)
	}
}

func TestDoctorCmdMissingGoModFails(t *testing.T) {
	tmp := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cmd := newDoctorCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatalf("expected error when go.mod missing")
	}
	if out := buf.String(); !containsAll(out, "go.mod", "missing") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func containsAll(haystack string, needles ...string) bool {
	for _, n := range needles {
		if !strings.Contains(haystack, n) {
			return false
		}
	}
	return true
}
