package generator

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deicod/erm/orm/dsl"
)

func writeGraphQLArtifacts(root string, entities []Entity, modulePath string) error {
	if strings.TrimSpace(modulePath) == "" {
		return fmt.Errorf("module path is required to generate GraphQL artifacts")
	}
	if err := EnsureRuntimeScaffolds(root, modulePath); err != nil {
		return err
	}
	if err := writeGraphQLSchema(root, entities); err != nil {
		return err
	}
	if err := writeGraphQLResolvers(root, entities, modulePath); err != nil {
		return err
	}
	if err := writeGraphQLDataloaders(root, entities, modulePath); err != nil {
		return err
	}
	if err := ensureGraphQLScalarHelpers(root); err != nil {
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
	path := filepath.Join(root, "graphql", "schema.graphqls")
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

	enums := collectGraphQLEnums(entities)
	if len(enums) > 0 {
		names := make([]string, 0, len(enums))
		for name := range enums {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			builder.WriteString(fmt.Sprintf("enum %s {\n", name))
			for _, value := range enums[name] {
				builder.WriteString(fmt.Sprintf("  %s\n", value))
			}
			builder.WriteString("}\n\n")
		}
	}

	queryFields := make([]string, 0, len(entities)*2)
	mutationFields := make([]string, 0, len(entities)*3)
	subscriptionFields := make([]string, 0, len(entities)*3)
	for _, ent := range entities {
		builder.WriteString(renderEntityType(ent))
		builder.WriteString("\n")
		builder.WriteString(renderConnectionTypes(ent))
		builder.WriteString("\n")
		builder.WriteString(renderEntityInputTypes(ent))
		builder.WriteString("\n")
		queryFields = append(queryFields, renderEntityQueryFields(ent)...)
		mutationFields = append(mutationFields, renderEntityMutationFields(ent)...)
		subscriptionFields = append(subscriptionFields, renderEntitySubscriptionFields(ent)...)
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
	if len(subscriptionFields) > 0 {
		builder.WriteString("\nextend type Subscription {\n")
		for _, field := range subscriptionFields {
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
	rule := ent.Authorization.Read
	fields := []string{
		appendAuthDirective(fmt.Sprintf("%s(id: ID!): %s", singular, ent.Name), rule),
		appendAuthDirective(fmt.Sprintf("%s(first: Int, after: String, last: Int, before: String): %sConnection!", plural, ent.Name), rule),
	}
	return fields
}

func renderEntityMutationFields(ent Entity) []string {
	return []string{
		appendAuthDirective(fmt.Sprintf("create%s(input: Create%sInput!): Create%sPayload!", ent.Name, ent.Name, ent.Name), ent.Authorization.Create),
		appendAuthDirective(fmt.Sprintf("update%s(input: Update%sInput!): Update%sPayload!", ent.Name, ent.Name, ent.Name), ent.Authorization.Update),
		appendAuthDirective(fmt.Sprintf("delete%s(input: Delete%sInput!): Delete%sPayload!", ent.Name, ent.Name, ent.Name), ent.Authorization.Delete),
	}
}

func renderEntitySubscriptionFields(ent Entity) []string {
	events := entitySubscriptionEvents(ent)
	if len(events) == 0 {
		return nil
	}
	fields := make([]string, 0, len(events))
	prefix := lowerCamel(ent.Name)
	for _, event := range events {
		switch event {
		case dsl.SubscriptionEventCreate:
			fields = append(fields, appendAuthDirective(fmt.Sprintf("%sCreated: %s!", prefix, ent.Name), ent.Authorization.Create))
		case dsl.SubscriptionEventUpdate:
			fields = append(fields, appendAuthDirective(fmt.Sprintf("%sUpdated: %s!", prefix, ent.Name), ent.Authorization.Update))
		case dsl.SubscriptionEventDelete:
			fields = append(fields, appendAuthDirective(fmt.Sprintf("%sDeleted: ID!", prefix), ent.Authorization.Delete))
		}
	}
	return fields
}

func appendAuthDirective(def string, rule *dsl.AuthRule) string {
	directive := buildAuthDirective(rule)
	if directive == "" {
		return def
	}
	return def + " " + directive
}

func buildAuthDirective(rule *dsl.AuthRule) string {
	if rule == nil {
		return ""
	}
	requirement := rule.Requirement
	if requirement == "" || requirement == dsl.AuthRequirementPublic {
		return ""
	}
	if len(rule.Roles) == 0 {
		return "@auth"
	}
	roles := make([]string, 0, len(rule.Roles))
	for _, role := range rule.Roles {
		if strings.TrimSpace(role) == "" {
			continue
		}
		roles = append(roles, fmt.Sprintf("\"%s\"", role))
	}
	if len(roles) == 0 {
		return "@auth"
	}
	return fmt.Sprintf("@auth(roles: [%s])", strings.Join(roles, ", "))
}

func entitySubscriptionEvents(ent Entity) []dsl.SubscriptionEvent {
	if len(ent.Annotations) == 0 {
		return nil
	}
	events := make([]dsl.SubscriptionEvent, 0, 3)
	seen := map[dsl.SubscriptionEvent]struct{}{}
	for _, ann := range ent.Annotations {
		if strings.ToLower(ann.Name) != dsl.AnnotationGraphQL {
			continue
		}
		raw, ok := ann.Payload["subscriptions"]
		if !ok {
			continue
		}
		switch vals := raw.(type) {
		case []string:
			for _, val := range vals {
				if event, ok := normalizeSubscriptionEvent(val); ok {
					if _, exists := seen[event]; !exists {
						seen[event] = struct{}{}
						events = append(events, event)
					}
				}
			}
		case []any:
			for _, item := range vals {
				if event, ok := normalizeSubscriptionEvent(item); ok {
					if _, exists := seen[event]; !exists {
						seen[event] = struct{}{}
						events = append(events, event)
					}
				}
			}
		}
	}
	return events
}

func normalizeSubscriptionEvent(val any) (dsl.SubscriptionEvent, bool) {
	switch v := val.(type) {
	case dsl.SubscriptionEvent:
		if v == "" {
			return "", false
		}
		return v, true
	case string:
		key := strings.ToLower(v)
		switch key {
		case string(dsl.SubscriptionEventCreate), "subscriptioneventcreate":
			return dsl.SubscriptionEventCreate, true
		case string(dsl.SubscriptionEventUpdate), "subscriptioneventupdate":
			return dsl.SubscriptionEventUpdate, true
		case string(dsl.SubscriptionEventDelete), "subscriptioneventdelete":
			return dsl.SubscriptionEventDelete, true
		}
	}
	return "", false
}

func hasSubscriptionEvent(ent Entity, event dsl.SubscriptionEvent) bool {
	for _, candidate := range entitySubscriptionEvents(ent) {
		if candidate == event {
			return true
		}
	}
	return false
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
		if len(field.EnumValues) > 0 && field.EnumName != "" {
			return field.EnumName, nil
		}
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

type nullHelperUsage struct {
	HasSQLNull bool
	types      map[string]struct{}
}

func (usage nullHelperUsage) NeedsTime() bool {
	if usage.types == nil {
		return false
	}
	_, ok := usage.types["sql.NullTime"]
	return ok
}

func (usage nullHelperUsage) GoTypes() []string {
	if usage.types == nil {
		return nil
	}
	goTypes := make([]string, 0, len(usage.types))
	for typ := range usage.types {
		goTypes = append(goTypes, typ)
	}
	sort.Strings(goTypes)
	return goTypes
}

func collectNullHelperUsage(entities []Entity) nullHelperUsage {
	usage := nullHelperUsage{types: map[string]struct{}{}}
	for _, ent := range entities {
		for _, field := range ent.Fields {
			goType := defaultGoType(field)
			if strings.HasPrefix(goType, "sql.Null") {
				usage.HasSQLNull = true
				usage.types[goType] = struct{}{}
			}
		}
	}
	if len(usage.types) == 0 {
		usage.types = nil
	}
	return usage
}

func hasEnumFields(entities []Entity) bool {
	for _, ent := range entities {
		for _, field := range ent.Fields {
			if len(field.EnumValues) > 0 && field.EnumName != "" {
				return true
			}
		}
	}
	return false
}

func nullableHelperName(goType string) string {
	suffix := strings.TrimPrefix(goType, "sql.Null")
	if suffix == "" {
		return ""
	}
	return "nullable" + suffix
}

func sqlNullFieldName(goType string) string {
	switch goType {
	case "sql.NullBool":
		return "Bool"
	case "sql.NullFloat64":
		return "Float64"
	case "sql.NullInt16":
		return "Int16"
	case "sql.NullInt32":
		return "Int32"
	case "sql.NullInt64":
		return "Int64"
	case "sql.NullByte":
		return "Byte"
	case "sql.NullString":
		return "String"
	case "sql.NullTime":
		return "Time"
	default:
		return ""
	}
}

func sqlNullValueType(goType string) string {
	switch goType {
	case "sql.NullBool":
		return "bool"
	case "sql.NullFloat64":
		return "float64"
	case "sql.NullInt16", "sql.NullInt32":
		return "int"
	case "sql.NullInt64":
		return "int64"
	case "sql.NullByte":
		return "byte"
	case "sql.NullString":
		return "string"
	case "sql.NullTime":
		return "time.Time"
	default:
		return ""
	}
}

func sqlNullConversionExpr(goType string) string {
	field := sqlNullFieldName(goType)
	if field == "" {
		return ""
	}
	switch goType {
	case "sql.NullInt16", "sql.NullInt32":
		return fmt.Sprintf("int(input.%s)", field)
	default:
		return fmt.Sprintf("input.%s", field)
	}
}

func renderGraphQLEnumHelpers() string {
	builder := &strings.Builder{}
	builder.WriteString("func toGraphQLEnum[T ~string](value string) T {\n")
	builder.WriteString("    return T(value)\n}\n\n")
	builder.WriteString("func toGraphQLEnumPtr[T ~string](value *string) *T {\n")
	builder.WriteString("    if value == nil {\n        return nil\n    }\n")
	builder.WriteString("    out := T(*value)\n")
	builder.WriteString("    return &out\n}\n\n")
	builder.WriteString("func fromGraphQLEnum[T ~string](value T) string {\n")
	builder.WriteString("    return string(value)\n}\n\n")
	builder.WriteString("func fromGraphQLEnumPtr[T ~string](value *T) *string {\n")
	builder.WriteString("    if value == nil {\n        return nil\n    }\n")
	builder.WriteString("    out := string(*value)\n")
	builder.WriteString("    return &out\n}\n\n")
	return builder.String()
}

func writeGraphQLResolvers(root string, entities []Entity, modulePath string) error {
	sort.Slice(entities, func(i, j int) bool { return entities[i].Name < entities[j].Name })
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "// Code generated by erm. DO NOT EDIT.\n")
	fmt.Fprintf(buf, "package resolvers\n\n")

	if err := ensureGraphQLEntityHooksFile(root); err != nil {
		return err
	}

	helperUsage := collectNullHelperUsage(entities)
	hasEnums := hasEnumFields(entities)

	imports := map[string]struct{}{
		"context":                             {},
		"fmt":                                 {},
		fmt.Sprintf("%s/graphql", modulePath): {},
		fmt.Sprintf("%s/graphql/relay", modulePath): {},
	}
	if len(entities) > 0 {
		imports[fmt.Sprintf("%s/graphql/dataloaders", modulePath)] = struct{}{}
		imports[fmt.Sprintf("%s/orm/gen", modulePath)] = struct{}{}
	}
	if helperUsage.HasSQLNull {
		imports["database/sql"] = struct{}{}
	}
	if helperUsage.NeedsTime() {
		imports["time"] = struct{}{}
	}
	if needsGraphQLIntRangeCheck(entities) {
		imports["math"] = struct{}{}
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

	buf.WriteString(renderEntityHooksSupport(entities))
	buf.WriteString(renderNodeResolver(entities))
	if len(entities) > 0 {
		for _, ent := range entities {
			buf.WriteString(renderEntityHelpers(ent))
			buf.WriteString("\n")
			buf.WriteString(renderEntityQueryResolvers(ent))
			buf.WriteString("\n")
			buf.WriteString(renderEntityMutationResolvers(ent))
			buf.WriteString("\n")
			buf.WriteString(renderEntitySubscriptionResolvers(ent))
			buf.WriteString("\n")
		}
	}

	if hasEnums {
		buf.WriteString(renderGraphQLEnumHelpers())
	}
	needsIntPtrHelper := needsGraphQLIntPointerHelper(entities)
	if helperUsage.HasSQLNull {
		for _, goType := range helperUsage.GoTypes() {
			helper := nullableHelperName(goType)
			valueType := sqlNullValueType(goType)
			field := sqlNullFieldName(goType)
			conversion := sqlNullConversionExpr(goType)
			if helper == "" || valueType == "" || field == "" || conversion == "" {
				continue
			}
			fmt.Fprintf(buf, "func %s(input %s) *%s {\n", helper, goType, valueType)
			fmt.Fprintf(buf, "    if !input.Valid {\n        return nil\n    }\n")
			fmt.Fprintf(buf, "    value := %s\n", conversion)
			fmt.Fprintf(buf, "    return &value\n}\n\n")
		}
	}
	if needsIntPtrHelper {
		buf.WriteString(renderGraphQLIntPointerHelper())
	}

	path := filepath.Join(root, "graphql", "resolvers", "entities_gen.go")
	return writeGoFile(path, buf.Bytes())
}

func needsGraphQLIntPointerHelper(entities []Entity) bool {
	for _, ent := range entities {
		for _, field := range ent.Fields {
			goType := defaultGoType(field)
			if goType == "*int16" || goType == "*int32" {
				return true
			}
		}
	}
	return false
}

func needsGraphQLIntRangeCheck(entities []Entity) bool {
	for _, ent := range entities {
		for _, field := range ent.Fields {
			if requiresSmallIntRangeCheck(field) {
				return true
			}
		}
	}
	return false
}

func renderGraphQLIntPointerHelper() string {
	return strings.TrimSpace(`func toGraphQLIntPtr[T ~int16 | ~int32](input *T) *int {
    if input == nil {
        return nil
    }
    value := int(*input)
    return &value
}`) + "\n\n"
}

func ensureGraphQLEntityHooksFile(root string) error {
	path := filepath.Join(root, "graphql", "resolvers", "entities_hooks.go")
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	stub := []byte(strings.TrimSpace(`package resolvers

// newEntityHooks configures optional GraphQL entity hooks.
//
// Edit this file to inject custom behaviour before mutating or returning
// records from generated resolvers. Return an entityHooks struct with the
// desired callbacks populated. The zero value performs no additional work.
func newEntityHooks() entityHooks {
        return entityHooks{}
}`) + "\n")
	_, err := writeFile(path, stub)
	return err
}

func renderEntityHooksSupport(entities []Entity) string {
	if len(entities) == 0 {
		return ""
	}
	builder := &strings.Builder{}
	builder.WriteString("type entityHooks struct {\n")
	for _, ent := range entities {
		name := ent.Name
		fmt.Fprintf(builder, "    BeforeCreate%[1]s func(ctx context.Context, r *Resolver, input graphql.Create%[1]sInput, model *gen.%[1]s) error\n", name)
		fmt.Fprintf(builder, "    AfterCreate%[1]s func(ctx context.Context, r *Resolver, record *gen.%[1]s) error\n", name)
		fmt.Fprintf(builder, "    BeforeUpdate%[1]s func(ctx context.Context, r *Resolver, input graphql.Update%[1]sInput, model *gen.%[1]s) error\n", name)
		fmt.Fprintf(builder, "    AfterUpdate%[1]s func(ctx context.Context, r *Resolver, record *gen.%[1]s) error\n", name)
		fmt.Fprintf(builder, "    BeforeDelete%[1]s func(ctx context.Context, r *Resolver, input graphql.Delete%[1]sInput, id string) error\n", name)
		fmt.Fprintf(builder, "    AfterDelete%[1]s func(ctx context.Context, r *Resolver, input graphql.Delete%[1]sInput, id string) error\n", name)
		fmt.Fprintf(builder, "    BeforeReturn%[1]s func(ctx context.Context, r *Resolver, record *gen.%[1]s) error\n", name)
	}
	builder.WriteString("}\n\n")
	for _, ent := range entities {
		builder.WriteString(renderEntityHookHelpers(ent))
	}
	return builder.String()
}

func renderEntityHookHelpers(ent Entity) string {
	builder := &strings.Builder{}
	name := ent.Name
	fmt.Fprintf(builder, "func (r *Resolver) applyBeforeCreate%[1]s(ctx context.Context, input graphql.Create%[1]sInput, model *gen.%[1]s) error {\n", name)
	builder.WriteString("    if r == nil || r.hooks.BeforeCreate" + name + " == nil {\n        return nil\n    }\n")
	fmt.Fprintf(builder, "    return r.hooks.BeforeCreate%[1]s(ctx, r, input, model)\n}\n\n", name)

	fmt.Fprintf(builder, "func (r *Resolver) applyAfterCreate%[1]s(ctx context.Context, record *gen.%[1]s) error {\n", name)
	builder.WriteString("    if r == nil || record == nil || r.hooks.AfterCreate" + name + " == nil {\n        return nil\n    }\n")
	fmt.Fprintf(builder, "    return r.hooks.AfterCreate%[1]s(ctx, r, record)\n}\n\n", name)

	fmt.Fprintf(builder, "func (r *Resolver) applyBeforeUpdate%[1]s(ctx context.Context, input graphql.Update%[1]sInput, model *gen.%[1]s) error {\n", name)
	builder.WriteString("    if r == nil || r.hooks.BeforeUpdate" + name + " == nil {\n        return nil\n    }\n")
	fmt.Fprintf(builder, "    return r.hooks.BeforeUpdate%[1]s(ctx, r, input, model)\n}\n\n", name)

	fmt.Fprintf(builder, "func (r *Resolver) applyAfterUpdate%[1]s(ctx context.Context, record *gen.%[1]s) error {\n", name)
	builder.WriteString("    if r == nil || record == nil || r.hooks.AfterUpdate" + name + " == nil {\n        return nil\n    }\n")
	fmt.Fprintf(builder, "    return r.hooks.AfterUpdate%[1]s(ctx, r, record)\n}\n\n", name)

	fmt.Fprintf(builder, "func (r *Resolver) applyBeforeDelete%[1]s(ctx context.Context, input graphql.Delete%[1]sInput, id string) error {\n", name)
	builder.WriteString("    if r == nil || r.hooks.BeforeDelete" + name + " == nil {\n        return nil\n    }\n")
	fmt.Fprintf(builder, "    return r.hooks.BeforeDelete%[1]s(ctx, r, input, id)\n}\n\n", name)

	fmt.Fprintf(builder, "func (r *Resolver) applyAfterDelete%[1]s(ctx context.Context, input graphql.Delete%[1]sInput, id string) error {\n", name)
	builder.WriteString("    if r == nil || r.hooks.AfterDelete" + name + " == nil {\n        return nil\n    }\n")
	fmt.Fprintf(builder, "    return r.hooks.AfterDelete%[1]s(ctx, r, input, id)\n}\n\n", name)

	fmt.Fprintf(builder, "func (r *Resolver) applyBeforeReturn%[1]s(ctx context.Context, record *gen.%[1]s) error {\n", name)
	builder.WriteString("    if r == nil || record == nil || r.hooks.BeforeReturn" + name + " == nil {\n        return nil\n    }\n")
	fmt.Fprintf(builder, "    return r.hooks.BeforeReturn%[1]s(ctx, r, record)\n}\n\n", name)

	return builder.String()
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
		fmt.Fprintf(builder, "        if err := r.applyBeforeReturn%[1]s(ctx, record); err != nil {\n            return nil, err\n        }\n", ent.Name)
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
		fmt.Fprintf(builder, "        %s: %s,\n", fieldName, graphqlFieldValue(field))
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

func graphqlFieldValue(field dsl.Field) string {
	base := fmt.Sprintf("record.%s", exportName(field.Name))
	goType := defaultGoType(field)
	if len(field.EnumValues) > 0 && field.EnumName != "" {
		enumType := fmt.Sprintf("graphql.%s", field.EnumName)
		if strings.HasPrefix(goType, "*") {
			return fmt.Sprintf("toGraphQLEnumPtr[%s](%s)", enumType, base)
		}
		return fmt.Sprintf("toGraphQLEnum[%s](%s)", enumType, base)
	}
	if strings.HasPrefix(goType, "sql.Null") {
		helper := nullableHelperName(goType)
		if helper != "" {
			return fmt.Sprintf("%s(%s)", helper, base)
		}
	}
	if isGraphQLIntType(field.Type) {
		switch goType {
		case "int16", "int32":
			return fmt.Sprintf("int(%s)", base)
		case "*int16", "*int32":
			return fmt.Sprintf("toGraphQLIntPtr(%s)", base)
		}
	}
	return base
}

func isGraphQLIntType(ft dsl.FieldType) bool {
	switch ft {
	case dsl.TypeSmallInt, dsl.TypeInteger, dsl.TypeSmallSerial, dsl.TypeSerial:
		return true
	default:
		return false
	}
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
	fmt.Fprintf(builder, "    if err := r.applyBeforeReturn%[1]s(ctx, record); err != nil {\n        return nil, err\n    }\n", ent.Name)
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
	fmt.Fprintf(builder, "        if err := r.applyBeforeReturn%[1]s(ctx, record); err != nil {\n            return nil, err\n        }\n", ent.Name)
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
	fmt.Fprintf(builder, "    if err := r.applyBeforeCreate%[1]s(ctx, input, model); err != nil {\n        return nil, err\n    }\n", ent.Name)
	fmt.Fprintf(builder, "    record, err := r.ORM.%s().Create(ctx, model)\n", pluralName)
	fmt.Fprintf(builder, "    if err != nil {\n        return nil, err\n    }\n")
	fmt.Fprintf(builder, "    if err := r.applyAfterCreate%[1]s(ctx, record); err != nil {\n        return nil, err\n    }\n", ent.Name)
	fmt.Fprintf(builder, "    if err := r.applyBeforeReturn%[1]s(ctx, record); err != nil {\n        return nil, err\n    }\n", ent.Name)
	fmt.Fprintf(builder, "    gqlRecord := toGraphQL%[1]s(record)\n", ent.Name)
	fmt.Fprintf(builder, "    r.prime%[1]s(ctx, record)\n", ent.Name)
	if hasSubscriptionEvent(ent, dsl.SubscriptionEventCreate) {
		fmt.Fprintf(builder, "    publishSubscriptionEvent(ctx, r.subscriptionBroker(), \"%s\", SubscriptionTriggerCreated, gqlRecord)\n", ent.Name)
	}
	fmt.Fprintf(builder, "    return &graphql.Create%[1]sPayload{\n", ent.Name)
	fmt.Fprintf(builder, "        ClientMutationID: input.ClientMutationID,\n")
	fmt.Fprintf(builder, "        %s: gqlRecord,\n", exportName(ent.Name))
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
	fmt.Fprintf(builder, "    if err := r.applyBeforeUpdate%[1]s(ctx, input, model); err != nil {\n        return nil, err\n    }\n", ent.Name)
	fmt.Fprintf(builder, "    record, err := r.ORM.%s().Update(ctx, model)\n", pluralName)
	fmt.Fprintf(builder, "    if err != nil {\n        return nil, err\n    }\n")
	fmt.Fprintf(builder, "    if err := r.applyAfterUpdate%[1]s(ctx, record); err != nil {\n        return nil, err\n    }\n", ent.Name)
	fmt.Fprintf(builder, "    if err := r.applyBeforeReturn%[1]s(ctx, record); err != nil {\n        return nil, err\n    }\n", ent.Name)
	fmt.Fprintf(builder, "    gqlRecord := toGraphQL%[1]s(record)\n", ent.Name)
	fmt.Fprintf(builder, "    r.prime%[1]s(ctx, record)\n", ent.Name)
	if hasSubscriptionEvent(ent, dsl.SubscriptionEventUpdate) {
		fmt.Fprintf(builder, "    publishSubscriptionEvent(ctx, r.subscriptionBroker(), \"%s\", SubscriptionTriggerUpdated, gqlRecord)\n", ent.Name)
	}
	fmt.Fprintf(builder, "    return &graphql.Update%[1]sPayload{\n", ent.Name)
	fmt.Fprintf(builder, "        ClientMutationID: input.ClientMutationID,\n")
	fmt.Fprintf(builder, "        %s: gqlRecord,\n", exportName(ent.Name))
	fmt.Fprintf(builder, "    }, nil\n}\n\n")

	fmt.Fprintf(builder, "func (r *mutationResolver) Delete%[1]s(ctx context.Context, input graphql.Delete%[1]sInput) (*graphql.Delete%[1]sPayload, error) {\n", ent.Name)
	fmt.Fprintf(builder, "    if r.ORM == nil {\n        return nil, fmt.Errorf(\"orm client is not configured\")\n    }\n")
	fmt.Fprintf(builder, "    nativeID, err := decode%[1]sID(input.ID)\n", ent.Name)
	fmt.Fprintf(builder, "    if err != nil {\n        return nil, err\n    }\n")
	fmt.Fprintf(builder, "    if err := r.applyBeforeDelete%[1]s(ctx, input, nativeID); err != nil {\n        return nil, err\n    }\n", ent.Name)
	fmt.Fprintf(builder, "    if err := r.ORM.%s().Delete(ctx, nativeID); err != nil {\n        return nil, err\n    }\n", pluralName)
	fmt.Fprintf(builder, "    if err := r.applyAfterDelete%[1]s(ctx, input, nativeID); err != nil {\n        return nil, err\n    }\n", ent.Name)
	fmt.Fprintf(builder, "    deletedID := relay.ToGlobalID(\"%[1]s\", nativeID)\n", ent.Name)
	if hasSubscriptionEvent(ent, dsl.SubscriptionEventDelete) {
		fmt.Fprintf(builder, "    publishSubscriptionEvent(ctx, r.subscriptionBroker(), \"%s\", SubscriptionTriggerDeleted, deletedID)\n", ent.Name)
	}
	fmt.Fprintf(builder, "    return &graphql.Delete%[1]sPayload{\n", ent.Name)
	fmt.Fprintf(builder, "        ClientMutationID: input.ClientMutationID,\n")
	fmt.Fprintf(builder, "        Deleted%[1]sID: deletedID,\n", ent.Name)
	fmt.Fprintf(builder, "    }, nil\n}\n\n")

	return builder.String()
}

func renderEntitySubscriptionResolvers(ent Entity) string {
	events := entitySubscriptionEvents(ent)
	if len(events) == 0 {
		return ""
	}
	builder := &strings.Builder{}
	for _, event := range events {
		switch event {
		case dsl.SubscriptionEventCreate:
			writeEntitySubscriptionResolver(builder, ent, event, fmt.Sprintf("%sCreated", exportName(ent.Name)), fmt.Sprintf("*graphql.%s", ent.Name))
		case dsl.SubscriptionEventUpdate:
			writeEntitySubscriptionResolver(builder, ent, event, fmt.Sprintf("%sUpdated", exportName(ent.Name)), fmt.Sprintf("*graphql.%s", ent.Name))
		case dsl.SubscriptionEventDelete:
			writeEntitySubscriptionResolver(builder, ent, event, fmt.Sprintf("%sDeleted", exportName(ent.Name)), "string")
		}
	}
	return builder.String()
}

func writeEntitySubscriptionResolver(builder *strings.Builder, ent Entity, event dsl.SubscriptionEvent, methodName, channelType string) {
	fmt.Fprintf(builder, "func (r *subscriptionResolver) %s(ctx context.Context) (<-chan %s, error) {\n", methodName, channelType)
	fmt.Fprintf(builder, "    stream, stop, err := subscribeToEntity(ctx, r.subscriptionBroker(), \"%s\", %s)\n", ent.Name, subscriptionTriggerLiteral(event))
	fmt.Fprintf(builder, "    if err != nil {\n        return nil, err\n    }\n")
	fmt.Fprintf(builder, "    out := make(chan %s, 1)\n", channelType)
	fmt.Fprintf(builder, "    go func() {\n")
	fmt.Fprintf(builder, "        defer close(out)\n")
	fmt.Fprintf(builder, "        if stop != nil {\n            defer stop()\n        }\n")
	fmt.Fprintf(builder, "        for {\n")
	fmt.Fprintf(builder, "            select {\n")
	fmt.Fprintf(builder, "            case <-ctx.Done():\n                return\n")
	fmt.Fprintf(builder, "            case payload, ok := <-stream:\n")
	fmt.Fprintf(builder, "                if !ok {\n                    return\n                }\n")
	switch event {
	case dsl.SubscriptionEventDelete:
		fmt.Fprintf(builder, "                value, ok := payload.(string)\n")
		fmt.Fprintf(builder, "                if !ok || value == \"\" {\n                    continue\n                }\n")
	default:
		fmt.Fprintf(builder, "                obj, ok := payload.(%s)\n", channelType)
		fmt.Fprintf(builder, "                if !ok || obj == nil {\n                    continue\n                }\n")
	}
	fmt.Fprintf(builder, "                select {\n")
	switch event {
	case dsl.SubscriptionEventDelete:
		fmt.Fprintf(builder, "                case out <- value:\n                    continue\n")
	default:
		fmt.Fprintf(builder, "                case out <- obj:\n                    continue\n")
	}
	fmt.Fprintf(builder, "                case <-ctx.Done():\n                    return\n                }\n")
	fmt.Fprintf(builder, "            }\n        }\n    }()\n")
	fmt.Fprintf(builder, "    return out, nil\n}\n\n")
}

func subscriptionTriggerLiteral(event dsl.SubscriptionEvent) string {
	switch event {
	case dsl.SubscriptionEventCreate:
		return "SubscriptionTriggerCreated"
	case dsl.SubscriptionEventUpdate:
		return "SubscriptionTriggerUpdated"
	case dsl.SubscriptionEventDelete:
		return "SubscriptionTriggerDeleted"
	default:
		return "SubscriptionTriggerCreated"
	}
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
	customGoType := hasCustomGoType(field)
	if len(field.EnumValues) > 0 && field.EnumName != "" {
		fmt.Fprintf(builder, "    if %s.%s != nil {\n", inputVar, inputField)
		enumType := fmt.Sprintf("graphql.%s", field.EnumName)
		goType := defaultGoType(field)
		if strings.HasPrefix(goType, "*") {
			if customGoType {
				baseType := baseGoType(field)
				for strings.HasPrefix(baseType, "*") {
					baseType = strings.TrimPrefix(baseType, "*")
				}
				if baseType == "" {
					baseType = strings.TrimPrefix(goType, "*")
				}
				fmt.Fprintf(builder, "        value := %s(fromGraphQLEnum[%s](*%s.%s))\n", baseType, enumType, inputVar, inputField)
				fmt.Fprintf(builder, "        %s.%s = &value\n", targetVar, fieldName)
			} else {
				fmt.Fprintf(builder, "        %s.%s = fromGraphQLEnumPtr[%s](%s.%s)\n", targetVar, fieldName, enumType, inputVar, inputField)
			}
		} else {
			valueExpr := fmt.Sprintf("fromGraphQLEnum[%s](*%s.%s)", enumType, inputVar, inputField)
			if customGoType {
				fmt.Fprintf(builder, "        %s.%s = %s(%s)\n", targetVar, fieldName, goType, valueExpr)
			} else {
				fmt.Fprintf(builder, "        %s.%s = %s\n", targetVar, fieldName, valueExpr)
			}
		}
		fmt.Fprintf(builder, "    }\n")
		return builder.String()
	}
	goType := defaultGoType(field)
	fmt.Fprintf(builder, "    if %s.%s != nil {\n", inputVar, inputField)
	if renderGraphQLIntInputAssignment(builder, inputVar, inputField, targetVar, fieldName, field, goType, customGoType) {
		fmt.Fprintf(builder, "    }\n")
		return builder.String()
	}
	if field.Type == dsl.TypeJSON || field.Type == dsl.TypeJSONB {
		fmt.Fprintf(builder, "        %s.%s = %s.%s\n", targetVar, fieldName, inputVar, inputField)
		fmt.Fprintf(builder, "    }\n")
		return builder.String()
	}
	switch {
	case strings.HasPrefix(goType, "*"):
		if customGoType {
			baseType := baseGoType(field)
			for strings.HasPrefix(baseType, "*") {
				baseType = strings.TrimPrefix(baseType, "*")
			}
			if baseType == "" {
				baseType = strings.TrimPrefix(goType, "*")
			}
			if baseType == "" {
				fmt.Fprintf(builder, "        %s.%s = %s.%s\n", targetVar, fieldName, inputVar, inputField)
			} else {
				fmt.Fprintf(builder, "        value := %s(*%s.%s)\n", baseType, inputVar, inputField)
				fmt.Fprintf(builder, "        %s.%s = &value\n", targetVar, fieldName)
			}
		} else {
			fmt.Fprintf(builder, "        %s.%s = %s.%s\n", targetVar, fieldName, inputVar, inputField)
		}
	case strings.HasPrefix(goType, "sql.Null"):
		field := sqlNullFieldName(goType)
		if field == "" {
			fmt.Fprintf(builder, "        %s.%s = *%s.%s\n", targetVar, fieldName, inputVar, inputField)
		} else {
			fmt.Fprintf(builder, "        %s.%s = %s{%s: *%s.%s, Valid: true}\n", targetVar, fieldName, goType, field, inputVar, inputField)
		}
	default:
		if customGoType {
			fmt.Fprintf(builder, "        %s.%s = %s(*%s.%s)\n", targetVar, fieldName, goType, inputVar, inputField)
		} else {
			fmt.Fprintf(builder, "        %s.%s = *%s.%s\n", targetVar, fieldName, inputVar, inputField)
		}
	}
	fmt.Fprintf(builder, "    }\n")
	return builder.String()
}

func renderGraphQLIntInputAssignment(builder *strings.Builder, inputVar, inputField, targetVar, fieldName string, field dsl.Field, goType string, customGoType bool) bool {
	if !isGraphQLIntType(field.Type) {
		return false
	}

	baseType := baseGoType(field)
	for strings.HasPrefix(baseType, "*") {
		baseType = strings.TrimPrefix(baseType, "*")
	}
	if baseType == "" {
		baseType = strings.TrimPrefix(goType, "*")
	}
	if baseType == "" {
		baseType = goType
	}

	if requiresSmallIntRangeCheck(field) {
		graphqlFieldName := lowerCamel(field.Name)
		if graphqlFieldName == "" {
			graphqlFieldName = fieldName
		}
		if graphqlFieldName == "" {
			graphqlFieldName = field.Name
		}
		if graphqlFieldName == "" {
			graphqlFieldName = "field"
		}
		message := fmt.Sprintf("%s must be between %%d and %%d", graphqlFieldName)
		fmt.Fprintf(builder, "        if *%s.%s < math.MinInt16 || *%s.%s > math.MaxInt16 {\n", inputVar, inputField, inputVar, inputField)
		fmt.Fprintf(builder, "            return nil, fmt.Errorf(\"%s\", math.MinInt16, math.MaxInt16)\n", message)
		fmt.Fprintf(builder, "        }\n")
	}

	switch {
	case strings.HasPrefix(goType, "*"):
		pointerType := strings.TrimPrefix(goType, "*")
		if pointerType == "" {
			pointerType = baseType
		}
		conversion := fmt.Sprintf("%s(*%s.%s)", pointerType, inputVar, inputField)
		if customGoType && pointerType != baseType && baseType != "" {
			conversion = fmt.Sprintf("%s(%s(*%s.%s))", pointerType, baseType, inputVar, inputField)
		}
		fmt.Fprintf(builder, "        value := %s\n", conversion)
		fmt.Fprintf(builder, "        %s.%s = &value\n", targetVar, fieldName)
		return true
	case strings.HasPrefix(goType, "sql.Null"):
		sqlField := sqlNullFieldName(goType)
		if sqlField == "" {
			conversion := fmt.Sprintf("%s(*%s.%s)", goType, inputVar, inputField)
			fmt.Fprintf(builder, "        %s.%s = %s\n", targetVar, fieldName, conversion)
			return true
		}
		conversion := fmt.Sprintf("%s(*%s.%s)", baseType, inputVar, inputField)
		fmt.Fprintf(builder, "        %s.%s = %s{%s: %s, Valid: true}\n", targetVar, fieldName, goType, sqlField, conversion)
		return true
	default:
		conversion := fmt.Sprintf("%s(*%s.%s)", goType, inputVar, inputField)
		if customGoType && goType != baseType && baseType != "" {
			conversion = fmt.Sprintf("%s(%s(*%s.%s))", goType, baseType, inputVar, inputField)
		}
		fmt.Fprintf(builder, "        %s.%s = %s\n", targetVar, fieldName, conversion)
		return true
	}
}

func requiresSmallIntRangeCheck(field dsl.Field) bool {
	switch field.Type {
	case dsl.TypeSmallInt, dsl.TypeSmallSerial:
		return true
	default:
		return false
	}
}

func hasCustomGoType(field dsl.Field) bool {
	if field.GoType == "" {
		return false
	}
	clone := field
	clone.GoType = ""
	return baseGoType(clone) != field.GoType
}
func writeGraphQLDataloaders(root string, entities []Entity, modulePath string) error {
	sort.Slice(entities, func(i, j int) bool { return entities[i].Name < entities[j].Name })
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "// Code generated by erm. DO NOT EDIT.\n")
	fmt.Fprintf(buf, "package dataloaders\n\n")

	imports := map[string]struct{}{
		fmt.Sprintf("%s/observability/metrics", modulePath): {},
		fmt.Sprintf("%s/orm/gen", modulePath):               {},
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

	path := filepath.Join(root, "graphql", "dataloaders", "entities_gen.go")
	return writeGoFile(path, buf.Bytes())
}

var predeclaredScalars = func() map[string]struct{} {
	scalars := map[string]struct{}{
		"Boolean": {},
		"Float":   {},
		"ID":      {},
		"Int":     {},
		"String":  {},
	}

	scanner := bufio.NewScanner(strings.NewReader(graphqlBaseSchema))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "scalar ") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(line, "scalar "))
		if name == "" {
			continue
		}
		scalars[name] = struct{}{}
	}

	return scalars
}()

func collectGraphQLEnums(entities []Entity) map[string][]string {
	enums := make(map[string][]string)
	for _, ent := range entities {
		for _, field := range ent.Fields {
			if len(field.EnumValues) == 0 || field.EnumName == "" {
				continue
			}
			if existing, ok := enums[field.EnumName]; ok {
				if !equalStringSlices(existing, field.EnumValues) {
					panic(fmt.Sprintf("conflicting enum values for %s", field.EnumName))
				}
				continue
			}
			enums[field.EnumName] = append([]string(nil), field.EnumValues...)
		}
	}
	return enums
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
}

type Subscription {
  _noop: Boolean
}`
