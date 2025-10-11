package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deicod/erm/orm/dsl"
)

func TestPluralize(t *testing.T) {
	cases := map[string]string{
		"Company":   "companies",
		"company":   "companies",
		"Address":   "addresses",
		"Bus":       "buses",
		"Glass":     "glasses",
		"Hero":      "heroes",
		"Leaf":      "leaves",
		"Person":    "people",
		"Companies": "companies",
	}

	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			got := pluralize(in)
			if got != want {
				t.Fatalf("pluralize(%q) = %q, want %q", in, got, want)
			}
		})
	}
}

func TestGeneratorsUsePluralizedNames(t *testing.T) {
	entities := []Entity{
		{Name: "Company", Fields: []dsl.Field{{Name: "id", Type: dsl.TypeUUID, GoType: "string", IsPrimary: true}}},
		{Name: "Address", Fields: []dsl.Field{{Name: "id", Type: dsl.TypeUUID, GoType: "string", IsPrimary: true}}},
		{Name: "Bus", Fields: []dsl.Field{{Name: "id", Type: dsl.TypeUUID, GoType: "string", IsPrimary: true}}},
	}

	root := t.TempDir()
	modulePath := "example.com/app"

	if err := writeRegistry(root, entities); err != nil {
		t.Fatalf("writeRegistry: %v", err)
	}
	registry := readFile(t, filepath.Join(root, "internal", "orm", "gen", "registry_gen.go"))
	assertContains(t, registry, "Table: \"companies\"")
	assertContains(t, registry, "Table: \"addresses\"")
	assertContains(t, registry, "Table: \"buses\"")

	if err := writeClients(root, entities); err != nil {
		t.Fatalf("writeClients: %v", err)
	}
	client := readFile(t, filepath.Join(root, "internal", "orm", "gen", "client_gen.go"))
	assertContains(t, client, "INSERT INTO companies")
	assertContains(t, client, "SELECT id FROM addresses")
	assertContains(t, client, "DELETE FROM buses")

	if err := writeGraphQLArtifacts(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLArtifacts: %v", err)
	}
	schema := readFile(t, filepath.Join(root, "internal", "graphql", "schema.graphqls"))
	assertContains(t, schema, "companies(first: Int, after: String, last: Int, before: String): CompanyConnection!")
	assertContains(t, schema, "addresses(first: Int, after: String, last: Int, before: String): AddressConnection!")
	assertContains(t, schema, "buses(first: Int, after: String, last: Int, before: String): BusConnection!")

	migration := renderInitialMigration(entities, extensionFlags{})
	assertContains(t, migration, "CREATE TABLE IF NOT EXISTS companies")
	assertContains(t, migration, "CREATE TABLE IF NOT EXISTS addresses")
	assertContains(t, migration, "CREATE TABLE IF NOT EXISTS buses")
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected %q to contain %q", haystack, needle)
	}
}
