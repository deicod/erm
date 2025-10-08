package resolvers

import (
	"context"

	"github.com/deicod/erm/internal/graphql"
	"github.com/deicod/erm/internal/graphql/dataloaders"
	"github.com/deicod/erm/internal/observability/metrics"
	"github.com/deicod/erm/internal/orm/gen"
)

// Options allows configuring resolver behaviour.
type Options struct {
	ORM       *gen.Client
	Collector metrics.Collector
}

// Resolver wires GraphQL resolvers into the executable schema.
type Resolver struct {
	ORM       *gen.Client
	collector metrics.Collector
}

// New creates a resolver root bound to the provided ORM client.
func New(orm *gen.Client) *Resolver {
	return NewWithOptions(Options{ORM: orm})
}

// NewWithOptions provides advanced resolver configuration.
func NewWithOptions(opts Options) *Resolver {
	collector := opts.Collector
	if collector == nil {
		collector = metrics.NoopCollector{}
	}
	return &Resolver{ORM: opts.ORM, collector: collector}
}

// WithLoaders attaches per-request dataloaders to the supplied context.
func (r *Resolver) WithLoaders(ctx context.Context) context.Context {
	if r == nil || r.ORM == nil {
		return ctx
	}
	loaders := dataloaders.New(r.ORM, r.collector)
	return dataloaders.ToContext(ctx, loaders)
}

func (r *Resolver) Mutation() graphql.MutationResolver { return &mutationResolver{r} }
func (r *Resolver) Query() graphql.QueryResolver       { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
