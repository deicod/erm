package generator

import (
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
				dsl.TimestampTZ("created_at"),
				dsl.JSONB("metadata"),
				dsl.Array("tags", dsl.TypeText),
				dsl.Vector("embedding", 3),
			},
		},
	}

	schema := buildGraphQLGeneratedSection(entities)

	checks := []string{
		"scalar BigInt",
		"scalar Decimal",
		"scalar JSONB",
		"scalar Timestamptz",
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

	resolverPath := filepath.Join(root, "graphql", "resolvers", "entities_gen.go")
	resolverSrc, err := os.ReadFile(resolverPath)
	if err != nil {
		t.Fatalf("read resolvers: %v", err)
	}
	expectations := []string{
		"func (r *mutationResolver) CreateWidget",
		"func (r *queryResolver) Widgets",
		"func decodeWidgetID",
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
