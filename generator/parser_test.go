package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deicod/erm/orm/dsl"
)

func TestLoadEntitiesSynthesizesInverseEdges(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schema")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatalf("mkdir schema: %v", err)
	}

	source := `package schema

import "github.com/deicod/erm/orm/dsl"

type User struct{ dsl.Schema }

type Pet struct{ dsl.Schema }

func (User) Fields() []dsl.Field { return nil }
func (User) Edges() []dsl.Edge {
        return []dsl.Edge{
                dsl.ToMany("pets", "Pet").Inverse("owner"),
        }
}
func (User) Indexes() []dsl.Index { return nil }

func (Pet) Fields() []dsl.Field { return nil }
func (Pet) Edges() []dsl.Edge { return nil }
func (Pet) Indexes() []dsl.Index { return nil }
`
	if err := os.WriteFile(filepath.Join(schemaDir, "entities.schema.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	entities, err := loadEntities(dir)
	if err != nil {
		t.Fatalf("loadEntities: %v", err)
	}

	user := findEntity(entities, "User")
	if len(user.Edges) != 1 {
		t.Fatalf("expected 1 user edge, got %d", len(user.Edges))
	}
	pets := user.Edges[0]
	if pets.InverseName != "owner" {
		t.Fatalf("expected inverse name 'owner', got %q", pets.InverseName)
	}

	pet := findEntity(entities, "Pet")
	if len(pet.Edges) != 1 {
		t.Fatalf("expected synthesized pet edge, got %d", len(pet.Edges))
	}
	owner := pet.Edges[0]
	if owner.Name != "owner" {
		t.Fatalf("expected synthesized edge named 'owner', got %q", owner.Name)
	}
	if owner.Target != "User" {
		t.Fatalf("expected synthesized edge target 'User', got %q", owner.Target)
	}
	if owner.Kind != dsl.EdgeToOne {
		t.Fatalf("expected synthesized edge kind EdgeToOne, got %v", owner.Kind)
	}
	if owner.InverseName != "pets" {
		t.Fatalf("expected synthesized edge inverse 'pets', got %q", owner.InverseName)
	}
	if owner.Column != "user_id" {
		t.Fatalf("expected synthesized edge column 'user_id', got %q", owner.Column)
	}
	if owner.RefName != "user_id" {
		t.Fatalf("expected synthesized edge ref name 'user_id', got %q", owner.RefName)
	}
}

func TestLoadEntitiesSkipsExistingInverse(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schema")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatalf("mkdir schema: %v", err)
	}

	source := `package schema

import "github.com/deicod/erm/orm/dsl"

type Author struct{ dsl.Schema }

type Article struct{ dsl.Schema }

func (Author) Fields() []dsl.Field { return nil }
func (Author) Edges() []dsl.Edge {
        return []dsl.Edge{
                dsl.ToMany("articles", "Article").Inverse("author"),
        }
}
func (Author) Indexes() []dsl.Index { return nil }

func (Article) Fields() []dsl.Field { return nil }
func (Article) Edges() []dsl.Edge {
        return []dsl.Edge{
                dsl.ToOne("author", "Author"),
        }
}
func (Article) Indexes() []dsl.Index { return nil }
`
	if err := os.WriteFile(filepath.Join(schemaDir, "blog.schema.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	entities, err := loadEntities(dir)
	if err != nil {
		t.Fatalf("loadEntities: %v", err)
	}

	article := findEntity(entities, "Article")
	if got := len(article.Edges); got != 1 {
		t.Fatalf("expected 1 article edge, got %d", got)
	}
	if article.Edges[0].Name != "author" {
		t.Fatalf("expected article edge 'author', got %q", article.Edges[0].Name)
	}
}

func TestInverseEdgeDerivesDefaultNames(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schema")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatalf("mkdir schema: %v", err)
	}

	source := `package schema

import "github.com/deicod/erm/orm/dsl"

type User struct{ dsl.Schema }

type Profile struct{ dsl.Schema }

func (User) Fields() []dsl.Field { return nil }
func (User) Edges() []dsl.Edge { return nil }
func (User) Indexes() []dsl.Index { return nil }

func (Profile) Fields() []dsl.Field { return nil }
func (Profile) Edges() []dsl.Edge {
        return []dsl.Edge{
                dsl.ToOne("user", "User").Inverse("profiles"),
        }
}
func (Profile) Indexes() []dsl.Index { return nil }
`
	if err := os.WriteFile(filepath.Join(schemaDir, "profile.schema.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	entities, err := loadEntities(dir)
	if err != nil {
		t.Fatalf("loadEntities: %v", err)
	}

	user := findEntity(entities, "User")
	if len(user.Edges) != 1 {
		t.Fatalf("expected synthesized user edge, got %d", len(user.Edges))
	}
	profiles := user.Edges[0]
	if profiles.Name != "profiles" {
		t.Fatalf("expected synthesized edge named 'profiles', got %q", profiles.Name)
	}
	if profiles.RefName != "user_id" {
		t.Fatalf("expected synthesized ref name 'user_id', got %q", profiles.RefName)
	}
}

func TestInverseEdgeHonorsExplicitNames(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schema")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatalf("mkdir schema: %v", err)
	}

	source := `package schema

import "github.com/deicod/erm/orm/dsl"

type User struct{ dsl.Schema }

type Tenant struct{ dsl.Schema }

type Order struct{ dsl.Schema }

func (User) Fields() []dsl.Field { return nil }
func (User) Edges() []dsl.Edge {
        return []dsl.Edge{
                dsl.ToMany("orders", "Order").Ref("customer_fk").Inverse("customer"),
        }
}
func (User) Indexes() []dsl.Index { return nil }

func (Tenant) Fields() []dsl.Field { return nil }
func (Tenant) Edges() []dsl.Edge {
        return []dsl.Edge{
                dsl.ToOne("owner", "User").Field("tenant_owner").Inverse("managedTenants"),
        }
}
func (Tenant) Indexes() []dsl.Index { return nil }

func (Order) Fields() []dsl.Field { return nil }
func (Order) Edges() []dsl.Edge { return nil }
func (Order) Indexes() []dsl.Index { return nil }
`
	if err := os.WriteFile(filepath.Join(schemaDir, "edges.schema.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	entities, err := loadEntities(dir)
	if err != nil {
		t.Fatalf("loadEntities: %v", err)
	}

	order := findEntity(entities, "Order")
	if len(order.Edges) != 1 {
		t.Fatalf("expected synthesized order edge, got %d", len(order.Edges))
	}
	customer := order.Edges[0]
	if customer.Column != "customer_fk" {
		t.Fatalf("expected synthesized column 'customer_fk', got %q", customer.Column)
	}

	user := findEntity(entities, "User")
	if len(user.Edges) != 2 {
		t.Fatalf("expected two user edges including inverse, got %d", len(user.Edges))
	}
	var managedTenants dsl.Edge
	found := false
	for _, edge := range user.Edges {
		if edge.Name == "managedTenants" {
			managedTenants = edge
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected synthesized edge 'managedTenants' on user")
	}
	if managedTenants.RefName != "tenant_owner" {
		t.Fatalf("expected synthesized ref name 'tenant_owner', got %q", managedTenants.RefName)
	}
}

func TestWriteRegistryIncludesInverse(t *testing.T) {
	root := t.TempDir()
	entities := []Entity{
		{
			Name: "User",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
			},
			Edges: []dsl.Edge{
				dsl.ToMany("pets", "Pet").Inverse("owner"),
			},
		},
		{
			Name: "Pet",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
			},
			Edges: []dsl.Edge{
				dsl.ToOne("owner", "User").Inverse("pets"),
			},
		},
	}

	if err := writeRegistry(root, entities); err != nil {
		t.Fatalf("writeRegistry: %v", err)
	}

	path := filepath.Join(root, "orm", "gen", "registry_gen.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Inverse: \"owner\"") {
		t.Fatalf("expected registry to contain owner inverse, got:\n%s", content)
	}
	if !strings.Contains(content, "Inverse: \"pets\"") {
		t.Fatalf("expected registry to contain pets inverse, got:\n%s", content)
	}
}

func TestEnumAndDefaultParsing(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schema")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatalf("mkdir schema: %v", err)
	}

	source := `package schema

import "github.com/deicod/erm/orm/dsl"

type Task struct{ dsl.Schema }

func (Task) Fields() []dsl.Field {
        return []dsl.Field{
                dsl.Enum("status", "NEW", "DONE").Default("NEW"),
                dsl.Boolean("archived").Default(false),
        }
}

func (Task) Edges() []dsl.Edge { return nil }
func (Task) Indexes() []dsl.Index { return nil }
`
	if err := os.WriteFile(filepath.Join(schemaDir, "task.schema.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	entities, err := loadEntities(dir)
	if err != nil {
		t.Fatalf("loadEntities: %v", err)
	}

	task := findEntity(entities, "Task")
	if len(task.Fields) != 3 {
		t.Fatalf("expected 3 fields including id, got %d", len(task.Fields))
	}

	var statusField, archivedField dsl.Field
	for _, field := range task.Fields {
		switch field.Name {
		case "status":
			statusField = field
		case "archived":
			archivedField = field
		}
	}

	if len(statusField.EnumValues) != 2 {
		t.Fatalf("expected enum values, got %#v", statusField.EnumValues)
	}
	if statusField.EnumName != "TaskStatus" {
		t.Fatalf("expected enum name TaskStatus, got %q", statusField.EnumName)
	}
	if statusField.DefaultExpr != "'NEW'" {
		t.Fatalf("expected enum default 'NEW', got %q", statusField.DefaultExpr)
	}
	if archivedField.DefaultExpr != "FALSE" {
		t.Fatalf("expected boolean default FALSE, got %q", archivedField.DefaultExpr)
	}
}
