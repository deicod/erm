package blog_test

import (
	"testing"

	"github.com/deicod/erm/examples/blog/schema"
)

func TestWorkspaceSlugConstraints(t *testing.T) {
	var slugFieldFound bool
	for _, field := range (schema.Workspace{}).Fields() {
		if field.Name == "slug" {
			slugFieldFound = true
			if !field.IsUnique {
				t.Fatalf("expected slug field to be unique")
			}
			if field.Annotations["notEmpty"] != true {
				t.Fatalf("expected slug field to be marked not empty")
			}
			break
		}
	}
	if !slugFieldFound {
		t.Fatalf("slug field not defined on workspace")
	}
}

func TestMembershipCompositeIndex(t *testing.T) {
	indexes := (schema.Membership{}).Indexes()
	for _, idx := range indexes {
		if len(idx.Columns) == 2 && idx.Columns[0] == "workspace_id" && idx.Columns[1] == "user_id" {
			if !idx.IsUnique {
				t.Fatalf("workspace_id/user_id index should be unique")
			}
			return
		}
	}
	t.Fatalf("expected composite index on workspace_id + user_id")
}
