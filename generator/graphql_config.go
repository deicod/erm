package generator

import (
	"os"
	"path/filepath"
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

const defaultGQLGenConfig = `schema:
  - graphql/schema.graphqls
exec:
  filename: graphql/generated.go
model:
  filename: graphql/models_gen.go
resolver:
  layout: follow-schema
  dir: graphql/resolvers
  package: resolvers
models:
  ID:
    model:
      - github.com/99designs/gqlgen/graphql.ID
      - string
  Time:
    model:
      - time.Time
`
