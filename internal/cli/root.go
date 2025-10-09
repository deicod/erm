package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCmd constructs the root command with all subcommands attached.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "erm",
		Short: "erm - GraphQL + ORM code generator for Go (Relay, pgx, OIDC)",
		Long:  "erm generates a Relay-compliant GraphQL backend and an ent-like ORM over PostgreSQL (pgx v5) with OIDC middleware.",
	}
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newNewCmd())
	cmd.AddCommand(newGenCmd())
	cmd.AddCommand(newMigrateCmd())
	cmd.AddCommand(newGraphQLInitCmd())
	cmd.AddCommand(newDoctorCmd())
	return cmd
}

// Execute runs the CLI entrypoint.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
