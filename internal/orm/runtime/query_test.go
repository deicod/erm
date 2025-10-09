package runtime

import "testing"

func TestBuildSelectSQL(t *testing.T) {
	spec := SelectSpec{
		Table:   "users",
		Columns: []string{"id", "email"},
		Predicates: []Predicate{
			{Column: "email", Operator: OpILike, Value: "%example.com"},
		},
		Orders: []Order{
			{Column: "created_at", Direction: SortDesc},
		},
		Limit:  25,
		Offset: 10,
	}

	sql, args := BuildSelectSQL(spec)
	expected := "SELECT id, email FROM users WHERE email ILIKE $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
	if sql != expected {
		t.Fatalf("unexpected SQL:\n got: %s\nwant: %s", sql, expected)
	}
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(args))
	}
	if args[0] != "%example.com" || args[1] != 25 || args[2] != 10 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildSelectSQLDefaults(t *testing.T) {
	spec := SelectSpec{Table: "users"}

	sql, args := BuildSelectSQL(spec)
	if sql != "SELECT * FROM users" {
		t.Fatalf("unexpected SQL: %s", sql)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args, got %d", len(args))
	}
}

func TestBuildAggregateSQL(t *testing.T) {
	spec := AggregateSpec{
		Table: "users",
		Predicates: []Predicate{
			{Column: "created_at", Operator: OpGTE, Value: "2024-01-01"},
		},
		Aggregate: Aggregate{Func: AggCount, Column: "id"},
	}

	sql, args := BuildAggregateSQL(spec)
	expected := "SELECT COUNT(id) FROM users WHERE created_at >= $1"
	if sql != expected {
		t.Fatalf("unexpected SQL:\n got: %s\nwant: %s", sql, expected)
	}
	if len(args) != 1 || args[0] != "2024-01-01" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildAggregateSQLDefaultColumn(t *testing.T) {
	spec := AggregateSpec{
		Table:     "users",
		Aggregate: Aggregate{Func: AggCount},
	}

	sql, args := BuildAggregateSQL(spec)
	if sql != "SELECT COUNT(*) FROM users" {
		t.Fatalf("unexpected SQL: %s", sql)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args, got %d", len(args))
	}
}
