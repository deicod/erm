package generator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/deicod/erm/templates"
)

// EnsureRuntimeScaffolds materialises the runtime scaffolding packages required by generated code.
// It writes files only when they are absent so repeated executions remain idempotent.
func EnsureRuntimeScaffolds(root, modulePath string) error {
	rendered, err := templates.RenderRuntimeScaffolds(normaliseModulePath(modulePath))
	if err != nil {
		return err
	}
	for rel, content := range rendered {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func normaliseModulePath(modulePath string) string {
	trimmed := strings.TrimSpace(modulePath)
	if trimmed == "" {
		return "github.com/your/module"
	}
	return trimmed
}
