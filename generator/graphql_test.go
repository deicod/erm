package generator

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deicod/erm/orm/dsl"
)

func TestGraphQLTypeMappings(t *testing.T) {
	entities := []Entity{
		{
			Name: "Sample",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.Text("title"),
				dsl.BigInt("counter"),
				dsl.Decimal("price", 10, 2),
				dsl.Date("ship_date").Optional(),
				dsl.Time("ship_time").Optional(),
				dsl.TimestampTZ("created_at"),
				dsl.JSONB("metadata"),
				dsl.Array("tags", dsl.TypeText),
				dsl.Vector("embedding", 3),
			},
		},
	}

	schema := buildGraphQLGeneratedSection(entities)

	checks := []string{
		"counter: BigInt!",
		"price: Decimal!",
		"metadata: JSONB!",
		"createdAt: Timestamptz!",
		"shipDate: Date",
		"shipTime: Time",
		"title: String",
		"CreateSampleInput",
		"updateSample",
		"deleteSample",
	}

	for _, needle := range checks {
		if !strings.Contains(schema, needle) {
			t.Fatalf("expected GraphQL schema to contain %q\nactual: %s", needle, schema)
		}
	}

	declaredScalars := []string{
		"scalar BigInt",
		"scalar Decimal",
		"scalar JSONB",
		"scalar Timestamptz",
		"scalar Date",
	}

	for _, scalar := range declaredScalars {
		mustContain(t, schema, scalar)
	}

	mustNotContain(t, schema, "scalar Time\n")
}

func TestGraphQLEnumGeneration(t *testing.T) {
	entities := []Entity{{
		Name: "Task",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
			dsl.Enum("status", "NEW", "DONE"),
		},
	}}
	assignEnumMetadata(entities)

	schema := buildGraphQLGeneratedSection(entities)
	mustContain(t, schema, "enum TaskStatus")
	mustContain(t, schema, "status: TaskStatus!")
}

func TestGraphQLResolverGeneration(t *testing.T) {
	entities := []Entity{{
		Name: "Widget",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
			dsl.Text("name"),
			dsl.Integer("version"),
		},
	}}

	root := t.TempDir()
	modulePath := "example.com/app"
	if err := writeGraphQLResolvers(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLResolvers: %v", err)
	}
	if err := writeGraphQLDataloaders(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLDataloaders: %v", err)
	}

	hooksPath := filepath.Join(root, "graphql", "resolvers", "entities_hooks.go")
	hooksSrc, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("read hooks: %v", err)
	}
	mustContain(t, string(hooksSrc), "newEntityHooks")

	resolverPath := filepath.Join(root, "graphql", "resolvers", "entities_gen.go")
	resolverSrc, err := os.ReadFile(resolverPath)
	if err != nil {
		t.Fatalf("read resolvers: %v", err)
	}
	expectations := []string{
		"func (r *mutationResolver) CreateWidget",
		"func (r *queryResolver) Widgets",
		"func decodeWidgetID",
		"type entityHooks struct",
		"applyBeforeCreateWidget",
		"applyBeforeReturnWidget",
		modulePath + "/graphql",
		modulePath + "/graphql/dataloaders",
	}
	for _, needle := range expectations {
		if !strings.Contains(string(resolverSrc), needle) {
			t.Fatalf("expected resolver source to contain %q\n%s", needle, resolverSrc)
		}
	}
	if strings.Contains(string(resolverSrc), "github.com/deicod/erm") {
		t.Fatalf("resolver source still references repository module path\n%s", resolverSrc)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), resolverPath, resolverSrc, parser.AllErrors); err != nil {
		t.Fatalf("resolvers parse: %v", err)
	}

	if err := writeGraphQLResolvers(root, entities, modulePath); err != nil {
		t.Fatalf("second writeGraphQLResolvers: %v", err)
	}
	hooksAgain, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("read hooks second: %v", err)
	}
	if !bytes.Equal(hooksSrc, hooksAgain) {
		t.Fatalf("expected hooks stub to remain unchanged")
	}

	loaderPath := filepath.Join(root, "graphql", "dataloaders", "entities_gen.go")
	loaderSrc, err := os.ReadFile(loaderPath)
	if err != nil {
		t.Fatalf("read dataloaders: %v", err)
	}
	loaderExpectations := []string{
		"configureEntityLoaders",
		"func (l *Loaders) Widget()",
		"orm.Widgets().ByID",
		modulePath + "/observability/metrics",
	}
	for _, needle := range loaderExpectations {
		if !strings.Contains(string(loaderSrc), needle) {
			t.Fatalf("expected dataloader source to contain %q\n%s", needle, loaderSrc)
		}
	}
	if strings.Contains(string(loaderSrc), "github.com/deicod/erm") {
		t.Fatalf("dataloader source still references repository module path\n%s", loaderSrc)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), loaderPath, loaderSrc, parser.AllErrors); err != nil {
		t.Fatalf("dataloaders parse: %v", err)
	}
}

func TestGraphQLInitialismHandling(t *testing.T) {
	entities := []Entity{{
		Name: "Profile",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
			dsl.UUIDv7("post_id"),
			dsl.String("avatar_url"),
			dsl.String("api_token"),
		},
	}}

	schema := buildGraphQLGeneratedSection(entities)
	mustContain(t, schema, "postID")
	mustContain(t, schema, "avatarURL")
	mustContain(t, schema, "apiToken")

	root := t.TempDir()
	modulePath := "example.com/app"
	if err := writeGraphQLResolvers(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLResolvers: %v", err)
	}

	resolverPath := filepath.Join(root, "graphql", "resolvers", "entities_gen.go")
	resolverSrc, err := os.ReadFile(resolverPath)
	if err != nil {
		t.Fatalf("read resolvers: %v", err)
	}

	mustContain(t, string(resolverSrc), "input.PostID")
	mustContain(t, string(resolverSrc), "model.PostID")
	mustContain(t, string(resolverSrc), "input.AvatarURL")
	mustContain(t, string(resolverSrc), "model.AvatarURL")
	mustContain(t, string(resolverSrc), "input.APIToken")
	mustContain(t, string(resolverSrc), "model.APIToken")
}

func TestWriteGraphQLArtifactsEnsuresScalarHelpers(t *testing.T) {
	template, err := os.ReadFile(filepath.Join("..", "cli", "templates", "graphql", "scalars.go.tmpl"))
	if err != nil {
		t.Fatalf("read scalars template: %v", err)
	}
	RegisterGraphQLScalarTemplate(template)
	t.Cleanup(func() { RegisterGraphQLScalarTemplate(nil) })

	root := t.TempDir()
	modulePath := "example.com/app"
	entities := []Entity{{
		Name: "Widget",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
		},
	}}

	if err := writeGraphQLArtifacts(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLArtifacts: %v", err)
	}

	path := filepath.Join(root, "graphql", "scalars.go")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected scalars helper to be written: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read scalars helper: %v", err)
	}
	if string(content) != string(template) {
		t.Fatalf("unexpected scalars helper content\nwant:\n%s\ngot:\n%s", template, content)
	}
}

func mustNotContain(t *testing.T, content, needle string) {
	t.Helper()
	if strings.Contains(content, needle) {
		t.Fatalf("expected GraphQL schema to omit %q\nactual: %s", needle, content)
	}
}
