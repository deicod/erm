package resolvers

import (
	"github.com/deicod/erm/internal/graphql"
	"github.com/deicod/erm/internal/orm/gen"
)

// Resolver wires GraphQL resolvers into the executable schema.
type Resolver struct {
	ORM *gen.Client
}

// New creates a resolver root bound to the provided ORM client.
func New(orm *gen.Client) *Resolver {
	return &Resolver{ORM: orm}
}

func (r *Resolver) Mutation() graphql.MutationResolver { return &mutationResolver{r} }
func (r *Resolver) Query() graphql.QueryResolver       { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
