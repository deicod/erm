package resolvers

import (
	"context"
	"errors"
	"testing"

	"github.com/deicod/erm/graphql"
	"github.com/deicod/erm/orm/gen"
)

// These tests document how projects can override the generated hooks without
// editing entities_gen.go. They wire custom callbacks into the resolver and
// assert the helpers invoked by CRUD resolvers respect them.

func TestBeforeReturnHookCanMutateRecord(t *testing.T) {
	resolver := NewWithOptions(Options{})
	user := &gen.User{Slug: "original"}
	called := false
	resolver.hooks.BeforeReturnUser = func(ctx context.Context, r *Resolver, record *gen.User) error {
		called = true
		record.Slug = "masked"
		return nil
	}

	if err := resolver.applyBeforeReturnUser(context.Background(), user); err != nil {
		t.Fatalf("applyBeforeReturnUser: %v", err)
	}
	if !called {
		t.Fatal("expected hook to be called")
	}
	if user.Slug != "masked" {
		t.Fatalf("expected slug to be mutated, got %q", user.Slug)
	}
}

func TestBeforeCreateHookCanRejectMutation(t *testing.T) {
	resolver := NewWithOptions(Options{})
	blocked := errors.New("blocked")
	resolver.hooks.BeforeCreateUser = func(ctx context.Context, r *Resolver, input graphql.CreateUserInput, model *gen.User) error {
		return blocked
	}

	if err := resolver.applyBeforeCreateUser(context.Background(), graphql.CreateUserInput{}, &gen.User{}); !errors.Is(err, blocked) {
		t.Fatalf("expected %v, got %v", blocked, err)
	}
}
