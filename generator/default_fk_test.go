package generator

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultForeignKeyNamingFixture(t *testing.T) {
	root := filepath.Join("testdata", "default_fk")

	entities, err := loadEntities(root)
	if err != nil {
		t.Fatalf("loadEntities: %v", err)
	}

	pet := findEntity(entities, "Pet")
	if len(pet.Edges) != 1 {
		t.Fatalf("expected synthesized pet edge, got %d", len(pet.Edges))
	}
	owner := pet.Edges[0]
	if owner.Column != "user_id" {
		t.Fatalf("expected synthesized edge column 'user_id', got %q", owner.Column)
	}
	if owner.RefName != "user_id" {
		t.Fatalf("expected synthesized edge ref name 'user_id', got %q", owner.RefName)
	}

	sql := renderInitialMigration(entities, extensionFlags{})
	if !strings.Contains(sql, "user_id uuid NOT NULL") {
		t.Fatalf("expected generated SQL to include derived user_id column, got:\n%s", sql)
	}
	if !strings.Contains(sql, "CONSTRAINT fk_pets_user_id FOREIGN KEY (user_id) REFERENCES users (id)") {
		t.Fatalf("expected generated SQL to include user_id foreign key constraint, got:\n%s", sql)
	}
}
