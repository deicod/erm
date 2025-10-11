package cache

import (
	"context"
	"testing"
)

func TestNopStore(t *testing.T) {
	store := Nop()
	if store == nil {
		t.Fatalf("expected nop store")
	}
	ctx := context.Background()
	if _, ok, err := store.Get(ctx, "key"); err != nil || ok {
		t.Fatalf("expected miss with nil error, got ok=%v err=%v", ok, err)
	}
	if err := store.Set(ctx, "key", 1); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := store.Delete(ctx, "key"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
