package cli

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

type dockerComposeData struct {
	Image           string
	ImageComment    string
	ContainerName   string
	PortMapping     string
	DataVolume      string
	DataVolumeMount string
	InitDBMount     string
	Database        string
	User            string
	Password        string
	ExtensionsNote  string
}

type dbSettings struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
}

func newDockerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Manage Docker assets for local development",
	}
	cmd.AddCommand(newDockerSyncCmd())
	return cmd
}

func newDockerSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Generate docker-compose scaffolding aligned with erm.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadProjectConfig(".")
			if err != nil {
				return wrapError(
					"docker sync: load config",
					err,
					"Ensure erm.yaml exists and is valid YAML.",
					1,
				)
			}
			if err := syncDockerAssets(".", cfg); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Synchronized docker/local assets.")
			return nil
		},
	}
	return cmd
}

func syncDockerAssets(root string, cfg projectConfig) error {
	compose, note, err := renderDockerCompose(cfg)
	if err != nil {
		return wrapError(
			"docker sync: render docker-compose.yml",
			err,
			"Report this issue to the erm maintainers with your configuration.",
			1,
		)
	}
	composePath := filepath.Join(root, "docker", "local", "docker-compose.yml")
	if err := ensureDir(filepath.Dir(composePath)); err != nil {
		return wrapError(
			fmt.Sprintf("docker sync: create directory %s", filepath.Dir(composePath)),
			err,
			"Check directory permissions or run the command from a writable workspace.",
			1,
		)
	}
	if err := os.WriteFile(composePath, compose, 0o644); err != nil {
		return wrapError(
			fmt.Sprintf("docker sync: write file %s", composePath),
			err,
			"Ensure the path is writable and not protected by source control attributes.",
			1,
		)
	}

	initDir := filepath.Join(root, "docker", "local", "initdb.d")
	if err := ensureDir(initDir); err != nil {
		return wrapError(
			fmt.Sprintf("docker sync: create directory %s", initDir),
			err,
			"Check directory permissions or run the command from a writable workspace.",
			1,
		)
	}
	initPath := filepath.Join(initDir, "extensions.sql")
	sql := renderExtensionSQL(cfg, note)
	if err := os.WriteFile(initPath, sql, 0o644); err != nil {
		return wrapError(
			fmt.Sprintf("docker sync: write file %s", initPath),
			err,
			"Ensure the path is writable and not protected by source control attributes.",
			1,
		)
	}
	return nil
}

func renderDockerCompose(cfg projectConfig) ([]byte, string, error) {
	settings := parseDatabaseURL(cfg.Database.URL)
	extensions := enabledExtensions(cfg)
	image, comment := selectDockerImage(cfg)
	slug := slugify(settings.Database, "postgres")
	volume := fmt.Sprintf("%s-data", slug)

	data := dockerComposeData{
		Image:           image,
		ImageComment:    comment,
		ContainerName:   fmt.Sprintf("%s-postgres", slug),
		PortMapping:     fmt.Sprintf("%s:5432", settings.Port),
		DataVolume:      volume,
		DataVolumeMount: fmt.Sprintf("%s:/var/lib/postgresql/data", volume),
		InitDBMount:     "./initdb.d:/docker-entrypoint-initdb.d",
		Database:        settings.Database,
		User:            settings.User,
		Password:        settings.Password,
	}
	if len(extensions) > 0 {
		data.ExtensionsNote = fmt.Sprintf("Enabled extensions: %s", strings.Join(extensions, ", "))
	}

	buf := &bytes.Buffer{}
	tpl := template.Must(template.New("docker-compose").Funcs(template.FuncMap{"quote": yamlQuote}).Parse(dockerComposeTemplate))
	if err := tpl.Execute(buf, data); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), data.ExtensionsNote, nil
}

func renderExtensionSQL(cfg projectConfig, note string) []byte {
	extensions := enabledExtensions(cfg)
	var b strings.Builder
	b.WriteString("-- Managed by `erm docker sync`.\n")
	if note != "" {
		b.WriteString(fmt.Sprintf("-- %s\n", note))
	}
	if len(extensions) == 0 {
		b.WriteString("-- Enable extensions in erm.yaml to generate CREATE EXTENSION statements.\n")
		return []byte(b.String())
	}
	b.WriteString("\n")
	for _, ext := range extensions {
		switch ext {
		case "postgis":
			b.WriteString("CREATE EXTENSION IF NOT EXISTS postgis;\n")
		case "pgvector":
			b.WriteString("CREATE EXTENSION IF NOT EXISTS vector;\n")
		case "timescaledb":
			b.WriteString("CREATE EXTENSION IF NOT EXISTS timescaledb;\n")
		}
	}
	return []byte(b.String())
}

func parseDatabaseURL(raw string) dbSettings {
	defaults := dbSettings{
		Host:     "localhost",
		Port:     "5432",
		Database: "app",
		User:     "postgres",
		Password: "postgres",
	}
	if strings.TrimSpace(raw) == "" {
		return defaults
	}
	u, err := url.Parse(raw)
	if err != nil {
		return defaults
	}
	if host := u.Hostname(); host != "" {
		defaults.Host = host
	}
	if port := u.Port(); port != "" {
		defaults.Port = port
	}
	if path := strings.TrimPrefix(u.Path, "/"); path != "" {
		defaults.Database = path
	}
	if user := u.User.Username(); user != "" {
		defaults.User = user
	}
	if password, ok := u.User.Password(); ok {
		defaults.Password = password
	}
	return defaults
}

func slugify(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		value = fallback
	}
	lower := strings.ToLower(value)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug := strings.Trim(re.ReplaceAllString(lower, "-"), "-")
	if slug == "" {
		slug = fallback
	}
	return slug
}

func enabledExtensions(cfg projectConfig) []string {
	var extensions []string
	if cfg.Extensions.PostGIS {
		extensions = append(extensions, "postgis")
	}
	if cfg.Extensions.PGVector {
		extensions = append(extensions, "pgvector")
	}
	if cfg.Extensions.Timescale {
		extensions = append(extensions, "timescaledb")
	}
	sort.Strings(extensions)
	return extensions
}

func selectDockerImage(cfg projectConfig) (string, string) {
	switch {
	case cfg.Extensions.PostGIS && cfg.Extensions.Timescale && !cfg.Extensions.PGVector:
		return "timescale/timescaledb-ha:pg16.2-ts2.14.2", "Includes TimescaleDB and PostGIS support."
	case cfg.Extensions.PostGIS && !cfg.Extensions.PGVector && !cfg.Extensions.Timescale:
		return "postgis/postgis:16-3.4", "Includes PostGIS support."
	case cfg.Extensions.PGVector && !cfg.Extensions.PostGIS && !cfg.Extensions.Timescale:
		return "pgvector/pgvector:pg16", "Includes the pgvector extension."
	case cfg.Extensions.Timescale && !cfg.Extensions.PostGIS && !cfg.Extensions.PGVector:
		return "timescale/timescaledb-ha:pg16.2-ts2.14.2", "Includes TimescaleDB support."
	default:
		return "postgres:16", "Use a custom image to bundle additional extensions."
	}
}

func yamlQuote(value string) string {
	return fmt.Sprintf("\"%s\"", strings.ReplaceAll(value, "\"", "\\\""))
}

const dockerComposeTemplate = `# Generated by erm docker sync. Do not edit manually.
services:
  postgres:
    container_name: {{ quote .ContainerName }}
    image: {{ quote .Image }}
{{- if .ImageComment }}
    # {{ .ImageComment }}
{{- end }}
{{- if .ExtensionsNote }}
    # {{ .ExtensionsNote }}
{{- end }}
    restart: unless-stopped
    ports:
      - {{ quote .PortMapping }}
    environment:
      POSTGRES_DB: {{ quote .Database }}
      POSTGRES_USER: {{ quote .User }}
      POSTGRES_PASSWORD: {{ quote .Password }}
    volumes:
      - {{ quote .DataVolumeMount }}
      - {{ quote .InitDBMount }}
    healthcheck:
      test:
        - "CMD-SHELL"
        - "pg_isready -d {{ .Database }} -U {{ .User }}"
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 5s
volumes:
  {{ .DataVolume }}:
    driver: local

`
