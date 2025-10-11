package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInitCmdScaffoldsWorkspace(t *testing.T) {
	tmp := t.TempDir()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(wd); chdirErr != nil {
			t.Fatalf("chdir back: %v", chdirErr)
		}
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cmd := newInitCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("run init: %v", err)
	}

	assertFileContains(t, "erm.yaml", "module: \"\"")
	assertFileContains(t, "README.md", "This workspace was bootstrapped with 'erm init'")
	assertFileContains(t, "AGENTS.md", "Development Workflow")
	assertFileContains(t, filepath.Join("cmd", "api", "main.go"), "package main")
	assertFileContains(t, filepath.Join("cmd", "api", "main.go"), "TODO: mount your GraphQL handler")
	assertFileContains(t, filepath.Join("schema", "AGENTS.md"), "Schema Development Workflow")
	assertFileContains(t, filepath.Join("graphql", "README.md"), "# GraphQL workspace")

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("second init should be idempotent: %v", err)
	}

	if got := buf.String(); got == "" {
		t.Fatalf("expected init to print confirmation message")
	}
}

func assertFileContains(t *testing.T, path, substr string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !bytes.Contains(data, []byte(substr)) {
		t.Fatalf("file %s missing %q\ncontent:\n%s", path, substr, string(data))
	}
}
