package orm

import (
	"testing"

	"github.com/deicod/erm/orm/runtime"
)

func BenchmarkBuildBulkInsertSQL(b *testing.B) {
	rows := [][]any{{1, "alice"}, {2, "bob"}, {3, "carol"}}
	spec := runtime.BulkInsertSpec{Table: "users", Columns: []string{"id", "name"}, Returning: []string{"id", "name"}, Rows: rows}
	for i := 0; i < b.N; i++ {
		if _, _, err := runtime.BuildBulkInsertSQL(spec); err != nil {
			b.Fatalf("build bulk insert: %v", err)
		}
	}
}

func BenchmarkBuildBulkUpdateSQL(b *testing.B) {
	rows := []runtime.BulkUpdateRow{
		{Primary: 1, Values: []any{"alice"}},
		{Primary: 2, Values: []any{"bob"}},
	}
	spec := runtime.BulkUpdateSpec{Table: "users", PrimaryColumn: "id", Columns: []string{"name"}, Returning: []string{"id", "name"}, Rows: rows}
	for i := 0; i < b.N; i++ {
		if _, _, err := runtime.BuildBulkUpdateSQL(spec); err != nil {
			b.Fatalf("build bulk update: %v", err)
		}
	}
}

func BenchmarkBuildBulkDeleteSQL(b *testing.B) {
	spec := runtime.BulkDeleteSpec{Table: "users", PrimaryColumn: "id", IDs: []any{1, 2, 3, 4, 5}}
	for i := 0; i < b.N; i++ {
		if _, _, err := runtime.BuildBulkDeleteSQL(spec); err != nil {
			b.Fatalf("build bulk delete: %v", err)
		}
	}
}
