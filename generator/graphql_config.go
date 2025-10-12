package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ensureGQLGenConfig(root string) (string, error) {
	path := filepath.Join(root, "graphql", "gqlgen.yml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if _, err := writeFile(path, []byte(defaultGQLGenConfig)); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	return path, nil
}

// graphqlModelTypeMappings defines the Go types gqlgen should use for custom scalars.
var graphqlModelTypeMappings = map[string][]string{
	"BitString": {
		"string",
	},
	"BigInt": {
		"int64",
	},
	"CIDR": {
		"string",
	},
	"Date": {
		"time.Time",
	},
	"DateRange": {
		"string",
	},
	"Decimal": {
		"string",
	},
	"ID": {
		"github.com/99designs/gqlgen/graphql.ID",
		"string",
	},
	"Inet": {
		"string",
	},
	"Int4Range": {
		"string",
	},
	"Int8Range": {
		"string",
	},
	"Interval": {
		"string",
	},
	"JSON": {
		"encoding/json.RawMessage",
	},
	"JSONB": {
		"encoding/json.RawMessage",
	},
	"MacAddr": {
		"string",
	},
	"MacAddr8": {
		"string",
	},
	"Money": {
		"string",
	},
	"NumRange": {
		"string",
	},
	"TSQuery": {
		"string",
	},
	"TSRange": {
		"string",
	},
	"TSVector": {
		"string",
	},
	"TSTZRange": {
		"string",
	},
	"Time": {
		"time.Time",
	},
	"Timetz": {
		"time.Time",
	},
	"Timestamp": {
		"time.Time",
	},
	"Timestamptz": {
		"time.Time",
	},
	"VarBitString": {
		"string",
	},
	"XML": {
		"string",
	},
}

var defaultGQLGenConfig = func() string {
	builder := &strings.Builder{}
	fmt.Fprintln(builder, "schema:")
	fmt.Fprintln(builder, "  - graphql/schema.graphqls")
	fmt.Fprintln(builder, "exec:")
	fmt.Fprintln(builder, "  filename: graphql/generated.go")
	fmt.Fprintln(builder, "model:")
	fmt.Fprintln(builder, "  filename: graphql/models_gen.go")
	fmt.Fprintln(builder, "resolver:")
	fmt.Fprintln(builder, "  layout: follow-schema")
	fmt.Fprintln(builder, "  dir: graphql/resolvers")
	fmt.Fprintln(builder, "  package: resolvers")
	builder.WriteString(GraphQLModelsSection())
	return builder.String()
}()

// GraphQLModelTypeMappings returns a defensive copy of the scalar-to-Go type mappings used
// when scaffolding gqlgen configuration.
func GraphQLModelTypeMappings() map[string][]string {
	result := make(map[string][]string, len(graphqlModelTypeMappings))
	for name, goTypes := range graphqlModelTypeMappings {
		copied := make([]string, len(goTypes))
		copy(copied, goTypes)
		result[name] = copied
	}
	return result
}

// GraphQLModelsSection renders the gqlgen `models` configuration block using the shared
// scalar mappings.
func GraphQLModelsSection() string {
	builder := &strings.Builder{}
	fmt.Fprintln(builder, "models:")

	keys := make([]string, 0, len(graphqlModelTypeMappings))
	for key := range graphqlModelTypeMappings {
		if key == "ID" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := make([]string, 0, len(graphqlModelTypeMappings))
	ordered = append(ordered, "ID")
	ordered = append(ordered, keys...)

	for _, name := range ordered {
		goTypes := graphqlModelTypeMappings[name]
		if len(goTypes) == 0 {
			continue
		}
		fmt.Fprintf(builder, "  %s:\n", name)
		fmt.Fprintln(builder, "    model:")
		for _, goType := range goTypes {
			fmt.Fprintf(builder, "      - %s\n", goType)
		}
	}

	return builder.String()
}
