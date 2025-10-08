package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var gqlInitCmd = &cobra.Command{
	Use:   "graphql init",
	Short: "Add gqlgen config and bootstrap Relay schema",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := os.WriteFile("internal/graphql/gqlgen.yml", []byte(gqlgenYAML), 0o644); err != nil { return err }
		if err := os.WriteFile("internal/graphql/schema.graphqls", []byte(schemaGraphQLS), 0o644); err != nil { return err }
		fmt.Println("Initialized gqlgen configuration.")
		return nil
	},
}

var gqlgenYAML = `schema:
  - internal/graphql/schema.graphqls
exec:
  filename: internal/graphql/generated.go
model:
  filename: internal/graphql/models_gen.go
resolver:
  layout: follow-schema
  dir: internal/graphql
autobind:
  - github.com/deicod/erm/internal/orm/gen
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
