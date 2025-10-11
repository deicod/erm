package directives

import (
	"context"
	"fmt"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/deicod/erm/oidc"
)

func RequireRoles(roles []string) func(ctx context.Context, obj interface{}, next func(ctx context.Context) (res interface{}, err error)) (interface{}, error) {
	return func(ctx context.Context, obj interface{}, next func(ctx context.Context) (res interface{}, err error)) (interface{}, error) {
		claims, ok := oidc.FromContext(ctx)
		if !ok {
			return nil, gqlerror.Errorf("unauthorized")
		}
		roleSet := make(map[string]struct{}, len(claims.Roles))
		for _, r := range claims.Roles {
			roleSet[r] = struct{}{}
		}
		for _, required := range roles {
			if _, ok := roleSet[required]; !ok {
				return nil, gqlerror.Errorf("forbidden: missing role %s", required)
			}
		}
		return next(ctx)
	}
}

func RequireAuth() func(ctx context.Context, obj interface{}, next func(ctx context.Context) (res interface{}, err error)) (interface{}, error) {
	return func(ctx context.Context, obj interface{}, next func(ctx context.Context) (res interface{}, err error)) (interface{}, error) {
		if _, ok := oidc.FromContext(ctx); !ok {
			return nil, fmt.Errorf("unauthorized")
		}
		return next(ctx)
	}
}
