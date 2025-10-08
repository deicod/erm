package generator

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
)

func writeGoFile(path string, content []byte) error {
	formatted, err := format.Source(content)
	if err != nil {
		return err
	}
	return writeFile(path, formatted)
}

func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if existing, err := os.ReadFile(path); err == nil {
		if bytes.Equal(existing, content) {
			return nil
		}
	}
	return os.WriteFile(path, content, 0o644)
}
