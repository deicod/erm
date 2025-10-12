package generator

import (
	"fmt"
	"strings"
	"testing"
)

func TestDefaultGQLGenConfigIncludesScalarMappings(t *testing.T) {
	config := defaultGQLGenConfig
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
}

func TestGraphQLModelsSectionRendersExpectedMappings(t *testing.T) {
	section := GraphQLModelsSection()
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
