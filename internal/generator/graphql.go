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
	if err := writeGraphQLSchema(root, entities); err != nil {
		return err
	}
	if err := writeGraphQLResolvers(root, entities); err != nil {
		return err
	}
	if err := writeGraphQLDataloaders(root, entities); err != nil {
		return err
	}
	return nil
}

func writeGraphQLSchema(root string, entities []Entity) error {
	buf := &bytes.Buffer{}
	buf.WriteString(strings.TrimSpace(graphqlBaseSchema))
	buf.WriteString("\n\n# BEGIN GENERATED\n")
	buf.WriteString(buildGraphQLGeneratedSection(entities))
	buf.WriteString("\n# END GENERATED\n")
	path := filepath.Join(root, "internal", "graphql", "schema.graphqls")
	_, err := writeFile(path, buf.Bytes())
	return err
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

	queryFields := make([]string, 0, len(entities)*2)
	mutationFields := make([]string, 0, len(entities)*3)
	for _, ent := range entities {
		builder.WriteString(renderEntityType(ent))
		builder.WriteString("\n")
		builder.WriteString(renderConnectionTypes(ent))
		builder.WriteString("\n")
		builder.WriteString(renderEntityInputTypes(ent))
		builder.WriteString("\n")
		queryFields = append(queryFields, renderEntityQueryFields(ent)...)
		mutationFields = append(mutationFields, renderEntityMutationFields(ent)...)
	}
	if len(queryFields) > 0 {
		builder.WriteString("extend type Query {\n")
		for _, field := range queryFields {
			builder.WriteString(fmt.Sprintf("  %s\n", field))
		}
		builder.WriteString("}\n")
	}
	if len(mutationFields) > 0 {
		builder.WriteString("\nextend type Mutation {\n")
		for _, field := range mutationFields {
			builder.WriteString(fmt.Sprintf("  %s\n", field))
		}
		builder.WriteString("}\n")
	}

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

func renderEntityInputTypes(ent Entity) string {
	builder := &strings.Builder{}
	builder.WriteString(fmt.Sprintf("input Create%sInput {\n", ent.Name))
	builder.WriteString("  clientMutationId: String\n")
	for _, field := range ent.Fields {
		fieldName := lowerCamel(field.Name)
		if fieldName == "" {
			continue
		}
		if field.Name == "id" {
			builder.WriteString(fmt.Sprintf("  %s: ID\n", fieldName))
			continue
		}
		gqlType, _ := graphqlNamedType(field)
		builder.WriteString(fmt.Sprintf("  %s: %s\n", fieldName, trimNonNull(gqlType)))
	}
	builder.WriteString("}\n\n")

	builder.WriteString(fmt.Sprintf("type Create%sPayload {\n", ent.Name))
	builder.WriteString("  clientMutationId: String\n")
	builder.WriteString(fmt.Sprintf("  %s: %s\n", lowerCamel(ent.Name), ent.Name))
	builder.WriteString("}\n\n")

	builder.WriteString(fmt.Sprintf("input Update%sInput {\n", ent.Name))
	builder.WriteString("  clientMutationId: String\n")
	builder.WriteString("  id: ID!\n")
	for _, field := range ent.Fields {
		if field.Name == "id" {
			continue
		}
		fieldName := lowerCamel(field.Name)
		if fieldName == "" {
			continue
		}
		gqlType, _ := graphqlNamedType(field)
		builder.WriteString(fmt.Sprintf("  %s: %s\n", fieldName, trimNonNull(gqlType)))
	}
	builder.WriteString("}\n\n")

	builder.WriteString(fmt.Sprintf("type Update%sPayload {\n", ent.Name))
	builder.WriteString("  clientMutationId: String\n")
	builder.WriteString(fmt.Sprintf("  %s: %s\n", lowerCamel(ent.Name), ent.Name))
	builder.WriteString("}\n\n")

	builder.WriteString(fmt.Sprintf("input Delete%sInput {\n", ent.Name))
	builder.WriteString("  clientMutationId: String\n")
	builder.WriteString("  id: ID!\n")
	builder.WriteString("}\n\n")

	builder.WriteString(fmt.Sprintf("type Delete%sPayload {\n", ent.Name))
	builder.WriteString("  clientMutationId: String\n")
	builder.WriteString(fmt.Sprintf("  deleted%sID: ID!\n", ent.Name))
	builder.WriteString("}\n")

	return builder.String()
}

func renderEntityQueryFields(ent Entity) []string {
	plural := lowerCamel(pluralize(ent.Name))
	singular := lowerCamel(ent.Name)
	return []string{
		fmt.Sprintf("%s(id: ID!): %s", singular, ent.Name),
		fmt.Sprintf("%s(first: Int, after: String, last: Int, before: String): %sConnection!", plural, ent.Name),
	}
}

func renderEntityMutationFields(ent Entity) []string {
	return []string{
		fmt.Sprintf("create%s(input: Create%sInput!): Create%sPayload!", ent.Name, ent.Name, ent.Name),
		fmt.Sprintf("update%s(input: Update%sInput!): Update%sPayload!", ent.Name, ent.Name, ent.Name),
		fmt.Sprintf("delete%s(input: Delete%sInput!): Delete%sPayload!", ent.Name, ent.Name, ent.Name),
	}
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

func trimNonNull(typ string) string {
	return strings.TrimSuffix(typ, "!")
}

func writeGraphQLResolvers(root string, entities []Entity) error {
	sort.Slice(entities, func(i, j int) bool { return entities[i].Name < entities[j].Name })
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "// Code generated by erm. DO NOT EDIT.\n")
	fmt.Fprintf(buf, "package resolvers\n\n")

	imports := map[string]struct{}{
		"context":                                {},
		"fmt":                                    {},
		"github.com/deicod/erm/internal/graphql": {},
		"github.com/deicod/erm/internal/graphql/relay": {},
	}
	if len(entities) > 0 {
		imports["github.com/deicod/erm/internal/graphql/dataloaders"] = struct{}{}
		imports["github.com/deicod/erm/internal/orm/gen"] = struct{}{}
	}
	if len(imports) > 0 {
		fmt.Fprintf(buf, "import (\n")
		keys := make([]string, 0, len(imports))
		for key := range imports {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(buf, "    \"%s\"\n", key)
		}
		fmt.Fprintf(buf, ")\n\n")
	}

	buf.WriteString(renderNodeResolver(entities))
	if len(entities) > 0 {
		for _, ent := range entities {
			buf.WriteString(renderEntityHelpers(ent))
			buf.WriteString("\n")
			buf.WriteString(renderEntityQueryResolvers(ent))
			buf.WriteString("\n")
			buf.WriteString(renderEntityMutationResolvers(ent))
			buf.WriteString("\n")
		}
	}

	path := filepath.Join(root, "internal", "graphql", "resolvers", "entities_gen.go")
	return writeGoFile(path, buf.Bytes())
}

func renderNodeResolver(entities []Entity) string {
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "func (r *queryResolver) Node(ctx context.Context, id string) (graphql.Node, error) {\n")
	fmt.Fprintf(builder, "    typ, nativeID, err := relay.FromGlobalID(id)\n")
	fmt.Fprintf(builder, "    if err != nil {\n        return nil, err\n    }\n")
	if len(entities) == 0 {
		fmt.Fprintf(builder, "    return nil, fmt.Errorf(\"unknown node type %%s\", typ)\n}\n\n")
		return builder.String()
	}
	fmt.Fprintf(builder, "    switch typ {\n")
	for _, ent := range entities {
		fmt.Fprintf(builder, "    case \"%s\":\n", ent.Name)
		fmt.Fprintf(builder, "        record, err := r.load%[1]s(ctx, nativeID)\n", ent.Name)
		fmt.Fprintf(builder, "        if err != nil {\n            return nil, err\n        }\n")
		fmt.Fprintf(builder, "        if record == nil {\n            return nil, nil\n        }\n")
		fmt.Fprintf(builder, "        return toGraphQL%[1]s(record), nil\n", ent.Name)
	}
	fmt.Fprintf(builder, "    default:\n        return nil, fmt.Errorf(\"unknown node type %%s\", typ)\n    }\n}\n\n")
	return builder.String()
}

func renderEntityHelpers(ent Entity) string {
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "func (r *Resolver) load%[1]s(ctx context.Context, id string) (*gen.%[1]s, error) {\n", ent.Name)
	fmt.Fprintf(builder, "    if r == nil || r.ORM == nil {\n        return nil, nil\n    }\n")
	fmt.Fprintf(builder, "    if loaders := dataloaders.FromContext(ctx); loaders != nil {\n")
	fmt.Fprintf(builder, "        if loader := loaders.%[1]s(); loader != nil {\n", ent.Name)
	fmt.Fprintf(builder, "            return loader.Load(ctx, id)\n        }\n    }\n")
	fmt.Fprintf(builder, "    return r.ORM.%[1]s().ByID(ctx, id)\n}\n\n", exportName(pluralize(ent.Name)))

	fmt.Fprintf(builder, "func (r *Resolver) prime%[1]s(ctx context.Context, record *gen.%[1]s) {\n", ent.Name)
	fmt.Fprintf(builder, "    if record == nil {\n        return\n    }\n")
	fmt.Fprintf(builder, "    if loaders := dataloaders.FromContext(ctx); loaders != nil {\n")
	fmt.Fprintf(builder, "        if loader := loaders.%[1]s(); loader != nil {\n", ent.Name)
	fmt.Fprintf(builder, "            loader.Prime(record.ID, record)\n        }\n    }\n}\n\n")

	fmt.Fprintf(builder, "func toGraphQL%[1]s(record *gen.%[1]s) *graphql.%[1]s {\n", ent.Name)
	fmt.Fprintf(builder, "    if record == nil {\n        return nil\n    }\n")
	fmt.Fprintf(builder, "    return &graphql.%[1]s{\n", ent.Name)
	fmt.Fprintf(builder, "        ID: relay.ToGlobalID(\"%[1]s\", record.ID),\n", ent.Name)
	for _, field := range ent.Fields {
		if field.Name == "id" {
			continue
		}
		fieldName := exportName(field.Name)
		fmt.Fprintf(builder, "        %s: record.%s,\n", fieldName, fieldName)
	}
	fmt.Fprintf(builder, "    }\n}\n\n")

	fmt.Fprintf(builder, "func decode%[1]sID(id string) (string, error) {\n", ent.Name)
	fmt.Fprintf(builder, "    if id == \"\" {\n        return \"\", fmt.Errorf(\"id is required\")\n    }\n")
	fmt.Fprintf(builder, "    typ, nativeID, err := relay.FromGlobalID(id)\n")
	fmt.Fprintf(builder, "    if err != nil {\n        return id, nil\n    }\n")
	fmt.Fprintf(builder, "    if typ != \"%[1]s\" {\n        return \"\", fmt.Errorf(\"invalid id for %[1]s: %%s\", typ)\n    }\n", ent.Name)
	fmt.Fprintf(builder, "    return nativeID, nil\n}\n\n")

	return builder.String()
}

func renderEntityQueryResolvers(ent Entity) string {
	builder := &strings.Builder{}
	pluralName := exportName(pluralize(ent.Name))
	pluralField := lowerCamel(pluralize(ent.Name))

	fmt.Fprintf(builder, "func (r *queryResolver) %[1]s(ctx context.Context, id string) (*graphql.%[2]s, error) {\n", exportName(ent.Name), ent.Name)
	fmt.Fprintf(builder, "    nativeID, err := decode%sID(id)\n", ent.Name)
	fmt.Fprintf(builder, "    if err != nil {\n        return nil, err\n    }\n")
	fmt.Fprintf(builder, "    record, err := r.load%s(ctx, nativeID)\n", ent.Name)
	fmt.Fprintf(builder, "    if err != nil {\n        return nil, err\n    }\n")
	fmt.Fprintf(builder, "    return toGraphQL%s(record), nil\n}\n\n", ent.Name)

	fmt.Fprintf(builder, "func (r *queryResolver) %[1]s(ctx context.Context, first *int, after *string, last *int, before *string) (*graphql.%[2]sConnection, error) {\n", exportName(pluralField), ent.Name)
	fmt.Fprintf(builder, "    if r.ORM == nil {\n        return nil, fmt.Errorf(\"orm client is not configured\")\n    }\n")
	fmt.Fprintf(builder, "    if last != nil || before != nil {\n        return nil, fmt.Errorf(\"backward pagination is not supported\")\n    }\n")
	fmt.Fprintf(builder, "    limit := defaultPageSize\n")
	fmt.Fprintf(builder, "    if first != nil && *first > 0 {\n        limit = *first\n    }\n")
	fmt.Fprintf(builder, "    offset := 0\n")
	fmt.Fprintf(builder, "    if after != nil && *after != \"\" {\n")
	fmt.Fprintf(builder, "        if decoded, err := decodeCursor(*after); err == nil {\n            offset = decoded + 1\n        }\n    }\n")
	fmt.Fprintf(builder, "    total, err := r.ORM.%[1]s().Count(ctx)\n", pluralName)
	fmt.Fprintf(builder, "    if err != nil {\n        return nil, err\n    }\n")
	fmt.Fprintf(builder, "    records, err := r.ORM.%[1]s().List(ctx, limit, offset)\n", pluralName)
	fmt.Fprintf(builder, "    if err != nil {\n        return nil, err\n    }\n")
	fmt.Fprintf(builder, "    edges := make([]*graphql.%sEdge, len(records))\n", ent.Name)
	fmt.Fprintf(builder, "    for idx, record := range records {\n")
	fmt.Fprintf(builder, "        cursor := encodeCursor(offset + idx)\n")
	fmt.Fprintf(builder, "        r.prime%s(ctx, record)\n", ent.Name)
	fmt.Fprintf(builder, "        edges[idx] = &graphql.%sEdge{\n", ent.Name)
	fmt.Fprintf(builder, "            Cursor: cursor,\n")
	fmt.Fprintf(builder, "            Node:   toGraphQL%s(record),\n", ent.Name)
	fmt.Fprintf(builder, "        }\n    }\n")
	fmt.Fprintf(builder, "    var startCursor, endCursor *string\n")
	fmt.Fprintf(builder, "    if len(edges) > 0 {\n")
	fmt.Fprintf(builder, "        sc := edges[0].Cursor\n")
	fmt.Fprintf(builder, "        ec := edges[len(edges)-1].Cursor\n")
	fmt.Fprintf(builder, "        startCursor = &sc\n")
	fmt.Fprintf(builder, "        endCursor = &ec\n    }\n")
	fmt.Fprintf(builder, "    pageInfo := &graphql.PageInfo{\n")
	fmt.Fprintf(builder, "        HasNextPage:     offset+len(edges) < total,\n")
	fmt.Fprintf(builder, "        HasPreviousPage: offset > 0,\n")
	fmt.Fprintf(builder, "        StartCursor:     startCursor,\n")
	fmt.Fprintf(builder, "        EndCursor:       endCursor,\n")
	fmt.Fprintf(builder, "    }\n")
	fmt.Fprintf(builder, "    return &graphql.%sConnection{\n", ent.Name)
	fmt.Fprintf(builder, "        Edges:      edges,\n")
	fmt.Fprintf(builder, "        PageInfo:   pageInfo,\n")
	fmt.Fprintf(builder, "        TotalCount: total,\n")
	fmt.Fprintf(builder, "    }, nil\n}\n\n")

	return builder.String()
}

func renderEntityMutationResolvers(ent Entity) string {
	builder := &strings.Builder{}
	pluralName := exportName(pluralize(ent.Name))

	fmt.Fprintf(builder, "func (r *mutationResolver) Create%[1]s(ctx context.Context, input graphql.Create%[1]sInput) (*graphql.Create%[1]sPayload, error) {\n", ent.Name)
	fmt.Fprintf(builder, "    if r.ORM == nil {\n        return nil, fmt.Errorf(\"orm client is not configured\")\n    }\n")
	fmt.Fprintf(builder, "    model := new(gen.%[1]s)\n", ent.Name)
	for _, field := range ent.Fields {
		builder.WriteString(renderInputAssignment("input", "model", field, true))
	}
	fmt.Fprintf(builder, "    record, err := r.ORM.%s().Create(ctx, model)\n", pluralName)
	fmt.Fprintf(builder, "    if err != nil {\n        return nil, err\n    }\n")
	fmt.Fprintf(builder, "    r.prime%[1]s(ctx, record)\n", ent.Name)
	fmt.Fprintf(builder, "    return &graphql.Create%[1]sPayload{\n", ent.Name)
	fmt.Fprintf(builder, "        ClientMutationID: input.ClientMutationID,\n")
	fmt.Fprintf(builder, "        %s: toGraphQL%s(record),\n", exportName(ent.Name), ent.Name)
	fmt.Fprintf(builder, "    }, nil\n}\n\n")

	fmt.Fprintf(builder, "func (r *mutationResolver) Update%[1]s(ctx context.Context, input graphql.Update%[1]sInput) (*graphql.Update%[1]sPayload, error) {\n", ent.Name)
	fmt.Fprintf(builder, "    if r.ORM == nil {\n        return nil, fmt.Errorf(\"orm client is not configured\")\n    }\n")
	fmt.Fprintf(builder, "    nativeID, err := decode%[1]sID(input.ID)\n", ent.Name)
	fmt.Fprintf(builder, "    if err != nil {\n        return nil, err\n    }\n")
	fmt.Fprintf(builder, "    model := &gen.%[1]s{ID: nativeID}\n", ent.Name)
	for _, field := range ent.Fields {
		if field.Name == "id" {
			continue
		}
		builder.WriteString(renderInputAssignment("input", "model", field, false))
	}
	fmt.Fprintf(builder, "    record, err := r.ORM.%s().Update(ctx, model)\n", pluralName)
	fmt.Fprintf(builder, "    if err != nil {\n        return nil, err\n    }\n")
	fmt.Fprintf(builder, "    r.prime%[1]s(ctx, record)\n", ent.Name)
	fmt.Fprintf(builder, "    return &graphql.Update%[1]sPayload{\n", ent.Name)
	fmt.Fprintf(builder, "        ClientMutationID: input.ClientMutationID,\n")
	fmt.Fprintf(builder, "        %s: toGraphQL%s(record),\n", exportName(ent.Name), ent.Name)
	fmt.Fprintf(builder, "    }, nil\n}\n\n")

	fmt.Fprintf(builder, "func (r *mutationResolver) Delete%[1]s(ctx context.Context, input graphql.Delete%[1]sInput) (*graphql.Delete%[1]sPayload, error) {\n", ent.Name)
	fmt.Fprintf(builder, "    if r.ORM == nil {\n        return nil, fmt.Errorf(\"orm client is not configured\")\n    }\n")
	fmt.Fprintf(builder, "    nativeID, err := decode%[1]sID(input.ID)\n", ent.Name)
	fmt.Fprintf(builder, "    if err != nil {\n        return nil, err\n    }\n")
	fmt.Fprintf(builder, "    if err := r.ORM.%s().Delete(ctx, nativeID); err != nil {\n        return nil, err\n    }\n", pluralName)
	fmt.Fprintf(builder, "    return &graphql.Delete%[1]sPayload{\n", ent.Name)
	fmt.Fprintf(builder, "        ClientMutationID: input.ClientMutationID,\n")
	fmt.Fprintf(builder, "        Deleted%[1]sID: relay.ToGlobalID(\"%[1]s\", nativeID),\n", ent.Name)
	fmt.Fprintf(builder, "    }, nil\n}\n\n")

	return builder.String()
}

func renderInputAssignment(inputVar, targetVar string, field dsl.Field, includeID bool) string {
	if field.Name == "id" && !includeID {
		return ""
	}
	builder := &strings.Builder{}
	fieldName := exportName(field.Name)
	inputField := exportName(field.Name)
	if field.Name == "id" {
		fmt.Fprintf(builder, "    if %s.%s != nil {\n", inputVar, inputField)
		fmt.Fprintf(builder, "        %s.%s = *%s.%s\n", targetVar, fieldName, inputVar, inputField)
		fmt.Fprintf(builder, "    }\n")
		return builder.String()
	}
	if field.Type == dsl.TypeArray || field.Type == dsl.TypeVector {
		fmt.Fprintf(builder, "    if %s.%s != nil {\n", inputVar, inputField)
		fmt.Fprintf(builder, "        %s.%s = *%s.%s\n", targetVar, fieldName, inputVar, inputField)
		fmt.Fprintf(builder, "    }\n")
		return builder.String()
	}
	fmt.Fprintf(builder, "    if %s.%s != nil {\n", inputVar, inputField)
	fmt.Fprintf(builder, "        %s.%s = *%s.%s\n", targetVar, fieldName, inputVar, inputField)
	fmt.Fprintf(builder, "    }\n")
	return builder.String()
}

func writeGraphQLDataloaders(root string, entities []Entity) error {
	sort.Slice(entities, func(i, j int) bool { return entities[i].Name < entities[j].Name })
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "// Code generated by erm. DO NOT EDIT.\n")
	fmt.Fprintf(buf, "package dataloaders\n\n")

	imports := map[string]struct{}{
		"github.com/deicod/erm/internal/observability/metrics": {},
		"github.com/deicod/erm/internal/orm/gen":               {},
	}
	if len(entities) > 0 {
		imports["context"] = struct{}{}
	}
	if len(imports) > 0 {
		fmt.Fprintf(buf, "import (\n")
		keys := make([]string, 0, len(imports))
		for key := range imports {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(buf, "    \"%s\"\n", key)
		}
		fmt.Fprintf(buf, ")\n\n")
	}

	fmt.Fprintf(buf, "func configureEntityLoaders(loaders *Loaders, orm *gen.Client, collector metrics.Collector) {\n")
	fmt.Fprintf(buf, "    if loaders == nil || orm == nil {\n        return\n    }\n")
	for _, ent := range entities {
		plural := pluralize(ent.Name)
		fmt.Fprintf(buf, "    loaders.register(\"%[1]s\", newEntityLoader[string, *gen.%[1]s](\"%[2]s\", collector, func(ctx context.Context, keys []string) (map[string]*gen.%[1]s, error) {\n", ent.Name, plural)
		fmt.Fprintf(buf, "        results := make(map[string]*gen.%[1]s, len(keys))\n", ent.Name)
		fmt.Fprintf(buf, "        for _, key := range keys {\n")
		fmt.Fprintf(buf, "            record, err := orm.%[1]s().ByID(ctx, key)\n", exportName(plural))
		fmt.Fprintf(buf, "            if err != nil {\n                return nil, err\n            }\n")
		fmt.Fprintf(buf, "            if record != nil {\n                results[key] = record\n            }\n")
		fmt.Fprintf(buf, "        }\n        return results, nil\n    }))\n")
	}
	fmt.Fprintf(buf, "}\n\n")

	for _, ent := range entities {
		fmt.Fprintf(buf, "func (l *Loaders) %[1]s() *EntityLoader[string, *gen.%[1]s] {\n", ent.Name)
		fmt.Fprintf(buf, "    if l == nil {\n        return nil\n    }\n")
		fmt.Fprintf(buf, "    if loader, ok := l.get(\"%[1]s\").(*EntityLoader[string, *gen.%[1]s]); ok {\n", ent.Name)
		fmt.Fprintf(buf, "        return loader\n    }\n")
		fmt.Fprintf(buf, "    return nil\n}\n\n")
	}

	path := filepath.Join(root, "internal", "graphql", "dataloaders", "entities_gen.go")
	return writeGoFile(path, buf.Bytes())
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
