package testkit

import (
	"context"
	"fmt"
	stdtesting "testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/deicod/erm/internal/graphql/server"
	"github.com/deicod/erm/internal/observability/metrics"
	"github.com/deicod/erm/internal/orm/gen"
)

// GraphQLHarnessOptions configures the in-memory GraphQL executor used in tests.
type GraphQLHarnessOptions struct {
	ORM           *gen.Client
	Collector     metrics.Collector
	ClientOptions []client.Option
}

// GraphQLHarness wires the generated schema, resolvers, and dataloaders together for tests.
type GraphQLHarness struct {
	options  server.Options
	client   *client.Client
	baseOpts []client.Option
}

// NewGraphQLHarness constructs a harness backed by the provided ORM client.
func NewGraphQLHarness(tb stdtesting.TB, opts GraphQLHarnessOptions) *GraphQLHarness {
	tb.Helper()
	if opts.ORM == nil {
		tb.Fatalf("ORM client is required")
	}
	serverOpts := server.Options{ORM: opts.ORM, Collector: opts.Collector}
	execSchema := server.NewExecutableSchema(serverOpts)
	srv := handler.NewDefaultServer(execSchema)
	harness := &GraphQLHarness{
		options:  serverOpts,
		client:   client.New(srv),
		baseOpts: append([]client.Option(nil), opts.ClientOptions...),
	}
	return harness
}

// Exec issues a GraphQL operation against the executable schema using request-scoped dataloaders.
func (h *GraphQLHarness) Exec(ctx context.Context, query string, resp any, opts ...client.Option) error {
	if h == nil {
		return fmt.Errorf("graphQL harness is not initialised")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = server.WithLoaders(ctx, h.options)
	allOpts := append(append([]client.Option{}, h.baseOpts...), opts...)
	allOpts = append(allOpts, withContext(ctx))
	return h.client.Post(query, resp, allOpts...)
}

// MustExec is a convenience wrapper that fails the supplied test if the request errors.
func (h *GraphQLHarness) MustExec(tb stdtesting.TB, ctx context.Context, query string, resp any, opts ...client.Option) {
	tb.Helper()
	if err := h.Exec(ctx, query, resp, opts...); err != nil {
		tb.Fatalf("graphql exec: %v", err)
	}
}

// Client exposes the underlying gqlgen test client for advanced assertions.
func (h *GraphQLHarness) Client() *client.Client {
	if h == nil {
		return nil
	}
	return h.client
}

func withContext(ctx context.Context) client.Option {
	return func(r *client.Request) {
		if ctx == nil {
			ctx = context.Background()
		}
		r.HTTP = r.HTTP.WithContext(ctx)
	}
}
