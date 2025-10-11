package resolvers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/deicod/erm/graphql"
	"github.com/deicod/erm/graphql/subscriptions"
)

func TestSubscriptionReceivesCreateEvent(t *testing.T) {
	broker := subscriptions.NewInMemoryBroker().WithBuffer(2)
	resolver := NewWithOptions(Options{Subscriptions: broker})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	stream, err := (&subscriptionResolver{resolver}).UserCreated(ctx)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	user := &graphql.User{ID: "user:1"}
	publishSubscriptionEvent(ctx, resolver.subscriptionBroker(), "User", SubscriptionTriggerCreated, user)

	select {
	case msg := <-stream:
		if msg == nil || msg.ID != user.ID {
			t.Fatalf("unexpected payload: %#v", msg)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for subscription event")
	}
}

func TestSubscriptionReceivesDeleteEvent(t *testing.T) {
	broker := subscriptions.NewInMemoryBroker()
	resolver := NewWithOptions(Options{Subscriptions: broker})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	stream, err := (&subscriptionResolver{resolver}).UserDeleted(ctx)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	publishSubscriptionEvent(ctx, resolver.subscriptionBroker(), "User", SubscriptionTriggerDeleted, "user:99")

	select {
	case msg := <-stream:
		if msg != "user:99" {
			t.Fatalf("unexpected payload %q", msg)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for delete event")
	}
}

func TestSubscriptionDisabled(t *testing.T) {
	resolver := NewWithOptions(Options{})
	_, err := (&subscriptionResolver{resolver}).UserCreated(context.Background())
	if !errors.Is(err, ErrSubscriptionsDisabled) {
		t.Fatalf("expected ErrSubscriptionsDisabled, got %v", err)
	}
}
