package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDockerSyncGeneratesComposeAndExtensions(t *testing.T) {
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

	config := `module: github.com/example/app
database:
  url: postgres://dev:secret@localhost:6543/app_dev?sslmode=disable
extensions:
  postgis: true
  pgvector: false
  timescaledb: true
`
	if err := os.WriteFile("erm.yaml", []byte(config), 0o644); err != nil {
		t.Fatalf("write erm.yaml: %v", err)
	}

	cmd := newDockerSyncCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("run docker sync: %v", err)
	}

	compose := filepath.Join("docker", "local", "compose.yaml")
	assertFileContains(t, compose, "timescale/timescaledb:")
	assertFileContains(t, compose, "6543:5432")
	assertFileContains(t, compose, "app-dev-postgres")
	assertFileContains(t, compose, "Enabled extensions: postgis, timescaledb")

	sql := filepath.Join("docker", "local", "initdb.d", "extensions.sql")
	assertFileContains(t, sql, "CREATE EXTENSION IF NOT EXISTS postgis;")
	assertFileContains(t, sql, "CREATE EXTENSION IF NOT EXISTS timescaledb;")
	assertFileContains(t, sql, "Enabled extensions: postgis, timescaledb")
	assertFileContains(t, sql, "Managed by `erm docker sync`")

	if got := buf.String(); !strings.Contains(got, "Synchronized docker/local assets.") {
		t.Fatalf("expected status message, got %q", got)
	}
}
