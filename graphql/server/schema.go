package server

import (
	"context"

	gql "github.com/99designs/gqlgen/graphql"

	"github.com/deicod/erm/graphql"
	"github.com/deicod/erm/graphql/dataloaders"
	"github.com/deicod/erm/graphql/directives"
	"github.com/deicod/erm/graphql/resolvers"
	"github.com/deicod/erm/graphql/subscriptions"
	"github.com/deicod/erm/observability/metrics"
	"github.com/deicod/erm/orm/gen"
)

// Options configures the executable schema and request scaffolding.
type Options struct {
	ORM           *gen.Client
	Collector     metrics.Collector
	Subscriptions SubscriptionOptions
}

type SubscriptionOptions struct {
	Enabled    bool
	Broker     subscriptions.Broker
	Transports SubscriptionTransports
}

type SubscriptionTransports struct {
	Websocket bool
	GraphQLWS bool
}

// NewExecutableSchema builds a gqlgen executable schema with default directives wired in.
func NewExecutableSchema(opts Options) gql.ExecutableSchema {
	opts = normaliseOptions(opts)
	collector := metrics.WithCollector(opts.Collector)
	resolver := resolvers.NewWithOptions(resolvers.Options{ORM: opts.ORM, Collector: collector, Subscriptions: opts.Subscriptions.Broker})
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
	opts = normaliseOptions(opts)
	collector := metrics.WithCollector(opts.Collector)
	loaders := dataloaders.New(opts.ORM, collector)
	return dataloaders.ToContext(ctx, loaders)
}

func normaliseOptions(opts Options) Options {
	subs := opts.Subscriptions
	if subs.Enabled && subs.Broker == nil {
		subs.Broker = subscriptions.NewInMemoryBroker()
	}
	if subs.Enabled {
		if !subs.Transports.Websocket && !subs.Transports.GraphQLWS {
			subs.Transports.Websocket = true
		}
	}
	opts.Subscriptions = subs
	return opts
}
