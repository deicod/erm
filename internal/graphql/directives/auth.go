package directives

import (
	"context"
	"fmt"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

type ClaimsKey struct{}

type Claims struct {
	Subject string
	Email string
	Username string
	Roles []string
}

func FromContext(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(ClaimsKey{}).(Claims)
	return c, ok
}

func RequireRoles(roles []string) func(ctx context.Context, obj interface{}, next func(ctx context.Context) (res interface{}, err error)) (interface{}, error) {
	return func(ctx context.Context, obj interface{}, next func(ctx context.Context) (res interface{}, err error)) (interface{}, error) {
		claims, ok := FromContext(ctx)
		if !ok { return nil, gqlerror.Errorf("unauthorized") }
		// naive role check
		set := map[string]struct{}{}; for _, r := range claims.Roles { set[r] = struct{}{} }
		for _, r := range roles { if _, ok := set[r]; !ok { return nil, gqlerror.Errorf("forbidden: missing role %s", r) } }
		return next(ctx)
	}
}

func RequireAuth() func(ctx context.Context, obj interface{}, next func(ctx context.Context) (res interface{}, err error)) (interface{}, error) {
	return func(ctx context.Context, obj interface{}, next func(ctx context.Context) (res interface{}, err error)) (interface{}, error) {
		_, ok := FromContext(ctx)
		if !ok { return nil, fmt.Errorf("unauthorized") }
		return next(ctx)
	}
}
