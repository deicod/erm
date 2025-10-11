package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/deicod/erm/orm/pg"
)

type projectConfig struct {
	Module   string `yaml:"module"`
	Database struct {
		URL          string                         `yaml:"url"`
		Pool         poolConfig                     `yaml:"pool"`
		Replicas     []replicaConfig                `yaml:"replicas"`
		Routing      replicaRoutingConfig           `yaml:"routing"`
		Environments map[string]databaseEnvironment `yaml:"environments"`
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

func (cfg projectConfig) ReplicaConfigs() []pg.ReplicaConfig {
	if len(cfg.Database.Replicas) == 0 {
		return nil
	}
	replicas := make([]pg.ReplicaConfig, 0, len(cfg.Database.Replicas))
	for _, replica := range cfg.Database.Replicas {
		replicas = append(replicas, pg.ReplicaConfig{
			Name:           replica.Name,
			URL:            replica.URL,
			ReadOnly:       replica.ReadOnly,
			MaxFollowerLag: replica.MaxFollowerLag,
		})
	}
	return replicas
}

func (cfg projectConfig) ReplicaPolicies() (string, map[string]pg.ReplicaReadOptions) {
	if len(cfg.Database.Routing.Policies) == 0 {
		return cfg.Database.Routing.DefaultPolicy, nil
	}
	policies := make(map[string]pg.ReplicaReadOptions, len(cfg.Database.Routing.Policies))
	for name, policy := range cfg.Database.Routing.Policies {
		policies[name] = pg.ReplicaReadOptions{
			MaxLag:          policy.MaxFollowerLag,
			RequireReadOnly: policy.ReadOnly,
			DisableFallback: policy.DisableFallback,
		}
	}
	return cfg.Database.Routing.DefaultPolicy, policies
}

type poolConfig struct {
	MaxConns          int32         `yaml:"max_conns"`
	MinConns          int32         `yaml:"min_conns"`
	MaxConnLifetime   time.Duration `yaml:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `yaml:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `yaml:"health_check_period"`
}

type databaseEnvironment struct {
	URL string `yaml:"url"`
}

type replicaConfig struct {
	Name           string        `yaml:"name"`
	URL            string        `yaml:"url"`
	ReadOnly       bool          `yaml:"read_only"`
	MaxFollowerLag time.Duration `yaml:"max_follower_lag"`
}

type replicaRoutingConfig struct {
	DefaultPolicy string                         `yaml:"default_policy"`
	Policies      map[string]replicaPolicyConfig `yaml:"policies"`
}

type replicaPolicyConfig struct {
	ReadOnly        bool          `yaml:"read_only"`
	MaxFollowerLag  time.Duration `yaml:"max_follower_lag"`
	DisableFallback bool          `yaml:"disable_fallback"`
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
