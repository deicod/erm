package dataloaders

import (
	"context"
	"sync"
	"time"

	"github.com/deicod/erm/internal/observability/metrics"
	"github.com/deicod/erm/internal/orm/gen"
)

// Loaders aggregates entity-specific dataloaders.
type Loaders struct {
	entries map[string]any
}

// New constructs per-request dataloaders backed by the ORM client.
func New(orm *gen.Client, collector metrics.Collector) *Loaders {
	if collector == nil {
		collector = metrics.NoopCollector{}
	}
	loaders := &Loaders{entries: make(map[string]any)}
	if orm == nil {
		return loaders
	}
	configureEntityLoaders(loaders, orm, collector)
	return loaders
}

func (l *Loaders) register(name string, loader any) {
	if l == nil {
		return
	}
	if l.entries == nil {
		l.entries = make(map[string]any)
	}
	l.entries[name] = loader
}

func (l *Loaders) get(name string) any {
	if l == nil {
		return nil
	}
	if l.entries == nil {
		return nil
	}
	return l.entries[name]
}

// EntityLoader caches entities fetched by key with optional batching instrumentation.
type EntityLoader[K comparable, V any] struct {
	name      string
	fetch     func(context.Context, []K) (map[K]V, error)
	collector metrics.Collector

	mu    sync.RWMutex
	cache map[K]V
}

func newEntityLoader[K comparable, V any](name string, collector metrics.Collector, fetch func(context.Context, []K) (map[K]V, error)) *EntityLoader[K, V] {
	if collector == nil {
		collector = metrics.NoopCollector{}
	}
	return &EntityLoader[K, V]{
		name:      name,
		fetch:     fetch,
		collector: collector,
		cache:     make(map[K]V),
	}
}

// Load resolves an entity by key, caching successful lookups.
func (l *EntityLoader[K, V]) Load(ctx context.Context, key K) (V, error) {
	l.mu.RLock()
	if val, ok := l.cache[key]; ok {
		l.mu.RUnlock()
		return val, nil
	}
	l.mu.RUnlock()

	start := time.Now()
	values, err := l.fetch(ctx, []K{key})
	duration := time.Since(start)
	if err != nil {
		var zero V
		return zero, err
	}
	l.collector.RecordDataloaderBatch(l.name, len(values), duration)

	var zero V
	val, ok := values[key]
	if !ok {
		return zero, nil
	}

	l.mu.Lock()
	l.cache[key] = val
	l.mu.Unlock()

	return val, nil
}

// Prime seeds the cache with known results to avoid duplicate fetches.
func (l *EntityLoader[K, V]) Prime(key K, value V) {
	l.mu.Lock()
	l.cache[key] = value
	l.mu.Unlock()
}

// NewEntityLoader exposes loader construction for testing and advanced customization.
func NewEntityLoader[K comparable, V any](name string, collector metrics.Collector, fetch func(context.Context, []K) (map[K]V, error)) *EntityLoader[K, V] {
	return newEntityLoader(name, collector, fetch)
}

// contextKey isolates loader storage on context.
type contextKey struct{}

// ToContext attaches loaders to the provided context for downstream resolvers.
func ToContext(ctx context.Context, loaders *Loaders) context.Context {
	if loaders == nil {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, loaders)
}

// FromContext extracts loaders from the context, if present.
func FromContext(ctx context.Context) *Loaders {
	if ctx == nil {
		return nil
	}
	if loaders, ok := ctx.Value(contextKey{}).(*Loaders); ok {
		return loaders
	}
	return nil
}
