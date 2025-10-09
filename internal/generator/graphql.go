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

	scalars := collectCustomScalars(entities)
	if len(scalars) > 0 {
		for _, scalar := range scalars {
			builder.WriteString(fmt.Sprintf("scalar %s\n", scalar))
		}
		builder.WriteString("\n")
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
	typ, _ := graphqlFieldType(field)
	return typ
}

func graphqlFieldType(field dsl.Field) (string, []string) {
	base, scalars := graphqlNamedType(field)
	if !field.Nullable {
		base += "!"
	}
	return base, scalars
}

func graphqlNamedType(field dsl.Field) (string, []string) {
	switch field.Type {
	case dsl.TypeArray:
		elemType, _ := field.Annotations["array_element"].(dsl.FieldType)
		if elemType == "" {
			elemType = dsl.TypeText
		}
		elemField := dsl.Field{Type: elemType}
		name, scalars := graphqlNamedType(elemField)
		name = strings.TrimSuffix(name, "!")
		return fmt.Sprintf("[%s!]", name), scalars
	case dsl.TypeVector:
		return "[Float!]", nil
	default:
		name, scalar := graphqlScalarName(field.Type)
		scalars := []string{}
		if scalar != "" {
			scalars = append(scalars, scalar)
		}
		return name, scalars
	}
}

func graphqlScalarName(ft dsl.FieldType) (string, string) {
	switch ft {
	case dsl.TypeUUID:
		return "ID", ""
	case dsl.TypeText, dsl.TypeVarChar, dsl.TypeChar, dsl.TypeBytea,
		dsl.TypePoint, dsl.TypeLine, dsl.TypeLseg, dsl.TypeBox, dsl.TypePath, dsl.TypePolygon,
		dsl.TypeCircle:
		return "String", ""
	case dsl.TypeBoolean:
		return "Boolean", ""
	case dsl.TypeSmallInt, dsl.TypeInteger, dsl.TypeSmallSerial, dsl.TypeSerial:
		return "Int", ""
	case dsl.TypeBigInt, dsl.TypeBigSerial:
		return "BigInt", "BigInt"
	case dsl.TypeDecimal, dsl.TypeNumeric:
		return "Decimal", "Decimal"
	case dsl.TypeReal, dsl.TypeDoublePrecision:
		return "Float", ""
	case dsl.TypeMoney:
		return "Money", "Money"
	case dsl.TypeDate:
		return "Date", "Date"
	case dsl.TypeTime:
		return "Time", "Time"
	case dsl.TypeTimeTZ:
		return "Timetz", "Timetz"
	case dsl.TypeTimestamp:
		return "Timestamp", "Timestamp"
	case dsl.TypeTimestampTZ:
		return "Timestamptz", "Timestamptz"
	case dsl.TypeInterval:
		return "Interval", "Interval"
	case dsl.TypeJSON:
		return "JSON", "JSON"
	case dsl.TypeJSONB:
		return "JSONB", "JSONB"
	case dsl.TypeXML:
		return "XML", "XML"
	case dsl.TypeInet:
		return "Inet", "Inet"
	case dsl.TypeCIDR:
		return "CIDR", "CIDR"
	case dsl.TypeMACAddr:
		return "MacAddr", "MacAddr"
	case dsl.TypeMACAddr8:
		return "MacAddr8", "MacAddr8"
	case dsl.TypeBit:
		return "BitString", "BitString"
	case dsl.TypeVarBit:
		return "VarBitString", "VarBitString"
	case dsl.TypeTSVector:
		return "TSVector", "TSVector"
	case dsl.TypeTSQuery:
		return "TSQuery", "TSQuery"
	case dsl.TypeInt4Range:
		return "Int4Range", "Int4Range"
	case dsl.TypeInt8Range:
		return "Int8Range", "Int8Range"
	case dsl.TypeNumRange:
		return "NumRange", "NumRange"
	case dsl.TypeTSRange:
		return "TSRange", "TSRange"
	case dsl.TypeTSTZRange:
		return "TSTZRange", "TSTZRange"
	case dsl.TypeDateRange:
		return "DateRange", "DateRange"
	case dsl.TypeGeometry, dsl.TypeGeography:
		return "JSON", "JSON"
	case dsl.TypeArray:
		return "[String!]", ""
	case dsl.TypeVector:
		return "[Float!]", ""
	default:
		return "String", ""
	}
}

var predeclaredScalars = map[string]struct{}{
	"Time": {},
}

func collectCustomScalars(entities []Entity) []string {
	set := map[string]struct{}{}
	for _, ent := range entities {
		for _, field := range ent.Fields {
			_, scalars := graphqlFieldType(field)
			for _, scalar := range scalars {
				if _, ok := predeclaredScalars[scalar]; ok {
					continue
				}
				set[scalar] = struct{}{}
			}
		}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for scalar := range set {
		out = append(out, scalar)
	}
	sort.Strings(out)
	return out
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
