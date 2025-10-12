package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGraphQLInitScaffoldsRuntimePackages(t *testing.T) {
	tmpDir := t.TempDir()
	modulePath := "github.com/example/app"

	goMod := "module " + modulePath + "\n\ngo 1.21\n\nrequire (\n\tgithub.com/99designs/gqlgen v0.17.80\n\tgithub.com/vektah/gqlparser/v2 v2.5.30\n)\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cmd := newGraphQLInitCmd()
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("execute graphql init: %v", err)
	}

	expectedFiles := []string{
		"graphql/gqlgen.yml",
		"graphql/schema.graphqls",
		"graphql/dataloaders/loader.go",
		"graphql/directives/auth.go",
		"graphql/relay/id.go",
		"graphql/scalars.go",
		"graphql/server/schema.go",
		"graphql/server/server.go",
		"graphql/subscriptions/bus.go",
	}

	for _, path := range expectedFiles {
		if _, err := os.Stat(filepath.Join(tmpDir, path)); err != nil {
			t.Fatalf("expected runtime file %s: %v", path, err)
		}
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "graphql", "directives", "auth.go"))
	if err != nil {
		t.Fatalf("read directives/auth.go: %v", err)
	}
	if !strings.Contains(string(content), modulePath+"/oidc") {
		t.Fatalf("module path not substituted in directives/auth.go: %s", content)
	}

	scaffoldRuntimeDependencies(t, tmpDir, modulePath)

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, output)
	}

	buildCmd := exec.Command("go", "test", "./graphql/dataloaders", "./graphql/directives", "./graphql/relay", "./graphql/server", "./graphql/subscriptions")
	buildCmd.Dir = tmpDir
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compile runtime packages: %v\n%s", err, output)
	}
}

func scaffoldRuntimeDependencies(t *testing.T, root, modulePath string) {
	t.Helper()

	stubs := map[string]string{
		filepath.Join(root, "orm", "gen", "client.go"): `package gen

type Client struct{}

func NewClient(any) *Client { return &Client{} }
`,
		filepath.Join(root, "observability", "metrics", "metrics.go"): `package metrics

import "time"

type Collector interface {
        RecordDataloaderBatch(string, int, time.Duration)
        RecordQuery(string, string, time.Duration, error)
}

type NoopCollector struct{}

func (NoopCollector) RecordDataloaderBatch(string, int, time.Duration) {}

func (NoopCollector) RecordQuery(string, string, time.Duration, error) {}

func WithCollector(primary Collector, others ...Collector) Collector {
        if primary != nil {
                return primary
        }
        return NoopCollector{}
}
`,
		filepath.Join(root, "oidc", "claims.go"): `package oidc

import "context"

type Claims struct {
        Roles []string
}

type claimsKey struct{}

func ToContext(ctx context.Context, claims Claims) context.Context {
        if ctx == nil {
                return context.Background()
        }
        return context.WithValue(ctx, claimsKey{}, claims)
}

func FromContext(ctx context.Context) (Claims, bool) {
        if ctx == nil {
                return Claims{}, false
        }
        claims, ok := ctx.Value(claimsKey{}).(Claims)
        return claims, ok
}
`,
		filepath.Join(root, "graphql", "dataloaders", "entities_gen.go"): `package dataloaders

import (
        "` + modulePath + `/observability/metrics"
        "` + modulePath + `/orm/gen"
)

func configureEntityLoaders(*Loaders, *gen.Client, metrics.Collector) {}
`,
		filepath.Join(root, "graphql", "graphql.go"): `package graphql

import (
        "context"

        gql "github.com/99designs/gqlgen/graphql"
)

type ExecutableSchema = gql.ExecutableSchema

type DirectiveRoot struct {
        Auth func(ctx context.Context, obj interface{}, next gql.Resolver, roles []string) (interface{}, error)
}

type Config struct {
        Resolvers interface{}
        Directives DirectiveRoot
}

type executionContext struct{}

func NewExecutableSchema(Config) gql.ExecutableSchema { return nil }
`,
		filepath.Join(root, "graphql", "resolvers", "resolver.go"): `package resolvers

import (
        "` + modulePath + `/graphql/subscriptions"
        "` + modulePath + `/observability/metrics"
        "` + modulePath + `/orm/gen"
)

type Options struct {
        ORM           *gen.Client
        Collector     metrics.Collector
        Subscriptions subscriptions.Broker
}

type Resolver struct {
        ORM *gen.Client
}

func NewWithOptions(opts Options) *Resolver { return &Resolver{ORM: opts.ORM} }
`,
		filepath.Join(root, "graphql", "subscriptions", "bus.go"): `package subscriptions

import "context"

type Broker interface {
        Publish(context.Context, string, any) error
        Subscribe(context.Context, string) (<-chan any, func(), error)
}

type InMemoryBroker struct{}

func NewInMemoryBroker() *InMemoryBroker { return &InMemoryBroker{} }

func (*InMemoryBroker) Publish(context.Context, string, any) error { return nil }

func (*InMemoryBroker) Subscribe(context.Context, string) (<-chan any, func(), error) {
        ch := make(chan any)
        return ch, func() { close(ch) }, nil
}
`,
		filepath.Join(root, "graphql", "server", "server.go"): `package server

import (
        "context"
        "net/http"

        "` + modulePath + `/observability/metrics"
        "` + modulePath + `/orm/gen"
)

type Options struct {
        ORM           *gen.Client
        Collector     metrics.Collector
        Subscriptions SubscriptionOptions
}

type SubscriptionOptions struct {
        Enabled    bool
        Broker     interface{}
        Transports SubscriptionTransports
}

type SubscriptionTransports struct {
        Websocket bool
        GraphQLWS bool
}

type Server struct{}

func NewServer(Options) *Server { return &Server{} }

func (Server) ServeHTTP(http.ResponseWriter, *http.Request) {}

func WithLoaders(ctx context.Context, _ Options) context.Context { return ctx }
`,
	}

	for path, content := range stubs {
		if _, err := os.Stat(path); err == nil {
			continue
		} else if err != nil && !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write stub %s: %v", path, err)
		}
	}
}
