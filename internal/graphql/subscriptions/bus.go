package subscriptions

import (
	"context"
	"errors"
	"sync"
)

// ErrInvalidTopic indicates an empty subscription topic.
var ErrInvalidTopic = errors.New("subscriptions: topic must be non-empty")

// Broker exposes minimal publish/subscribe semantics for GraphQL subscriptions.
type Broker interface {
	Publish(ctx context.Context, topic string, payload any) error
	Subscribe(ctx context.Context, topic string) (<-chan any, func(), error)
}

// InMemoryBroker fan-outs events to in-process subscribers.
type InMemoryBroker struct {
	mu     sync.RWMutex
	subs   map[string]map[int]chan any
	nextID int
	buffer int
}

// NewInMemoryBroker constructs a broker with a small buffered channel per subscriber.
func NewInMemoryBroker() *InMemoryBroker {
	return &InMemoryBroker{subs: make(map[string]map[int]chan any), buffer: 1}
}

// WithBuffer overrides the per-subscriber buffer (default 1).
func (b *InMemoryBroker) WithBuffer(size int) *InMemoryBroker {
	if size <= 0 {
		size = 1
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buffer = size
	return b
}

// Publish delivers payload to all subscribers registered for topic.
func (b *InMemoryBroker) Publish(ctx context.Context, topic string, payload any) error {
	if topic == "" {
		return ErrInvalidTopic
	}
	if ctx == nil {
		ctx = context.Background()
	}
	b.mu.RLock()
	subscribers, ok := b.subs[topic]
	if !ok {
		b.mu.RUnlock()
		return nil
	}
	for _, ch := range subscribers {
		if err := ctx.Err(); err != nil {
			b.mu.RUnlock()
			return err
		}
		select {
		case ch <- payload:
		default:
		}
	}
	b.mu.RUnlock()
	return nil
}

// Subscribe registers a subscriber for topic. Cancel releases resources and closes the channel.
func (b *InMemoryBroker) Subscribe(ctx context.Context, topic string) (<-chan any, func(), error) {
	if topic == "" {
		return nil, nil, ErrInvalidTopic
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ch := make(chan any, b.buffer)
	b.mu.Lock()
	if _, ok := b.subs[topic]; !ok {
		b.subs[topic] = make(map[int]chan any)
	}
	id := b.nextID
	b.nextID++
	b.subs[topic][id] = ch
	b.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			b.mu.Lock()
			if subs := b.subs[topic]; subs != nil {
				if subscriber, ok := subs[id]; ok {
					delete(subs, id)
					close(subscriber)
				}
				if len(subs) == 0 {
					delete(b.subs, topic)
				}
			}
			b.mu.Unlock()
		})
	}

	go func() {
		<-ctx.Done()
		cancel()
	}()

	return ch, cancel, nil
}
