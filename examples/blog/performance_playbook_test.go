package blog_test

import (
	"testing"

	"github.com/deicod/erm/examples/blog/schema"
	"github.com/deicod/erm/orm/dsl"
)

func TestPostQuerySpecDefaults(t *testing.T) {
	spec := (schema.Post{}).Query()
	if spec.DefaultLimit != 20 {
		t.Fatalf("expected default limit 20, got %d", spec.DefaultLimit)
	}
	if spec.MaxLimit != 200 {
		t.Fatalf("expected max limit 200, got %d", spec.MaxLimit)
	}
	assertPredicate(t, spec.Predicates, "workspace_id", dsl.OpEqual)
	assertPredicate(t, spec.Predicates, "author_id", dsl.OpEqual)
}

func assertPredicate(t *testing.T, predicates []dsl.Predicate, field string, op dsl.ComparisonOperator) {
	t.Helper()
	for _, predicate := range predicates {
		if predicate.Field == field {
			if predicate.Operator != op {
				t.Fatalf("unexpected operator for %s: %s", field, predicate.Operator)
			}
			return
		}
	}
	t.Fatalf("predicate for %s not found", field)
}
