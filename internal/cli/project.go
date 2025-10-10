package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/deicod/erm/internal/orm/pg"
)

type projectConfig struct {
	Module   string `yaml:"module"`
	Database struct {
		URL  string     `yaml:"url"`
		Pool poolConfig `yaml:"pool"`
	} `yaml:"database"`
	Observability struct {
		ORM struct {
			QueryLogging   bool `yaml:"query_logging"`
			EmitSpans      bool `yaml:"emit_spans"`
			CorrelationIDs bool `yaml:"correlation_ids"`
		} `yaml:"orm"`
	} `yaml:"observability"`
}

func (cfg projectConfig) QueryLoggingEnabled() bool { return cfg.Observability.ORM.QueryLogging }

func (cfg projectConfig) QuerySpansEnabled() bool { return cfg.Observability.ORM.EmitSpans }

func (cfg projectConfig) QueryCorrelationEnabled() bool { return cfg.Observability.ORM.CorrelationIDs }

func (cfg projectConfig) PoolOption() pg.Option {
	if opt := cfg.Database.Pool.Option(); opt != nil {
		return opt
	}
	return nil
}

type poolConfig struct {
	MaxConns          int32         `yaml:"max_conns"`
	MinConns          int32         `yaml:"min_conns"`
	MaxConnLifetime   time.Duration `yaml:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `yaml:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `yaml:"health_check_period"`
}

func (pc poolConfig) Option() pg.Option {
	if pc.MaxConns == 0 && pc.MinConns == 0 && pc.MaxConnLifetime == 0 && pc.MaxConnIdleTime == 0 && pc.HealthCheckPeriod == 0 {
		return nil
	}
	return pg.WithPoolConfig(pg.PoolConfig{
		MaxConns:          pc.MaxConns,
		MinConns:          pc.MinConns,
		MaxConnLifetime:   pc.MaxConnLifetime,
		MaxConnIdleTime:   pc.MaxConnIdleTime,
		HealthCheckPeriod: pc.HealthCheckPeriod,
	})
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
