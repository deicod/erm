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

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cmd := newGraphQLInitCmd()
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("execute graphql init: %v", err)
	}

	graphqlDir := filepath.Join("internal", "graphql")
	if _, err := os.Stat(graphqlDir); err != nil {
		t.Fatalf("graphql directory missing after init: %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(graphqlDir, "gqlgen.yml"))
	if err != nil {
		t.Fatalf("read gqlgen.yml: %v", err)
	}

	cfg := string(contents)
	if !strings.Contains(cfg, "dir: internal/graphql/resolvers") {
		t.Fatalf("resolver dir not set correctly in gqlgen.yml:\n%s", cfg)
	}
	if !strings.Contains(cfg, "package: resolvers") {
		t.Fatalf("resolver package not set correctly in gqlgen.yml:\n%s", cfg)
	}
}
