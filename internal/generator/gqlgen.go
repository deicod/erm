package generator

import (
	"os"
	"path/filepath"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
)

func runGQLGen(root string) error {
	cfgPath, err := ensureGQLGenConfig(root)
	if err != nil {
		return err
	}
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return err
	}
	cfg.SkipValidation = true
	if err := api.Generate(cfg); err != nil {
		return err
	}
	resolverStub := filepath.Join(root, "internal", "graphql", "resolvers", "schema.resolvers.go")
	_ = os.Remove(resolverStub)
	return nil
}
