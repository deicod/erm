package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize erm in the current workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		files := []struct{ path, content string }{
			{"erm.yaml", defaultConfig},
			{"schema/.gitkeep", ""},
			{"internal/graphql/README.md", gqlReadme},
			{"migrations/.gitkeep", ""},
		}
		for _, f := range files {
			if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil { return err }
			if _, err := os.Stat(f.path); err == nil { continue } // idempotent
			if err := os.WriteFile(f.path, []byte(f.content), 0o644); err != nil { return err }
		}
		fmt.Println("Initialized erm workspace.")
		return nil
	},
}

var defaultConfig = `# erm configuration
module: ""
database:
  url: "postgres://user:pass@localhost:5432/app?sslmode=disable"
oidc:
  issuer: "https://auth.example.com/realms/app"
  audience: "web-spa"
graphql:
  path: "/graphql"
extensions:
  postgis: true
  pgvector: true
  timescaledb: false
`

var gqlReadme = `This directory will hold generated GraphQL schema and resolvers.
Run: erm graphql init`
