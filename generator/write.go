package generator

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
)

type fileWriter interface {
	Write(path string, content []byte) (bool, error)
}

type defaultFileWriter struct{}

func (defaultFileWriter) Write(path string, content []byte) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if !forceWrite {
		if existing, err := os.ReadFile(path); err == nil {
			if bytes.Equal(existing, content) {
				return false, nil
			}
		}
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

var activeWriter fileWriter = defaultFileWriter{}

func writeGoFile(path string, content []byte) error {
	formatted, err := format.Source(content)
	if err != nil {
		return err
	}
	_, err = writeFile(path, formatted)
	return err
}

func writeFile(path string, content []byte) (bool, error) {
	return activeWriter.Write(path, content)
}
