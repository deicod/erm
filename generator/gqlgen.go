package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
)

func runGQLGen(root string) error {
	modulePath, err := detectModulePath(root)
	if err != nil {
		return err
	}
	if err := ensureGQLDependencies(root); err != nil {
		return err
	}
	cfgPath, err := ensureGQLGenConfig(root, modulePath)
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
	resolverStub := filepath.Join(root, "graphql", "resolvers", "schema.resolvers.go")
	_ = os.Remove(resolverStub)
	return nil
}

func ensureGQLDependencies(root string) error {
	deps := []string{
		"github.com/99designs/gqlgen",
		"github.com/vektah/gqlparser/v2",
	}
	env := append(os.Environ(), "GO111MODULE=on")
	for _, dep := range deps {
		cmd := exec.Command("go", "mod", "download", dep)
		cmd.Dir = root
		cmd.Env = env
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("go mod download %s: %w\n%s", dep, err, string(output))
		}
	}
	return nil
}
