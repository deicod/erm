package graphql

import (
	"context"
	"time"

	gql "github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

func (ec *executionContext) unmarshalInputTime(ctx context.Context, v any) (time.Time, error) {
	return gql.UnmarshalTime(v)
}

func (ec *executionContext) _Time(ctx context.Context, sel ast.SelectionSet, v *time.Time) gql.Marshaler {
	if v == nil {
		return gql.Null
	}
	return gql.MarshalTime(*v)
}

func (ec *executionContext) unmarshalInputTimestamptz(ctx context.Context, v any) (time.Time, error) {
	return gql.UnmarshalTime(v)
}

func (ec *executionContext) _Timestamptz(ctx context.Context, sel ast.SelectionSet, v *time.Time) gql.Marshaler {
	if v == nil {
		return gql.Null
	}
	return gql.MarshalTime(*v)
}
