package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var verbose bool

// NewRootCmd constructs the root command with all subcommands attached.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "erm",
		Short: "erm - GraphQL + ORM code generator for Go (Relay, pgx, OIDC)",
		Long:  "erm generates a Relay-compliant GraphQL backend and an ent-like ORM over PostgreSQL (pgx v5) with OIDC middleware.",
	}
	cmd.SilenceUsage = true
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging output")
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
		exitCode := 1
		var cerr CommandError
		if errors.As(err, &cerr) {
			msg := strings.TrimSpace(cerr.Message)
			if msg == "" && cerr.Cause != nil {
				msg = cerr.Cause.Error()
			}
			if msg != "" {
				fmt.Fprintln(os.Stderr, msg)
			}
			if cerr.Cause != nil && msg != cerr.Cause.Error() && (verbose || msg == "") {
				fmt.Fprintf(os.Stderr, "details: %v\n", cerr.Cause)
			}
			if cerr.Suggestion != "" {
				fmt.Fprintln(os.Stderr, formatSuggestion(cerr.Suggestion))
			}
			exitCode = cerr.ExitStatus()
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(exitCode)
	}
}

func logVerbose(cmd *cobra.Command, format string, args ...any) {
	if !verbose {
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "[verbose] "+format+"\n", args...)
}
