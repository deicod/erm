package runtime

import "testing"

func TestBuildBulkInsertSQL(t *testing.T) {
	sql, args, err := BuildBulkInsertSQL(BulkInsertSpec{
		Table:     "users",
		Columns:   []string{"id", "name"},
		Returning: []string{"id", "name"},
		Rows: [][]any{
			{1, "a"},
			{2, "b"},
		},
	})
	if err != nil {
		t.Fatalf("build insert: %v", err)
	}
	wantSQL := "INSERT INTO users (id, name) VALUES ($1, $2), ($3, $4) RETURNING id, name"
	if sql != wantSQL {
		t.Fatalf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 4 || args[0] != 1 || args[3] != "b" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildBulkUpdateSQL(t *testing.T) {
	sql, args, err := BuildBulkUpdateSQL(BulkUpdateSpec{
		Table:         "users",
		PrimaryColumn: "id",
		Columns:       []string{"name"},
		Returning:     []string{"id", "name"},
		Rows: []BulkUpdateRow{
			{Primary: 1, Values: []any{"alice"}},
			{Primary: 2, Values: []any{"bob"}},
		},
	})
	if err != nil {
		t.Fatalf("build update: %v", err)
	}
	wantSQL := "WITH data(id, name) AS (VALUES ($1, $2), ($3, $4)) UPDATE users AS t SET name = data.name FROM data WHERE t.id = data.id RETURNING id, name"
	if sql != wantSQL {
		t.Fatalf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 4 || args[0] != 1 || args[3] != "bob" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildBulkDeleteSQL(t *testing.T) {
	sql, args, err := BuildBulkDeleteSQL(BulkDeleteSpec{
		Table:         "users",
		PrimaryColumn: "id",
		IDs:           []any{1, 2, 3},
	})
	if err != nil {
		t.Fatalf("build delete: %v", err)
	}
	wantSQL := "DELETE FROM users WHERE id IN ($1, $2, $3)"
	if sql != wantSQL {
		t.Fatalf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 3 || args[0] != 1 || args[2] != 3 {
		t.Fatalf("unexpected args: %#v", args)
	}
}
