package blog_test

import (
	"testing"

	"github.com/deicod/erm/examples/blog/schema"
	"github.com/deicod/erm/internal/orm/dsl"
)

func TestBlogSchemaEdgeMetadata(t *testing.T) {
	post := schema.Post{}.Edges()
	workspaceEdge := edgeByName(post, "workspace")
	if workspaceEdge == nil {
		t.Fatalf("workspace edge not found on Post schema")
	}
	if workspaceEdge.Cascade.OnDelete != dsl.CascadeCascade {
		t.Fatalf("workspace OnDelete = %s, want %s", workspaceEdge.Cascade.OnDelete, dsl.CascadeCascade)
	}
	if got := len(workspaceEdge.PolymorphicTargets); got != 2 {
		t.Fatalf("expected workspace edge to declare two polymorphic targets, got %d", got)
	}

	commentEdges := schema.Comment{}.Edges()
	parent := edgeByName(commentEdges, "parent")
	if parent == nil {
		t.Fatalf("parent edge not found on Comment schema")
	}
	if parent.Cascade.OnDelete != dsl.CascadeSetNull {
		t.Fatalf("parent OnDelete = %s, want %s", parent.Cascade.OnDelete, dsl.CascadeSetNull)
	}
	replies := edgeByName(commentEdges, "replies")
	if replies == nil {
		t.Fatalf("replies edge not found on Comment schema")
	}
	if got := len(replies.PolymorphicTargets); got == 0 {
		t.Fatalf("expected replies edge to expose polymorphic targets")
	}
}

func edgeByName(edges []dsl.Edge, name string) *dsl.Edge {
	for i := range edges {
		if edges[i].Name == name {
			return &edges[i]
		}
	}
	return nil
}
