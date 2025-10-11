package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize erm in the current workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			files := []struct{ path, content string }{
				{"erm.yaml", defaultConfig},
				{"README.md", workspaceReadme},
				{"AGENTS.md", workspaceAgents},
				{"cmd/api/main.go", apiMain},
				{"schema/.gitkeep", ""},
				{"schema/AGENTS.md", schemaAgents},
				{"graphql/README.md", gqlReadme},
				{"migrations/.gitkeep", ""},
			}

			for _, f := range files {
				if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
					return wrapError(
						fmt.Sprintf("init: create directory %s", filepath.Dir(f.path)),
						err,
						"Check directory permissions or run the command from a writable workspace.",
						1,
					)
				}
				if _, err := os.Stat(f.path); err == nil {
					continue // idempotent
				}
				if err := os.WriteFile(f.path, []byte(f.content), 0o644); err != nil {
					return wrapError(
						fmt.Sprintf("init: write file %s", f.path),
						err,
						"Ensure the path is writable and not protected by source control attributes.",
						1,
					)
				}
			}
			fmt.Fprintln(out, "Initialized erm workspace.")
			return nil
		},
	}
	return cmd
}

const defaultConfig = `# erm configuration
# 1. Set module to your Go module path (for example "github.com/acme/app").
module: ""
database:
  # 2. Update the connection string to point at your development database.
  url: "postgres://user:pass@localhost:5432/app?sslmode=disable"
oidc:
  # 3. Configure the issuer and audience to match your identity provider.
  issuer: "https://auth.example.com/realms/app"
  audience: "web-spa"
graphql:
  # 4. The HTTP path your API will be served on.
  path: "/graphql"
extensions:
  postgis: true
  pgvector: true
  timescaledb: false
`

const workspaceReadme = `# Welcome to your erm project

This workspace was bootstrapped with 'erm init'. The goal of the generated
skeleton is to give you a runnable HTTP service, a place to define your schema,
and a repeatable workflow for regenerating code as your domain evolves.

## Next steps

1. Initialize your Go module and align 'erm.yaml':
       go mod init <module>
       go mod tidy
   Update 'module' in 'erm.yaml' to match the value passed to 'go mod init'.
2. Sketch your first entity with 'erm new <Entity>'.
3. Run 'erm gen' to materialize ORM, GraphQL, and migration artifacts.
4. Start the HTTP server with 'go run ./cmd/api' and iterate.

## Project layout

- 'cmd/api' — entrypoint for the HTTP server and integration glue.
- 'schema' — your application schema. Run 'erm gen' whenever it changes.
- 'graphql' — gqlgen configuration and generated resolvers.
- 'migrations' — versioned SQL migrations managed by 'erm gen'.

## Recommended workflow

1. Practice TDD: write or update tests alongside feature work.
2. Keep the code formatted with 'gofmt -w' on edited files.
3. Validate changes locally:
   - 'go test ./...'
   - 'go test -race ./...'
   - 'go vet ./...'
4. Regenerate artifacts with 'erm gen' and review the diff before committing.
5. Use 'erm migrate' to apply database changes during development.

Happy hacking!
`

const workspaceAgents = `# Development Workflow

When contributing to this workspace:

1. Follow TDD — add or update tests before changing behavior.
2. Keep formatting clean: run 'gofmt -w' on touched Go files.
3. Before pushing a branch, validate with:
   - 'go test ./...'
   - 'go test -race ./...'
   - 'go vet ./...'
4. Regenerate code after schema changes with 'erm gen' and commit the results.
5. Prefer small, reviewable commits and document notable workflows in the repo.
`

const apiMain = `package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "time"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    })

    // TODO: mount your GraphQL handler once 'erm graphql init' has generated the scaffolding.

    srv := &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }

    log.Printf("serving HTTP on %s", srv.Addr)

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
    defer stop()

    go func() {
        <-ctx.Done()
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := srv.Shutdown(shutdownCtx); err != nil {
            log.Printf("graceful shutdown failed: %v", err)
        }
    }()

    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("server error: %v", err)
    }
}
`

const schemaAgents = `# Schema Development Workflow

When touching files in this directory:

1. Practice TDD — write or update tests alongside schema changes.
2. Run 'gofmt -w' on edited schema files before committing.
3. Validate the project with:
   - 'go test ./...'
   - 'go test -race ./...'
   - 'go vet ./...'
4. Regenerate code with 'erm gen' when the schema shape changes and review the diff before committing.
`

const gqlReadme = `# GraphQL workspace

This directory will hold gqlgen configuration, generated schema stubs, and
resolver implementations.

Run 'erm graphql init' to scaffold gqlgen, then wire the generated handler into
'cmd/api/main.go'.

The recommended workflow is:

1. Define or update schema entities under 'schema/'.
2. Execute 'erm gen' to refresh ORM, GraphQL, and migration assets.
3. Implement resolver logic and keep tests under 'graphql' up to date.
`
