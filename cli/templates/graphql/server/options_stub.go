package server

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

type Options struct {
	Subscriptions SubscriptionOptions
}

type SubscriptionOptions struct {
	Enabled    bool
	Transports SubscriptionTransports
}

type SubscriptionTransports struct {
	Websocket bool
	GraphQLWS bool
}

func normaliseOptions(opts Options) Options { return opts }

func NewExecutableSchema(Options) graphql.ExecutableSchema { return executableSchemaStub{} }

func WithLoaders(ctx context.Context, _ Options) context.Context { return ctx }

type executableSchemaStub struct{}

func (executableSchemaStub) Schema() *ast.Schema { return nil }

func (executableSchemaStub) Complexity(context.Context, string, string, int, map[string]any) (int, bool) {
	return 0, false
}

func (executableSchemaStub) Exec(context.Context) graphql.ResponseHandler {
	return func(context.Context) *graphql.Response { return nil }
}
