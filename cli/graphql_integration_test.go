package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	testkit "github.com/deicod/erm/testing"
)

func TestGraphQLInitScaffoldsRuntimePackages(t *testing.T) {
	tmpDir := t.TempDir()
	modulePath := "github.com/example/app"

	goMod := "module " + modulePath + "\n\ngo 1.21\n\nrequire (\n\tgithub.com/99designs/gqlgen v0.17.80\n\tgithub.com/vektah/gqlparser/v2 v2.5.30\n)\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cmd := newGraphQLInitCmd()
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("execute graphql init: %v", err)
	}

	expectedFiles := []string{
		"graphql/gqlgen.yml",
		"graphql/schema.graphqls",
		"graphql/dataloaders/loader.go",
		"graphql/directives/auth.go",
		"graphql/relay/id.go",
		"graphql/scalars.go",
		"graphql/server/schema.go",
		"graphql/server/server.go",
		"graphql/subscriptions/bus.go",
	}

	for _, path := range expectedFiles {
		if _, err := os.Stat(filepath.Join(tmpDir, path)); err != nil {
			t.Fatalf("expected runtime file %s: %v", path, err)
		}
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "graphql", "directives", "auth.go"))
	if err != nil {
		t.Fatalf("read directives/auth.go: %v", err)
	}
	if !strings.Contains(string(content), modulePath+"/oidc") {
		t.Fatalf("module path not substituted in directives/auth.go: %s", content)
	}

	testkit.ScaffoldGraphQLRuntime(t, tmpDir, modulePath)

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, output)
	}

	buildCmd := exec.Command("go", "test", "./graphql/dataloaders", "./graphql/directives", "./graphql/relay", "./graphql/server", "./graphql/subscriptions")
	buildCmd.Dir = tmpDir
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compile runtime packages: %v\n%s", err, output)
	}
}
