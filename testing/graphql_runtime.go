package testkit

import (
	"os"
	"path/filepath"
	"strings"
	stdtesting "testing"

	"github.com/deicod/erm/templates"
)

// ScaffoldGraphQLRuntime ensures GraphQL runtime dependencies exist so generated code can compile.
func ScaffoldGraphQLRuntime(tb stdtesting.TB, root, modulePath string) {
	tb.Helper()

	rendered, err := templates.RenderRuntimeScaffolds(normaliseModulePath(modulePath))
	if err != nil {
		tb.Fatalf("render runtime scaffolds: %v", err)
	}
	for rel, content := range rendered {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err == nil {
			continue
		} else if err != nil && !os.IsNotExist(err) {
			tb.Fatalf("stat %s: %v", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			tb.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			tb.Fatalf("write scaffold %s: %v", path, err)
		}
	}

	stubs := map[string]string{
		filepath.Join(root, "orm", "gen", "client.go"): `package gen

type Client struct{}

func NewClient(any) *Client { return &Client{} }
`,
	}

	for path, content := range stubs {
		if _, err := os.Stat(path); err == nil {
			continue
		} else if err != nil && !os.IsNotExist(err) {
			tb.Fatalf("stat %s: %v", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			tb.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			tb.Fatalf("write stub %s: %v", path, err)
		}
	}
}

func normaliseModulePath(modulePath string) string {
	trimmed := strings.TrimSpace(modulePath)
	if trimmed == "" {
		return "github.com/your/module"
	}
	return trimmed
}

// RemoveGraphQLRuntimeStub removes the minimal ORM stub added by ScaffoldGraphQLRuntime.
func RemoveGraphQLRuntimeStub(tb stdtesting.TB, root string) {
	tb.Helper()

	stub := filepath.Join(root, "orm", "gen", "client.go")
	if err := os.Remove(stub); err != nil && !os.IsNotExist(err) {
		tb.Fatalf("remove stub %s: %v", stub, err)
	}
}
