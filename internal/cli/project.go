package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type projectConfig struct {
	Module   string `yaml:"module"`
	Database struct {
		URL string `yaml:"url"`
	} `yaml:"database"`
}

func loadProjectConfig(root string) (projectConfig, error) {
	path := filepath.Join(root, "erm.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return projectConfig{}, nil
		}
		return projectConfig{}, err
	}
	var cfg projectConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return projectConfig{}, err
	}
	return cfg, nil
}

func detectModule(root string) string {
	if cfg, err := loadProjectConfig(root); err == nil && cfg.Module != "" {
		return cfg.Module
	}
	return moduleFromGoMod(root)
}

func projectModule(root string) string {
	if cfg, err := loadProjectConfig(root); err == nil && cfg.Module != "" {
		return cfg.Module
	}
	return moduleFromGoMod(root)
}

func moduleFromGoMod(root string) string {
	path := filepath.Join(root, "go.mod")
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func writeFileOnce(path string, content []byte) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
