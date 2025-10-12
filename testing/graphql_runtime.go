package testkit

import (
	"os"
	"path/filepath"
	stdtesting "testing"
)

// ScaffoldGraphQLRuntime ensures GraphQL runtime dependencies exist so generated code can compile.
func ScaffoldGraphQLRuntime(tb stdtesting.TB, root, modulePath string) {
	tb.Helper()

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
		filepath.Join(root, "graphql", "directives", "auth.go"): `package directives

import (
        "context"

        gql "github.com/99designs/gqlgen/graphql"
)

func Auth(ctx context.Context, obj interface{}, next gql.Resolver, roles []string) (interface{}, error) {
        return next(ctx)
}
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
		filepath.Join(root, "graphql", "relay", "id.go"): `package relay

import (
        "encoding/base64"
        "fmt"
        "strings"
)

func MarshalID(typ, id string) string {
        return base64.StdEncoding.EncodeToString([]byte(typ + ":" + id))
}

func UnmarshalID(value string) (string, string, error) {
        decoded, err := base64.StdEncoding.DecodeString(value)
        if err != nil {
                return "", "", err
        }
        parts := strings.SplitN(string(decoded), ":", 2)
        if len(parts) != 2 {
                return "", "", fmt.Errorf("invalid relay id: %s", value)
        }
        return parts[0], parts[1], nil
}
`,
		filepath.Join(root, "graphql", "scalars.go"): `package graphql

// Placeholder scalars stub to satisfy go tooling in tests.
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
		filepath.Join(root, "graphql", "server", "schema.go"): `package server

import gql "github.com/99designs/gqlgen/graphql"

type ExecutableSchema = gql.ExecutableSchema

func NewExecutableSchema(Options) gql.ExecutableSchema { return nil }
`,
	}

	for path, content := range stubs {
		if _, err := os.Stat(path); err == nil {
			continue
		} else if err != nil && !os.IsNotExist(err) {
			tb.Fatalf("stat %s: %v", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			tb.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			tb.Fatalf("write stub %s: %v", path, err)
		}
	}
}
