package cache

import "context"

// Store defines the caching contract consumed by generated ORM clients.
type Store interface {
	// Get retrieves a cached value by key. The boolean result indicates presence.
	Get(ctx context.Context, key string) (any, bool, error)
	// Set associates a value with the provided key.
	Set(ctx context.Context, key string, value any) error
	// Delete removes the value for the provided key.
	Delete(ctx context.Context, key string) error
}

// nopStore is a Store implementation that never caches values.
type nopStore struct{}

// Nop returns a Store implementation that disables caching while preserving the
// expected interface contracts.
func Nop() Store {
	return nopStore{}
}

func (nopStore) Get(context.Context, string) (any, bool, error) { return nil, false, nil }

func (nopStore) Set(context.Context, string, any) error { return nil }

func (nopStore) Delete(context.Context, string) error { return nil }
