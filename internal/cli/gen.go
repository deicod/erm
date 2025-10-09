package cli

import (
	"fmt"

	"github.com/deicod/erm/internal/generator"
	"github.com/spf13/cobra"
)

func newGenCmd() *cobra.Command {
	var migrationName string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate ORM, GraphQL, and migrations from schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := generator.GeneratorOptions{
				MigrationName: migrationName,
				DryRun:        dryRun,
			}
			if err := generator.Run(".", opts); err != nil {
				return err
			}
			fmt.Println("Generation complete.")
			return nil
		},
	}
	cmd.Flags().StringVar(&migrationName, "migration-name", "", "Override the generated migration slug")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview migration SQL without writing files")
	return cmd
}
