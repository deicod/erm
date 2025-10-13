package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
)

func runGQLGen(root string) error {
	if err := ensureGQLDependencies(root); err != nil {
		return err
	}
	return runGQLGenInternal(root)
}

func runGQLGenInternal(root string) error {
	modulePath, err := detectModulePath(root)
	if err != nil {
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
	tidyAttempted := false
	for _, dep := range deps {
		if err := downloadGQLDependency(root, env, dep, &tidyAttempted); err != nil {
			return err
		}
	}
	return nil
}

func downloadGQLDependency(root string, env []string, dep string, tidyAttempted *bool) error {
	output, err := runGoModDownload(root, env, dep)
	if err == nil {
		return nil
	}

	if !*tidyAttempted && strings.Contains(output, "not a known dependency") {
		tidyOut, tidyErr := runGoModTidy(root, env)
		*tidyAttempted = true
		if tidyErr != nil {
			return fmt.Errorf("go mod download %s: dependency missing and go mod tidy failed: %w\n%s", dep, tidyErr, tidyOut)
		}

		output, err = runGoModDownload(root, env, dep)
		if err == nil {
			return nil
		}
		return fmt.Errorf("go mod download %s failed after go mod tidy: %w\n%s", dep, err, output)
	}

	return fmt.Errorf("go mod download %s: %w\n%s", dep, err, output)
}

func runGoModDownload(root string, env []string, dep string) (string, error) {
	cmd := exec.Command("go", "mod", "download", dep)
	cmd.Dir = root
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runGoModTidy(root string, env []string) (string, error) {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = root
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	return string(output), err
}
