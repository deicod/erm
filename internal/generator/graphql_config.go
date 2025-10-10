package generator

import (
	"os"
	"path/filepath"
)

func ensureGQLGenConfig(root string) (string, error) {
	path := filepath.Join(root, "internal", "graphql", "gqlgen.yml")
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
  - internal/graphql/schema.graphqls
exec:
  filename: internal/graphql/generated.go
model:
  filename: internal/graphql/models_gen.go
resolver:
  layout: follow-schema
  dir: internal/graphql/resolvers
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
