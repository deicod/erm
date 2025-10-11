package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Result captures the outcome of a single diagnostic check.
type Result struct {
	Name    string
	Status  Status
	Details string
}

type Status string

const (
	StatusOK    Status = "ok"
	StatusWarn  Status = "warn"
	StatusError Status = "error"
)

// Run executes the full suite of doctor checks.
var checks = []func(context.Context) Result{
	checkGoVersion,
	checkModuleFile,
	checkGqlGenDependency,
	checkPgxDependency,
	checkConfig,
	checkSchemaDir,
	checkGraphQLAssets,
}

func Run() []Result {
	ctx := context.Background()
	results := make([]Result, 0, len(checks))
	for _, check := range checks {
		results = append(results, check(ctx))
	}
	return results
}

func HasFailures(results []Result) bool {
	for _, res := range results {
		if res.Status == StatusError {
			return true
		}
	}
	return false
}

func checkGoVersion(context.Context) Result {
	path, err := exec.LookPath("go")
	if err != nil {
		return Result{Name: "Go toolchain", Status: StatusError, Details: "go binary not found in PATH"}
	}
	return Result{Name: "Go toolchain", Status: StatusOK, Details: fmt.Sprintf("%s (%s)", runtime.Version(), path)}
}

func checkModuleFile(context.Context) Result {
	_, err := os.Stat("go.mod")
	if err == nil {
		return Result{Name: "go.mod", Status: StatusOK}
	}
	if errors.Is(err, os.ErrNotExist) {
		return Result{Name: "go.mod", Status: StatusError, Details: "missing go.mod; run 'go mod init'"}
	}
	return Result{Name: "go.mod", Status: StatusError, Details: err.Error()}
}

func checkConfig(context.Context) Result {
	path := "erm.yaml"
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return Result{Name: "erm.yaml", Status: StatusError, Details: "expected file but found directory"}
		}
		return Result{Name: "erm.yaml", Status: StatusOK}
	}
	if errors.Is(err, os.ErrNotExist) {
		return Result{Name: "erm.yaml", Status: StatusWarn, Details: "config missing; run 'erm init'"}
	}
	return Result{Name: "erm.yaml", Status: StatusError, Details: err.Error()}
}

func checkSchemaDir(context.Context) Result {
	path := "schema"
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return Result{Name: "schema directory", Status: StatusError, Details: "schema exists but is not a directory"}
		}
		matches, err := filepath.Glob(filepath.Join(path, "*.schema.go"))
		if err != nil {
			return Result{Name: "schema directory", Status: StatusError, Details: err.Error()}
		}
		if len(matches) == 0 {
			return Result{Name: "schema directory", Status: StatusWarn, Details: "no *.schema.go files found; run 'erm new <Entity>'"}
		}
		return Result{Name: "schema directory", Status: StatusOK, Details: fmt.Sprintf("%d schema files", len(matches))}
	}
	if errors.Is(err, os.ErrNotExist) {
		return Result{Name: "schema directory", Status: StatusWarn, Details: "missing schema/; run 'erm init'"}
	}
	return Result{Name: "schema directory", Status: StatusError, Details: err.Error()}
}

func checkGqlGenDependency(context.Context) Result {
	ok, err := moduleListed("github.com/99designs/gqlgen")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{Name: "gqlgen dependency", Status: StatusWarn, Details: "go.mod missing"}
		}
		return Result{Name: "gqlgen dependency", Status: StatusError, Details: err.Error()}
	}
	if !ok {
		return Result{Name: "gqlgen dependency", Status: StatusWarn, Details: "module not declared; add 'github.com/99designs/gqlgen' to go.mod"}
	}
	return Result{Name: "gqlgen dependency", Status: StatusOK}
}

func checkPgxDependency(context.Context) Result {
	ok, err := moduleListed("github.com/jackc/pgx/v5")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{Name: "pgx dependency", Status: StatusWarn, Details: "go.mod missing"}
		}
		return Result{Name: "pgx dependency", Status: StatusError, Details: err.Error()}
	}
	if !ok {
		return Result{Name: "pgx dependency", Status: StatusWarn, Details: "module not declared; add 'github.com/jackc/pgx/v5' to go.mod"}
	}
	return Result{Name: "pgx dependency", Status: StatusOK}
}

func checkGraphQLAssets(context.Context) Result {
	base := filepath.Join("internal", "graphql")
	if _, err := os.Stat(base); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{Name: "GraphQL assets", Status: StatusWarn, Details: "missing graphql; run 'erm graphql init'"}
		}
		return Result{Name: "GraphQL assets", Status: StatusError, Details: err.Error()}
	}
	missing := []string{}
	for _, name := range []string{"gqlgen.yml", "schema.graphqls"} {
		if _, err := os.Stat(filepath.Join(base, name)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missing = append(missing, name)
			} else {
				return Result{Name: "GraphQL assets", Status: StatusError, Details: err.Error()}
			}
		}
	}
	if len(missing) > 0 {
		return Result{Name: "GraphQL assets", Status: StatusWarn, Details: fmt.Sprintf("missing %s; run 'erm graphql init'", strings.Join(missing, ", "))}
	}
	return Result{Name: "GraphQL assets", Status: StatusOK}
}

func moduleListed(module string) (bool, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return false, err
	}
	return strings.Contains(string(data), module), nil
}
