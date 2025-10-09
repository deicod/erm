package blog_test

import (
	"testing"

	"github.com/deicod/erm/examples/blog/schema"
)

func TestCommentThreadingEdges(t *testing.T) {
	var parentOptional bool
	var repliesEdge bool
	for _, edge := range (schema.Comment{}).Edges() {
		switch edge.Name {
		case "parent":
			if edge.Column != "parent_id" {
				t.Fatalf("expected parent edge to target parent_id column")
			}
			parentOptional = edge.Nullable
		case "replies":
			repliesEdge = true
			if edge.RefName != "parent" {
				t.Fatalf("expected replies edge to reference parent")
			}
		}
	}
	if !parentOptional {
		t.Fatalf("expected parent edge to be optional for orphan comments")
	}
	if !repliesEdge {
		t.Fatalf("expected replies edge for nested error walkthrough")
	}
}
