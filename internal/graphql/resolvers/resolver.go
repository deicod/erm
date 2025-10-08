package resolvers

import "context"

type Resolver struct{}

func (r *Resolver) Health(ctx context.Context) (string, error) { return "ok", nil }
