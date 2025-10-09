package generator

import (
	"strings"
	"testing"

	"github.com/deicod/erm/internal/orm/dsl"
)

func TestRenderInitialMigration_OneToManyEdges(t *testing.T) {
	entities := []Entity{
		{
			Name:   "User",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
			Edges: []dsl.Edge{
				dsl.ToMany("posts", "Post").Ref("author_id"),
			},
		},
		{
			Name:   "Post",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
		},
	}

	sql := renderInitialMigration(entities, extensionFlags{})

	if !strings.Contains(sql, "author_id uuid NOT NULL") {
		t.Fatalf("expected posts table to include author_id column, got:\n%s", sql)
	}
	if !strings.Contains(sql, "CONSTRAINT fk_posts_author_id FOREIGN KEY (author_id) REFERENCES users (id)") {
		t.Fatalf("expected posts table to include foreign key constraint, got:\n%s", sql)
	}
}

func TestRenderInitialMigration_ToManyDerivesRefColumn(t *testing.T) {
	entities := []Entity{
		{
			Name:   "User",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
			Edges: []dsl.Edge{
				dsl.ToMany("pets", "Pet"),
			},
		},
		{
			Name:   "Pet",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
		},
	}

	sql := renderInitialMigration(entities, extensionFlags{})

	if !strings.Contains(sql, "user_id uuid NOT NULL") {
		t.Fatalf("expected pets table to include derived user_id column, got:\n%s", sql)
	}
	if !strings.Contains(sql, "CONSTRAINT fk_pets_user_id FOREIGN KEY (user_id) REFERENCES users (id)") {
		t.Fatalf("expected pets table to include foreign key constraint, got:\n%s", sql)
	}
}

func TestRenderInitialMigration_ManyToManyEdges(t *testing.T) {
	entities := []Entity{
		{
			Name:   "User",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
			Edges: []dsl.Edge{
				dsl.ManyToMany("groups", "Group"),
			},
		},
		{
			Name:   "Group",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
			Edges: []dsl.Edge{
				dsl.ManyToMany("members", "User").ThroughTable("memberships"),
			},
		},
	}

	sql := renderInitialMigration(entities, extensionFlags{})

	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS groups_users") {
		t.Fatalf("expected default join table groups_users to be created, got:\n%s", sql)
	}
	if !strings.Contains(sql, "CONSTRAINT fk_groups_users_user_id FOREIGN KEY (user_id) REFERENCES users (id)") {
		t.Fatalf("expected groups_users table to reference users(id), got:\n%s", sql)
	}
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS memberships") {
		t.Fatalf("expected custom through table memberships to be created, got:\n%s", sql)
	}
	if !strings.Contains(sql, "CONSTRAINT fk_memberships_group_id FOREIGN KEY (group_id) REFERENCES groups (id)") {
		t.Fatalf("expected memberships table to reference groups(id), got:\n%s", sql)
	}
}

func TestFieldSQLType_PostgresFamilies(t *testing.T) {
	tests := []struct {
		name  string
		field dsl.Field
		want  string
	}{
		{"text", dsl.Text("title"), "text"},
		{"varchar", dsl.VarChar("code", 12), "varchar(12)"},
		{"decimal", dsl.Decimal("price", 10, 2), "decimal(10,2)"},
		{"timestamp", dsl.TimestampTZ("created_at"), "timestamptz"},
		{"identity", dsl.BigIntIdentity("id", dsl.IdentityAlways), "bigint GENERATED ALWAYS AS IDENTITY"},
		{"jsonb", dsl.JSONB("payload"), "jsonb"},
		{"bit", dsl.Bit("mask", 8), "bit(8)"},
		{"array", dsl.Array("tags", dsl.TypeText), "text[]"},
		{"vector", dsl.Vector("embedding", 3), "vector(3)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fieldSQLType(tt.field)
			if got != tt.want {
				t.Fatalf("fieldSQLType(%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
