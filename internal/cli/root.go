package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "erm",
	Short: "erm - GraphQL + ORM code generator for Go (Relay, pgx, OIDC)",
	Long:  "erm generates a Relay-compliant GraphQL backend and an ent-like ORM over PostgreSQL (pgx v5) with OIDC middleware.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(genCmd)
	rootCmd.AddCommand(gqlInitCmd)
}
