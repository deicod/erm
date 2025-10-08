package generator

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/deicod/erm/internal/orm/dsl"
)

func writeGraphQLArtifacts(root string, entities []Entity) error {
	buf := &bytes.Buffer{}
	buf.WriteString(strings.TrimSpace(graphqlBaseSchema))
	buf.WriteString("\n\n# BEGIN GENERATED\n")
	buf.WriteString(buildGraphQLGeneratedSection(entities))
	buf.WriteString("\n# END GENERATED\n")
	path := filepath.Join(root, "internal", "graphql", "schema.graphqls")
	return writeFile(path, buf.Bytes())
}

func buildGraphQLGeneratedSection(entities []Entity) string {
	if len(entities) == 0 {
		return "# No entities defined"
	}
	sort.Slice(entities, func(i, j int) bool { return entities[i].Name < entities[j].Name })
	builder := &strings.Builder{}
	if hasJSONField(entities) {
		builder.WriteString("scalar JSON\n\n")
	}

	for _, ent := range entities {
		builder.WriteString(renderEntityType(ent))
		builder.WriteString("\n")
		builder.WriteString(renderConnectionTypes(ent))
		builder.WriteString("\n")
	}

	builder.WriteString("extend type Query {\n")
	for _, ent := range entities {
		builder.WriteString(fmt.Sprintf("  %s(first: Int, after: String, last: Int, before: String): %sConnection!\n", lowerCamel(pluralize(ent.Name)), ent.Name))
	}
	builder.WriteString("}\n")

	return builder.String()
}

func renderEntityType(ent Entity) string {
	builder := &strings.Builder{}
	builder.WriteString(fmt.Sprintf("type %s implements Node {\n", ent.Name))
	seenID := false
	for _, field := range ent.Fields {
		gqlType := fieldGraphQLType(field)
		if field.Name == "id" {
			seenID = true
		}
		builder.WriteString(fmt.Sprintf("  %s: %s\n", lowerCamel(field.Name), gqlType))
	}
	if !seenID {
		builder.WriteString("  id: ID!\n")
	}
	builder.WriteString("}\n")
	return builder.String()
}

func renderConnectionTypes(ent Entity) string {
	builder := &strings.Builder{}
	builder.WriteString(fmt.Sprintf("type %sEdge {\n  cursor: String!\n  node: %s\n}\n\n", ent.Name, ent.Name))
	builder.WriteString(fmt.Sprintf("type %sConnection {\n  edges: [%sEdge!]!\n  pageInfo: PageInfo!\n  totalCount: Int!\n}\n", ent.Name, ent.Name))
	return builder.String()
}

func fieldGraphQLType(field dsl.Field) string {
	var base string
	switch field.Type {
	case dsl.TypeUUID:
		base = "ID"
	case dsl.TypeString:
		base = "String"
	case dsl.TypeInt:
		base = "Int"
	case dsl.TypeFloat:
		base = "Float"
	case dsl.TypeBool:
		base = "Boolean"
	case dsl.TypeBytes:
		base = "String"
	case dsl.TypeTime:
		base = "Time"
	case dsl.TypeJSON:
		base = "JSON"
	default:
		base = "String"
	}
	if !field.Nullable {
		base += "!"
	}
	return base
}

func hasJSONField(entities []Entity) bool {
	for _, ent := range entities {
		for _, f := range ent.Fields {
			if f.Type == dsl.TypeJSON {
				return true
			}
		}
	}
	return false
}

func lowerCamel(name string) string {
	if name == "" {
		return name
	}
	if strings.ToUpper(name) == name {
		return strings.ToLower(name)
	}
	parts := strings.Split(name, "_")
	if len(parts) > 1 {
		for i, part := range parts {
			if part == "" {
				continue
			}
			if i == 0 {
				parts[i] = strings.ToLower(part)
			} else {
				parts[i] = capitalize(part)
			}
		}
		return strings.Join(parts, "")
	}
	runes := []rune(name)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

const graphqlBaseSchema = `scalar Time

interface Node { id: ID! }

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

directive @auth(roles: [String!]) on FIELD_DEFINITION

type Query {
  node(id: ID!): Node
  health: String!
}

type Mutation {
  _noop: Boolean
}`
