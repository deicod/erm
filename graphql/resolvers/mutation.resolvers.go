package resolvers

import "context"

func (r *mutationResolver) Noop(ctx context.Context) (*bool, error) {
	value := true
	return &value, nil
}
