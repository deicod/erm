package dataloaders_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deicod/erm/internal/graphql/dataloaders"
)

type testCollector struct {
	batches int32
	total   int32
}

func (c *testCollector) RecordDataloaderBatch(_ string, size int, _ time.Duration) {
	atomic.AddInt32(&c.batches, 1)
	atomic.AddInt32(&c.total, int32(size))
}

func (testCollector) RecordQuery(string, string, time.Duration, error) {}

func TestEntityLoaderCachesResults(t *testing.T) {
	var fetchCalls int32
	collector := &testCollector{}
	loader := dataloaders.NewEntityLoader("test", collector, func(_ context.Context, keys []string) (map[string]string, error) {
		atomic.AddInt32(&fetchCalls, 1)
		out := make(map[string]string, len(keys))
		for _, key := range keys {
			out[key] = key + "-value"
		}
		return out, nil
	})

	ctx := context.Background()
	value, err := loader.Load(ctx, "alpha")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if value != "alpha-value" {
		t.Fatalf("unexpected value: %q", value)
	}

	// second load should hit cache
	value, err = loader.Load(ctx, "alpha")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if value != "alpha-value" {
		t.Fatalf("unexpected cached value: %q", value)
	}

	if calls := atomic.LoadInt32(&fetchCalls); calls != 1 {
		t.Fatalf("expected single fetch call, got %d", calls)
	}
	if batches := atomic.LoadInt32(&collector.batches); batches != 1 {
		t.Fatalf("expected metrics batch count 1, got %d", batches)
	}
	if total := atomic.LoadInt32(&collector.total); total != 1 {
		t.Fatalf("expected metrics total 1, got %d", total)
	}
}

func TestEntityLoaderPrime(t *testing.T) {
	collector := &testCollector{}
	loader := dataloaders.NewEntityLoader("test", collector, func(_ context.Context, keys []string) (map[string]string, error) {
		t.Fatalf("fetch should not execute for primed key, got keys %v", keys)
		return nil, nil
	})
	loader.Prime("beta", "primed")
	value, err := loader.Load(context.Background(), "beta")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if value != "primed" {
		t.Fatalf("unexpected primed value: %q", value)
	}
}
