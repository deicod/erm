package generator

import (
	"strings"
	"testing"

	"github.com/deicod/erm/internal/orm/dsl"
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
		"title: String!",
		"counter: BigInt!",
		"price: Decimal!",
		"shipDate: Date",
		"createdAt: Timestamptz!",
		"metadata: JSONB!",
		"tags: [String!]!",
		"embedding: [Float!]!",
	}

	for _, needle := range checks {
		if !strings.Contains(schema, needle) {
			t.Fatalf("expected GraphQL schema to contain %q\nactual: %s", needle, schema)
		}
	}
}
