package graphql

import (
	"context"

	gql "github.com/99designs/gqlgen/graphql"
)

type ResolverRoot any

type MutationResolver any

type QueryResolver any

type SubscriptionResolver any

type DirectiveRoot struct {
	Auth func(ctx context.Context, obj any, next gql.Resolver, roles []string) (any, error)
}

type Config struct {
	Resolvers  ResolverRoot
	Directives DirectiveRoot
}

type executionContext struct{}

func NewExecutableSchema(Config) gql.ExecutableSchema { return nil }
