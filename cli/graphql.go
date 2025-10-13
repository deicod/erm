package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deicod/erm/generator"
	"github.com/deicod/erm/templates"
	"github.com/spf13/cobra"
)

func newGraphQLInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "graphql init",
		Short: "Add gqlgen config and bootstrap Relay schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			graphqlDir := "graphql"
			if err := ensureDir(graphqlDir); err != nil {
				return wrapError(fmt.Sprintf("graphql init: create directory %s", graphqlDir), err, "Create the directory or run from the project root.", 1)
			}

			modulePath := detectModule(root)
			if modulePath == "" {
				return wrapError("graphql init: detect module path", errors.New("module path not configured"), "Set 'module' in erm.yaml or initialise go.mod before running this command.", 1)
			}

			runtimeFiles, err := renderGraphQLRuntimeTemplates(modulePath)
			if err != nil {
				return wrapError("graphql init: render runtime templates", err, "Report this issue to the erm maintainers.", 1)
			}

			files := map[string][]byte{
				filepath.Join(graphqlDir, "gqlgen.yml"):      []byte(renderGQLGenYAML(modulePath)),
				filepath.Join(graphqlDir, "schema.graphqls"): []byte(schemaGraphQLS),
			}

			for path, content := range runtimeFiles {
				files[path] = content
			}

			var paths []string
			for path := range files {
				paths = append(paths, path)
			}
			sort.Strings(paths)

			for _, path := range paths {
				content := files[path]
				if err := writeFileOnce(path, content); err != nil {
					return wrapError(fmt.Sprintf("graphql init: write %s", path), err, "Check that the GraphQL directory is writable.", 1)
				}
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Initialized gqlgen configuration and runtime packages.")
			return nil
		},
	}
	return cmd
}

func renderGQLGenYAML(modulePath string) string {
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
	builder.WriteString(generator.GraphQLModelsSection(modulePath))
	fmt.Fprintln(builder, "autobind:")
	fmt.Fprintf(builder, "  - %s/orm/gen\n", modulePath)
	return builder.String()
}

var schemaGraphQLS = `scalar Time

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
`

func renderGraphQLRuntimeTemplates(modulePath string) (map[string][]byte, error) {
	rendered, err := templates.RenderRuntimeScaffolds(modulePath)
	if err != nil {
		return nil, err
	}
	runtime := make(map[string][]byte, len(rendered))
	for path, content := range rendered {
		runtime[filepath.FromSlash(path)] = content
	}
	return runtime, nil
}
