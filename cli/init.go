package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/deicod/erm/generator"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize erm in the current workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			modulePath := detectModule(".")
			if modulePath == "" {
				fmt.Fprintln(out, "Warning: module path not detected; update go.mod or erm.yaml so GraphQL imports can be rewritten.")
			}

			files := []struct{ path, content string }{
				{"erm.yaml", defaultConfig},
				{"README.md", workspaceReadme},
				{"AGENTS.md", workspaceAgents},
				{"cmd/api/main.go", renderAPIMain(modulePath)},
				{"schema/.gitkeep", ""},
				{"schema/AGENTS.md", schemaAgents},
				{"graphql/README.md", gqlReadme},
				{"migrations/.gitkeep", ""},
			}

			for _, f := range files {
				if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
					return wrapError(
						fmt.Sprintf("init: create directory %s", filepath.Dir(f.path)),
						err,
						"Check directory permissions or run the command from a writable workspace.",
						1,
					)
				}
				if _, err := os.Stat(f.path); err == nil {
					continue // idempotent
				}
				if err := os.WriteFile(f.path, []byte(f.content), 0o644); err != nil {
					return wrapError(
						fmt.Sprintf("init: write file %s", f.path),
						err,
						"Ensure the path is writable and not protected by source control attributes.",
						1,
					)
				}
			}
			if err := generator.EnsureRuntimeScaffolds(".", modulePath); err != nil {
				return wrapError("init: scaffold runtime packages", err, "Report this issue to the erm maintainers.", 1)
			}

			fmt.Fprintln(out, "Initialized erm workspace.")
			return nil
		},
	}
	return cmd
}

const defaultConfig = `# erm configuration
# 1. Set module to your Go module path (for example "github.com/acme/app").
module: ""
database:
  # 2. Update the connection string to point at your development database.
  url: "postgres://user:pass@localhost:5432/app?sslmode=disable"
oidc:
  # 3. Configure the issuer and audience to match your identity provider.
  issuer: "https://auth.example.com/realms/app"
  audience: "web-spa"
graphql:
  # 4. The HTTP path your API will be served on.
  path: "/graphql"
extensions:
  postgis: true
  pgvector: true
  timescaledb: false
`

const workspaceReadme = `# Welcome to your erm project

This workspace was bootstrapped with 'erm init'. The goal of the generated
skeleton is to give you a runnable HTTP service, a place to define your schema,
and a repeatable workflow for regenerating code as your domain evolves.

## Next steps

1. Initialize your Go module and align 'erm.yaml':
       go mod init <module>
       go mod tidy
   Update 'module' in 'erm.yaml' to match the value passed to 'go mod init'.
2. Sketch your first entity with 'erm new <Entity>'.
3. Run 'erm gen' to materialize ORM, GraphQL, and migration artifacts.
4. Start the HTTP server with 'go run ./cmd/api' and iterate.

## Project layout

- 'cmd/api' — entrypoint for the HTTP server and integration glue.
- 'schema' — your application schema. Run 'erm gen' whenever it changes.
- 'graphql' — gqlgen configuration and generated resolvers.
- 'migrations' — versioned SQL migrations managed by 'erm gen'.

## Recommended workflow

1. Practice TDD: write or update tests alongside feature work.
2. Keep the code formatted with 'gofmt -w' on edited files.
3. Validate changes locally:
   - 'go test ./...'
   - 'go test -race ./...'
   - 'go vet ./...'
4. Regenerate artifacts with 'erm gen' and review the diff before committing.
5. Use 'erm migrate' to apply database changes during development.

Happy hacking!
`

const workspaceAgents = `# Development Workflow

When contributing to this workspace:

1. Follow TDD — add or update tests before changing behavior.
2. Keep formatting clean: run 'gofmt -w' on touched Go files.
3. Before pushing a branch, validate with:
   - 'go test ./...'
   - 'go test -race ./...'
   - 'go vet ./...'
4. Regenerate code after schema changes with 'erm gen' and commit the results.
5. Prefer small, reviewable commits and document notable workflows in the repo.
`

func renderAPIMain(modulePath string) string {
	data := struct {
		ModulePath  string
		ModuleUnset bool
		Backtick    string
	}{
		ModulePath: modulePath,
		Backtick:   "`",
	}
	if data.ModulePath == "" {
		data.ModulePath = "github.com/your/module"
		data.ModuleUnset = true
	}

	tpl := template.Must(template.New("apiMain").Parse(apiMainTemplate))
	buf := &bytes.Buffer{}
	_ = tpl.Execute(buf, data)
	return buf.String()
}

const apiMainTemplate = `package main

import (
        "context"
        "errors"
        "log"
        "net/http"
        "os"
        "os/signal"
        "strings"
        "syscall"
        "time"

        "{{.ModulePath}}/graphql/server"
        "{{.ModulePath}}/observability/metrics"
        "{{.ModulePath}}/orm/gen"

        "github.com/deicod/erm/orm/pg"
        "gopkg.in/yaml.v3"
)

{{- if .ModuleUnset }}// TODO: Set the module path in go.mod or erm.yaml, then regenerate the imports above.

{{- end }}
func main() {
        ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
        defer stop()

        cfg, err := loadConfig("erm.yaml")
        if err != nil {
                log.Fatalf("load config: %v", err)
        }

        dbURL := resolveDatabaseURL(cfg.Database)
        if dbURL == "" {
                log.Fatal("database url is empty; set database.url in erm.yaml or export ERM_DATABASE_URL")
        }

        db, err := connectDatabase(ctx, cfg.Database, dbURL)
        if err != nil {
                log.Fatalf("connect database: %v", err)
        }
        defer db.Close()

        collector := metrics.NoopCollector{} // TODO: Replace with metrics.WithCollector(...) once observability plumbing is in place.

        ormClient := gen.NewClient(db)

        gqlOpts := server.Options{
                ORM:       ormClient,
                Collector: collector,
                Subscriptions: server.SubscriptionOptions{
                        Enabled: cfg.GraphQL.Subscriptions.Enabled,
                        Transports: server.SubscriptionTransports{
                                Websocket: cfg.GraphQL.Subscriptions.Transports.Websocket,
                                GraphQLWS: cfg.GraphQL.Subscriptions.Transports.GraphQLWS,
                        },
                },
        }

        graphqlServer := server.NewServer(gqlOpts)

        graphqlPath := resolveGraphQLPath(cfg.GraphQL)

        mux := http.NewServeMux()
        mux.HandleFunc("/healthz", healthHandler)
        mux.Handle(graphqlPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                ctx := server.WithLoaders(r.Context(), gqlOpts)
                graphqlServer.ServeHTTP(w, r.WithContext(ctx))
        }))

        addr := resolveHTTPAddr()
        srv := &http.Server{
                Addr:    addr,
                Handler: mux,
        }

        errCh := make(chan error, 1)
        go func() {
                log.Printf("serving GraphQL on %s%s", addr, graphqlPath)
                if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
                        errCh <- err
                }
                close(errCh)
        }()

        select {
        case <-ctx.Done():
        case err := <-errCh:
                if err != nil {
                        log.Fatalf("server error: %v", err)
                }
                return
        }

        shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        if err := srv.Shutdown(shutdownCtx); err != nil {
                log.Printf("graceful shutdown failed: %v", err)
        }

        <-errCh
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
}

type config struct {
        Database databaseConfig {{.Backtick}}yaml:"database"{{.Backtick}}
        GraphQL  graphQLConfig  {{.Backtick}}yaml:"graphql"{{.Backtick}}
}

type databaseConfig struct {
        URL          string                         {{.Backtick}}yaml:"url"{{.Backtick}}
        Pool         poolConfig                     {{.Backtick}}yaml:"pool"{{.Backtick}}
        Replicas     []replicaConfig                {{.Backtick}}yaml:"replicas"{{.Backtick}}
        Routing      replicaRoutingConfig           {{.Backtick}}yaml:"routing"{{.Backtick}}
        Environments map[string]databaseEnvironment {{.Backtick}}yaml:"environments"{{.Backtick}}
}

type databaseEnvironment struct {
        URL string {{.Backtick}}yaml:"url"{{.Backtick}}
}

type poolConfig struct {
        MaxConns          int32         {{.Backtick}}yaml:"max_conns"{{.Backtick}}
        MinConns          int32         {{.Backtick}}yaml:"min_conns"{{.Backtick}}
        MaxConnLifetime   time.Duration {{.Backtick}}yaml:"max_conn_lifetime"{{.Backtick}}
        MaxConnIdleTime   time.Duration {{.Backtick}}yaml:"max_conn_idle_time"{{.Backtick}}
        HealthCheckPeriod time.Duration {{.Backtick}}yaml:"health_check_period"{{.Backtick}}
}

type replicaConfig struct {
        Name           string        {{.Backtick}}yaml:"name"{{.Backtick}}
        URL            string        {{.Backtick}}yaml:"url"{{.Backtick}}
        ReadOnly       bool          {{.Backtick}}yaml:"read_only"{{.Backtick}}
        MaxFollowerLag time.Duration {{.Backtick}}yaml:"max_follower_lag"{{.Backtick}}
}

type replicaRoutingConfig struct {
        DefaultPolicy string                         {{.Backtick}}yaml:"default_policy"{{.Backtick}}
        Policies      map[string]replicaPolicyConfig {{.Backtick}}yaml:"policies"{{.Backtick}}
}

type replicaPolicyConfig struct {
        ReadOnly        bool          {{.Backtick}}yaml:"read_only"{{.Backtick}}
        MaxFollowerLag  time.Duration {{.Backtick}}yaml:"max_follower_lag"{{.Backtick}}
        DisableFallback bool          {{.Backtick}}yaml:"disable_fallback"{{.Backtick}}
}

type graphQLConfig struct {
        Path          string {{.Backtick}}yaml:"path"{{.Backtick}}
        Subscriptions struct {
                Enabled    bool {{.Backtick}}yaml:"enabled"{{.Backtick}}
                Transports struct {
                        Websocket bool {{.Backtick}}yaml:"websocket"{{.Backtick}}
                        GraphQLWS bool {{.Backtick}}yaml:"graphql_ws"{{.Backtick}}
                } {{.Backtick}}yaml:"transports"{{.Backtick}}
        } {{.Backtick}}yaml:"subscriptions"{{.Backtick}}
}

func loadConfig(path string) (config, error) {
        raw, err := os.ReadFile(path)
        if err != nil {
                if errors.Is(err, os.ErrNotExist) {
                        return config{}, nil
                }
                return config{}, err
        }
        var cfg config
        if err := yaml.Unmarshal(raw, &cfg); err != nil {
                return config{}, err
        }
        return cfg, nil
}

func resolveDatabaseURL(cfg databaseConfig) string {
        if url := os.Getenv("ERM_DATABASE_URL"); url != "" {
                return url
        }
        if env := os.Getenv("ERM_ENV"); env != "" {
                if envCfg, ok := cfg.Environments[env]; ok && envCfg.URL != "" {
                        return envCfg.URL
                }
        }
        return cfg.URL
}

func connectDatabase(ctx context.Context, cfg databaseConfig, url string) (*pg.DB, error) {
        var opts []pg.Option
        if opt := cfg.Pool.option(); opt != nil {
                opts = append(opts, opt)
        }
        db, err := pg.ConnectCluster(ctx, url, cfg.replicaConfigs(), opts...)
        if err != nil {
                return nil, err
        }
        if def, policies := cfg.replicaPolicies(); def != "" || len(policies) > 0 {
                db.UseReplicaPolicies(def, policies)
        }
        return db, nil
}

func resolveGraphQLPath(cfg graphQLConfig) string {
        if path := os.Getenv("ERM_GRAPHQL_PATH"); path != "" {
                return path
        }
        if cfg.Path != "" {
                return cfg.Path
        }
        return "/graphql"
}

func resolveHTTPAddr() string {
        if addr := os.Getenv("ERM_HTTP_ADDR"); addr != "" {
                return addr
        }
        if port := os.Getenv("PORT"); port != "" {
                if strings.HasPrefix(port, ":") {
                        return port
                }
                return ":" + port
        }
        return ":8080"
}

func (pc poolConfig) option() pg.Option {
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

func (cfg databaseConfig) replicaConfigs() []pg.ReplicaConfig {
        if len(cfg.Replicas) == 0 {
                return nil
        }
        replicas := make([]pg.ReplicaConfig, 0, len(cfg.Replicas))
        for _, replica := range cfg.Replicas {
                if replica.URL == "" {
                        continue
                }
                replicas = append(replicas, pg.ReplicaConfig{
                        Name:           replica.Name,
                        URL:            replica.URL,
                        ReadOnly:       replica.ReadOnly,
                        MaxFollowerLag: replica.MaxFollowerLag,
                })
        }
        return replicas
}

func (cfg databaseConfig) replicaPolicies() (string, map[string]pg.ReplicaReadOptions) {
        if len(cfg.Routing.Policies) == 0 {
                return cfg.Routing.DefaultPolicy, nil
        }
        policies := make(map[string]pg.ReplicaReadOptions, len(cfg.Routing.Policies))
        for name, policy := range cfg.Routing.Policies {
                policies[name] = pg.ReplicaReadOptions{
                        MaxLag:          policy.MaxFollowerLag,
                        RequireReadOnly: policy.ReadOnly,
                        DisableFallback: policy.DisableFallback,
                }
        }
        return cfg.Routing.DefaultPolicy, policies
}
`

const schemaAgents = `# Schema Development Workflow

When touching files in this directory:

1. Practice TDD — write or update tests alongside schema changes.
2. Run 'gofmt -w' on edited schema files before committing.
3. Validate the project with:
   - 'go test ./...'
   - 'go test -race ./...'
   - 'go vet ./...'
4. Regenerate code with 'erm gen' when the schema shape changes and review the diff before committing.
`

const gqlReadme = `# GraphQL workspace

This directory will hold gqlgen configuration, generated schema stubs, and
resolver implementations.

Run 'erm graphql init' to scaffold gqlgen, then wire the generated handler into
'cmd/api/main.go'.

The recommended workflow is:

1. Define or update schema entities under 'schema/'.
2. Execute 'erm gen' to refresh ORM, GraphQL, and migration assets.
3. Implement resolver logic and keep tests under 'graphql' up to date.
`
