package resolvers

import (
	"context"
	"errors"
	"strings"

	"github.com/deicod/erm/graphql/subscriptions"
)

// SubscriptionTrigger identifies the CRUD action emitted for an entity change.
type SubscriptionTrigger string

const (
	SubscriptionTriggerCreated SubscriptionTrigger = "created"
	SubscriptionTriggerUpdated SubscriptionTrigger = "updated"
	SubscriptionTriggerDeleted SubscriptionTrigger = "deleted"
)

// ErrSubscriptionsDisabled signals that subscription resolvers are unavailable.
var ErrSubscriptionsDisabled = errors.New("graphql subscriptions disabled")

func publishSubscriptionEvent(ctx context.Context, broker subscriptions.Broker, entity string, trigger SubscriptionTrigger, payload any) {
	if broker == nil || entity == "" {
		return
	}
	_ = broker.Publish(ctx, subscriptionTopic(entity, trigger), payload)
}

// Publish forwards an entity change to the provided broker using the canonical topic naming scheme.
func Publish(ctx context.Context, broker subscriptions.Broker, entity string, trigger SubscriptionTrigger, payload any) {
	publishSubscriptionEvent(ctx, broker, entity, trigger, payload)
}

func subscribeToEntity(ctx context.Context, broker subscriptions.Broker, entity string, trigger SubscriptionTrigger) (<-chan any, func(), error) {
	if broker == nil {
		return nil, nil, ErrSubscriptionsDisabled
	}
	stream, cancel, err := broker.Subscribe(ctx, subscriptionTopic(entity, trigger))
	if err != nil {
		return nil, nil, err
	}
	return stream, cancel, nil
}

func subscriptionTopic(entity string, trigger SubscriptionTrigger) string {
	base := strings.ToLower(entity)
	if base == "" {
		base = "entity"
	}
	return base + ":" + string(trigger)
}

// Topic returns the canonical topic key for an entity/trigger combination.
func Topic(entity string, trigger SubscriptionTrigger) string {
	return subscriptionTopic(entity, trigger)
}
