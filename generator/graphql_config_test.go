package generator

import (
	"fmt"
	"strings"
	"testing"
)

func TestDefaultGQLGenConfigIncludesScalarMappings(t *testing.T) {
	modulePath := "github.com/example/app"
	config := defaultGQLGenConfig(modulePath)
	mappings := GraphQLModelTypeMappings()

	for name, goTypes := range mappings {
		sectionHeader := fmt.Sprintf("  %s:\n    model:\n", name)
		if !strings.Contains(config, sectionHeader) {
			t.Fatalf("default gqlgen config missing models entry for %s", name)
		}
		for _, goType := range goTypes {
			entry := fmt.Sprintf("      - %s\n", goType)
			if !strings.Contains(config, entry) {
				t.Fatalf("default gqlgen config missing Go type %s for scalar %s", goType, name)
			}
		}
	}

	builtin := map[string]string{
		"Boolean": fmt.Sprintf("%s/graphql.Boolean", modulePath),
		"Float":   fmt.Sprintf("%s/graphql.Float", modulePath),
		"ID":      fmt.Sprintf("%s/graphql.ID", modulePath),
		"Int":     fmt.Sprintf("%s/graphql.Int", modulePath),
		"String":  fmt.Sprintf("%s/graphql.String", modulePath),
	}
	for name, goType := range builtin {
		sectionHeader := fmt.Sprintf("  %s:\n    model:\n", name)
		if !strings.Contains(config, sectionHeader) {
			t.Fatalf("default gqlgen config missing built-in scalar %s", name)
		}
		entry := fmt.Sprintf("      - %s\n", goType)
		if !strings.Contains(config, entry) {
			t.Fatalf("default gqlgen config missing Go type %s for built-in scalar %s", goType, name)
		}
	}

	for name, goType := range map[string]string{
		"__Directive":         "github.com/99designs/gqlgen/graphql/introspection.Directive",
		"__DirectiveLocation": fmt.Sprintf("%s/graphql.String", modulePath),
		"__EnumValue":         "github.com/99designs/gqlgen/graphql/introspection.EnumValue",
		"__Field":             "github.com/99designs/gqlgen/graphql/introspection.Field",
		"__InputValue":        "github.com/99designs/gqlgen/graphql/introspection.InputValue",
		"__Schema":            "github.com/99designs/gqlgen/graphql/introspection.Schema",
		"__Type":              "github.com/99designs/gqlgen/graphql/introspection.Type",
		"__TypeKind":          fmt.Sprintf("%s/graphql.String", modulePath),
	} {
		sectionHeader := fmt.Sprintf("  %s:\n    model:\n", name)
		if !strings.Contains(config, sectionHeader) {
			t.Fatalf("default gqlgen config missing introspection type %s", name)
		}
		entry := fmt.Sprintf("      - %s\n", goType)
		if !strings.Contains(config, entry) {
			t.Fatalf("default gqlgen config missing Go type %s for introspection %s", goType, name)
		}
	}

	if strings.Contains(config, "autobind:") {
		t.Fatalf("default gqlgen config unexpectedly includes autobind block:\n%s", config)
	}
}

func TestGraphQLModelsSectionRendersExpectedMappings(t *testing.T) {
	modulePath := "github.com/example/app"
	section := GraphQLModelsSection(modulePath)
	mappings := GraphQLModelTypeMappings()

	for name, goTypes := range mappings {
		sectionHeader := fmt.Sprintf("  %s:\n    model:\n", name)
		if !strings.Contains(section, sectionHeader) {
			t.Fatalf("models section missing scalar %s", name)
		}
		for _, goType := range goTypes {
			entry := fmt.Sprintf("      - %s\n", goType)
			if !strings.Contains(section, entry) {
				t.Fatalf("models section missing Go type %s for scalar %s", goType, name)
			}
		}
	}
}

func TestGraphQLModelsSectionUsesModulePathForBuiltins(t *testing.T) {
	section := GraphQLModelsSection("github.com/example/app")
	for _, want := range []string{
		"      - github.com/example/app/graphql.Boolean\n",
		"      - github.com/example/app/graphql.Float\n",
		"      - github.com/example/app/graphql.ID\n",
		"      - github.com/example/app/graphql.Int\n",
		"      - github.com/example/app/graphql.String\n",
		"      - github.com/99designs/gqlgen/graphql/introspection.Directive\n",
		"      - github.com/99designs/gqlgen/graphql/introspection.EnumValue\n",
		"      - github.com/99designs/gqlgen/graphql/introspection.Field\n",
		"      - github.com/99designs/gqlgen/graphql/introspection.InputValue\n",
		"      - github.com/99designs/gqlgen/graphql/introspection.Schema\n",
		"      - github.com/99designs/gqlgen/graphql/introspection.Type\n",
	} {
		if !strings.Contains(section, want) {
			t.Fatalf("expected module-mapped type %q in section", want)
		}
	}
}
