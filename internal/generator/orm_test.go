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
	mustContain(t, content, "const postAuthorRelationQuery = `SELECT id, name FROM users WHERE id IN (%s)`")
	mustContain(t, content, "func (c *PostClient) LoadAuthor(")
	mustContain(t, content, "const userPostsRelationQuery = `SELECT id, author_id, title FROM posts WHERE author_id IN (%s)`")
	mustContain(t, content, "func (c *UserClient) LoadPosts(")
	mustContain(t, content, "const userGroupsRelationQuery = `SELECT id, name, jt.user_id FROM groups AS t JOIN users_groups AS jt ON t.id = jt.group_id WHERE jt.user_id IN (%s)`")
	mustContain(t, content, "func (c *UserClient) LoadGroups(")

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

func mustContain(t *testing.T, content, needle string) {
	t.Helper()
	if !strings.Contains(content, needle) {
		t.Fatalf("expected generated content to contain %q\nactual: %s", needle, content)
	}
}
