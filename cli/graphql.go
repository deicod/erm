package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newGraphQLInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "graphql init",
		Short: "Add gqlgen config and bootstrap Relay schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			graphqlDir := filepath.Join("internal", "graphql")
			if err := ensureDir(graphqlDir); err != nil {
				return wrapError(fmt.Sprintf("graphql init: create directory %s", graphqlDir), err, "Create the directory or run from the project root.", 1)
			}
			files := map[string]string{
				filepath.Join(graphqlDir, "gqlgen.yml"):      gqlgenYAML,
				filepath.Join(graphqlDir, "schema.graphqls"): schemaGraphQLS,
			}
			for path, content := range files {
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					return wrapError(fmt.Sprintf("graphql init: write %s", path), err, "Check that the GraphQL directory is writable.", 1)
				}
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Initialized gqlgen configuration.")
			return nil
		},
	}
	return cmd
}

var gqlgenYAML = `schema:
  - graphql/schema.graphqls
exec:
  filename: graphql/generated.go
model:
  filename: graphql/models_gen.go
resolver:
  layout: follow-schema
  dir: graphql/resolvers
  package: resolvers
autobind:
  - github.com/deicod/erm/orm/gen
`
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
