package graphql

import (
	"context"

	gql "github.com/99designs/gqlgen/graphql"
)

type ResolverRoot interface{}

type MutationResolver interface{}

type QueryResolver interface{}

type SubscriptionResolver interface{}

type DirectiveRoot struct {
	Auth func(ctx context.Context, obj any, next gql.Resolver, roles []string) (any, error)
}

type Config struct {
	Resolvers  ResolverRoot
	Directives DirectiveRoot
}

type executionContext struct{}

func NewExecutableSchema(Config) gql.ExecutableSchema { return nil }
