package resolvers

import "context"

// Noop provides a baseline subscription field so transports remain valid when
// consumers have not yet opted into entity-triggered subscriptions.
func (r *subscriptionResolver) Noop(ctx context.Context) (<-chan *bool, error) {
	ch := make(chan *bool)
	close(ch)
	return ch, nil
}
