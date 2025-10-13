package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureRuntimeScaffoldsIdempotent(t *testing.T) {
	dir := t.TempDir()
	modulePath := "github.com/example/app"

	if err := EnsureRuntimeScaffolds(dir, modulePath); err != nil {
		t.Fatalf("EnsureRuntimeScaffolds: %v", err)
	}

	expected := []string{
		"graphql/dataloaders/loader.go",
		"graphql/dataloaders/entities_gen.go",
		"graphql/directives/auth.go",
		"graphql/generated.go",
		"graphql/relay/id.go",
		"graphql/scalars.go",
		"graphql/types.go",
		"graphql/resolvers/resolver.go",
		"graphql/resolvers/entities_gen.go",
		"graphql/resolvers/entities_hooks.go",
		"graphql/server/schema.go",
		"graphql/server/server.go",
		"graphql/subscriptions/bus.go",
		"observability/metrics/metrics.go",
		"oidc/claims.go",
	}

	for _, rel := range expected {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
		if strings.Contains(rel, "directives/auth.go") && !strings.Contains(string(content), modulePath+"/oidc") {
			t.Fatalf("expected module path substitution in %s", rel)
		}
		if strings.Contains(rel, "dataloaders/loader.go") && !strings.Contains(string(content), modulePath+"/orm/gen") {
			t.Fatalf("expected module path substitution in %s", rel)
		}
		if strings.Contains(rel, "resolvers/resolver.go") && !strings.Contains(string(content), modulePath+"/graphql/subscriptions") {
			t.Fatalf("expected module path substitution in %s", rel)
		}
	}

	directivesPath := filepath.Join(dir, "graphql", "directives", "auth.go")
	if err := os.WriteFile(directivesPath, []byte("package directives\n"), 0o644); err != nil {
		t.Fatalf("override auth.go: %v", err)
	}

	if err := EnsureRuntimeScaffolds(dir, modulePath); err != nil {
		t.Fatalf("second EnsureRuntimeScaffolds: %v", err)
	}

	content, err := os.ReadFile(directivesPath)
	if err != nil {
		t.Fatalf("read directives after second ensure: %v", err)
	}
	if string(content) != "package directives\n" {
		t.Fatalf("expected existing file to remain unchanged, got:\n%s", content)
	}
}
