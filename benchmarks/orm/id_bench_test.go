package orm

import (
	"testing"

	"github.com/deicod/erm/orm/id"
)

func BenchmarkUUIDv7(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := id.NewV7(); err != nil {
			b.Fatalf("newv7: %v", err)
		}
	}
}
