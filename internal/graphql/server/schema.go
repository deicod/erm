package server

import (
	"context"

	gql "github.com/99designs/gqlgen/graphql"

	"github.com/deicod/erm/internal/graphql"
	"github.com/deicod/erm/internal/graphql/dataloaders"
	"github.com/deicod/erm/internal/graphql/directives"
	"github.com/deicod/erm/internal/graphql/resolvers"
	"github.com/deicod/erm/internal/observability/metrics"
	"github.com/deicod/erm/internal/orm/gen"
)

// Options configures the executable schema and request scaffolding.
type Options struct {
	ORM       *gen.Client
	Collector metrics.Collector
}

// NewExecutableSchema builds a gqlgen executable schema with default directives wired in.
func NewExecutableSchema(opts Options) gql.ExecutableSchema {
	collector := metrics.WithCollector(opts.Collector)
	resolver := resolvers.NewWithOptions(resolvers.Options{ORM: opts.ORM, Collector: collector})
	cfg := graphql.Config{
		Resolvers: resolver,
		Directives: graphql.DirectiveRoot{
			Auth: func(ctx context.Context, obj any, next gql.Resolver, roles []string) (any, error) {
				handler := directives.RequireAuth()
				if len(roles) > 0 {
					handler = directives.RequireRoles(roles)
				}
				return handler(ctx, obj, func(ctx context.Context) (interface{}, error) {
					return next(ctx)
				})
			},
		},
	}
	return graphql.NewExecutableSchema(cfg)
}

// WithLoaders decorates the provided context with request-scoped dataloaders.
func WithLoaders(ctx context.Context, opts Options) context.Context {
	collector := metrics.WithCollector(opts.Collector)
	loaders := dataloaders.New(opts.ORM, collector)
	return dataloaders.ToContext(ctx, loaders)
}
