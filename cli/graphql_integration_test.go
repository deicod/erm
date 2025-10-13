package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deicod/erm/generator"
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
		"graphql/types.go",
		"graphql/server/schema.go",
		"graphql/server/server.go",
		"graphql/subscriptions/bus.go",
		"observability/metrics/metrics.go",
		"oidc/claims.go",
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

	if err := os.MkdirAll(filepath.Join(tmpDir, "schema"), 0o755); err != nil {
		t.Fatalf("mkdir schema: %v", err)
	}

	newCmd := newNewCmd()
	if err := newCmd.RunE(newCmd, []string{"User"}); err != nil {
		t.Fatalf("execute new: %v", err)
	}

	testkit.ScaffoldGraphQLRuntime(t, tmpDir, modulePath)

	preGenTidy := exec.Command("go", "mod", "tidy")
	preGenTidy.Dir = tmpDir
	if output, err := preGenTidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy before generation: %v\n%s", err, output)
	}

	genCmd := newGenCmd()
	if err := genCmd.RunE(genCmd, []string{}); err != nil {
		t.Fatalf("execute gen: %v", err)
	}

	gqlgenOutput, err := os.ReadFile(filepath.Join(tmpDir, "graphql", "generated.go"))
	if err != nil {
		t.Fatalf("read graphql/generated.go: %v", err)
	}
	if !strings.Contains(string(gqlgenOutput), "return &executableSchema") {
		t.Fatalf("expected gqlgen output to include executable schema constructor; got:\n%s", gqlgenOutput)
	}

	rerun, err := generator.Run(tmpDir, generator.GenerateOptions{})
	if err != nil {
		t.Fatalf("second generation: %v", err)
	}
	for _, comp := range rerun.Components {
		if comp.Name != generator.ComponentGraphQL && comp.Name != generator.ComponentORM {
			continue
		}
		if comp.Changed {
			t.Fatalf("expected %s to be up-to-date on second generation; reason=%s files=%v", comp.Name, comp.Reason, comp.Files)
		}
	}

	testkit.RemoveGraphQLRuntimeStub(t, tmpDir)

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy after generation: %v\n%s", err, output)
	}

	buildCmd := exec.Command("go", "test", "./graphql/...", "./orm/gen")
	buildCmd.Dir = tmpDir
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compile runtime packages: %v\n%s", err, output)
	}
}
