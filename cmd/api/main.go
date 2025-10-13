package main

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

        "github.com/deicod/erm/graphql/server"
        "github.com/deicod/erm/observability/metrics"
        "github.com/deicod/erm/orm/gen"

        "github.com/deicod/erm/orm/pg"
        "gopkg.in/yaml.v3"
)
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
        Database databaseConfig `yaml:"database"`
        GraphQL  graphQLConfig  `yaml:"graphql"`
}

type databaseConfig struct {
        URL          string                         `yaml:"url"`
        Pool         poolConfig                     `yaml:"pool"`
        Replicas     []replicaConfig                `yaml:"replicas"`
        Routing      replicaRoutingConfig           `yaml:"routing"`
        Environments map[string]databaseEnvironment `yaml:"environments"`
}

type databaseEnvironment struct {
        URL string `yaml:"url"`
}

type poolConfig struct {
        MaxConns          int32         `yaml:"max_conns"`
        MinConns          int32         `yaml:"min_conns"`
        MaxConnLifetime   time.Duration `yaml:"max_conn_lifetime"`
        MaxConnIdleTime   time.Duration `yaml:"max_conn_idle_time"`
        HealthCheckPeriod time.Duration `yaml:"health_check_period"`
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

type graphQLConfig struct {
        Path          string `yaml:"path"`
        Subscriptions struct {
                Enabled    bool `yaml:"enabled"`
                Transports struct {
                        Websocket bool `yaml:"websocket"`
                        GraphQLWS bool `yaml:"graphql_ws"`
                } `yaml:"transports"`
        } `yaml:"subscriptions"`
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
