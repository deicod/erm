package generator

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deicod/erm/internal/orm/dsl"
)

func TestWriteORMClients_EdgeHelpers(t *testing.T) {
	entities := []Entity{
		{
			Name: "Post",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.UUIDv7("author_id"),
				dsl.String("title"),
			},
			Edges: []dsl.Edge{
				dsl.ToOne("author", "User").Field("author_id").Inverse("posts"),
			},
		},
		{
			Name: "User",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.String("name"),
			},
		},
		{
			Name: "Group",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.String("name"),
			},
			Edges: []dsl.Edge{
				dsl.ManyToMany("members", "User").Inverse("groups"),
			},
		},
	}

	synthesizeInverseEdges(entities)
	for i := range entities {
		ensureDefaultQuery(&entities[i])
	}

	root := t.TempDir()
	if err := writeModels(root, entities); err != nil {
		t.Fatalf("writeModels: %v", err)
	}
	if err := writeClients(root, entities); err != nil {
		t.Fatalf("writeClients: %v", err)
	}

	clientPath := filepath.Join(root, "internal", "orm", "gen", "client_gen.go")
	clientSrc, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("read client: %v", err)
	}

	content := string(clientSrc)
	mustContain(t, content, "var ValidationRegistry = validation.NewRegistry()")
	mustContain(t, content, "const userInsertQuery = `INSERT INTO users")
	mustContain(t, content, "const userSelectQuery = `SELECT id")
	mustContain(t, content, "const userListQuery = `SELECT id")
	mustContain(t, content, "const userUpdateQuery = `UPDATE users SET")
	mustContain(t, content, "const userCountQuery = `SELECT COUNT(*) FROM users`")
	mustContain(t, content, "const userDeleteQuery = `DELETE FROM users WHERE id = $1`")
	mustContain(t, content, "ValidationRegistry.Validate(ctx, \"User\", validation.OpCreate, userValidationRecord(input), input)")
	mustContain(t, content, "ValidationRegistry.Validate(ctx, \"User\", validation.OpUpdate, userValidationRecord(input), input)")
	mustContain(t, content, "func userValidationRecord(input *User) validation.Record {")
	mustContain(t, content, "const postAuthorRelationQuery = `SELECT id, name FROM users WHERE id IN (%s)`")
	mustContain(t, content, "func (c *PostClient) LoadAuthor(")
	mustContain(t, content, "const userPostsRelationQuery = `SELECT id, author_id, title FROM posts WHERE author_id IN (%s)`")
	mustContain(t, content, "func (c *UserClient) LoadPosts(")
	mustContain(t, content, "const userGroupsRelationQuery = `SELECT id, name, jt.user_id FROM groups AS t JOIN groups_users AS jt ON t.id = jt.group_id WHERE jt.user_id IN (%s)`")
	mustContain(t, content, "func (c *UserClient) LoadGroups(")
	mustContain(t, content, "type PostQuery struct {")
	mustContain(t, content, "func (c *PostClient) Query() *PostQuery")
	mustContain(t, content, "func (q *PostQuery) WhereIDEq(")
	mustContain(t, content, "func (q *PostQuery) Count(")

	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, clientPath, clientSrc, parser.AllErrors); err != nil {
		t.Fatalf("parse client: %v", err)
	}

	modelsPath := filepath.Join(root, "internal", "orm", "gen", "models_gen.go")
	modelsSrc, err := os.ReadFile(modelsPath)
	if err != nil {
		t.Fatalf("read models: %v", err)
	}
	models := string(modelsSrc)
	mustContain(t, models, "json:\"edges,omitempty\"")
	mustContain(t, models, "type PostEdges struct {")
	mustContain(t, models, "func ensurePostEdges(")
	mustContain(t, models, "func (m *Post) SetAuthor(")
	if _, err := parser.ParseFile(fset, modelsPath, modelsSrc, parser.AllErrors); err != nil {
		t.Fatalf("parse models: %v", err)
	}
}

func TestWriteORMClients_ComputedFields(t *testing.T) {
	entities := []Entity{
		{
			Name: "User",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.String("first_name"),
				dsl.String("last_name"),
				dsl.Text("full_name").Computed(dsl.Computed(dsl.Expression("first_name || ' ' || last_name", "first_name", "last_name"))),
			},
		},
	}

	for i := range entities {
		ensureDefaultQuery(&entities[i])
	}

	root := t.TempDir()
	if err := writeClients(root, entities); err != nil {
		t.Fatalf("writeClients: %v", err)
	}

	clientPath := filepath.Join(root, "internal", "orm", "gen", "client_gen.go")
	clientSrc, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("read client: %v", err)
	}

	content := string(clientSrc)
	mustContain(t, content, "INSERT INTO users (id, first_name, last_name) VALUES ($1, $2, $3) RETURNING id, first_name, last_name, full_name")
	mustContain(t, content, "if !runtime.IsZeroValue(input.FullName)")
	mustContain(t, content, "fmt.Errorf(\"User.FullName is computed and cannot be set\")")
	mustContain(t, content, "row := []any{input.ID, input.FirstName, input.LastName}")
}

func TestWriteRegistry_EdgeMetadata(t *testing.T) {
	entities := []Entity{
		{
			Name:   "Workspace",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
			Edges: []dsl.Edge{
				dsl.ToMany("members", "User").Ref("workspace").OnDeleteCascade().OnUpdateRestrict().Polymorphic(
					dsl.PolymorphicTarget("User", "role = 'admin'"),
					dsl.PolymorphicTarget("User", "role = 'member'"),
				),
			},
		},
		{
			Name:   "User",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary(), dsl.UUIDv7("workspace_id").Optional()},
			Edges: []dsl.Edge{
				dsl.ToOne("workspace", "Workspace").Field("workspace_id").OnDeleteCascade().OnUpdateRestrict(),
			},
		},
	}

	root := t.TempDir()
	if err := writeRegistry(root, entities); err != nil {
		t.Fatalf("writeRegistry: %v", err)
	}

	path := filepath.Join(root, "internal", "orm", "gen", "registry_gen.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	out := string(content)
	mustContain(t, out, "PolymorphicTargets: []runtime.EdgeTargetSpec{{Entity: \"User\", Condition: \"role = 'admin'\"}, {Entity: \"User\", Condition: \"role = 'member'\"}}")
	mustContain(t, out, "Cascade: runtime.CascadeSpec{OnDelete: runtime.CascadeCascade, OnUpdate: runtime.CascadeRestrict}")
}

func mustContain(t *testing.T, content, needle string) {
	t.Helper()
	if !strings.Contains(content, needle) {
		t.Fatalf("expected generated content to contain %q\nactual: %s", needle, content)
	}
}

func TestDefaultGoType_PostgresFamilies(t *testing.T) {
	tests := []struct {
		name  string
		field dsl.Field
		want  string
	}{
		{"uuid", dsl.UUID("id"), "string"},
		{"text", dsl.Text("title"), "string"},
		{"bigint", dsl.BigInt("counter"), "int64"},
		{"decimal", dsl.Decimal("price", 10, 2), "string"},
		{"real", dsl.Real("ratio"), "float32"},
		{"timestamp", dsl.TimestampTZ("created_at"), "time.Time"},
		{"jsonb", dsl.JSONB("payload"), "json.RawMessage"},
		{"vector", dsl.Vector("embedding", 3), "[]float32"},
		{"array_text", dsl.Array("tags", dsl.TypeText), "[]string"},
		{"array_int", dsl.Array("scores", dsl.TypeInteger), "[]int32"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultGoType(tt.field); got != tt.want {
				t.Fatalf("defaultGoType(%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
