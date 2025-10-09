package cli

import (
	"fmt"
	"path/filepath"

	"github.com/deicod/erm/internal/generator"
	"github.com/spf13/cobra"
)

var runGenerator = generator.Run

func newGenCmd() *cobra.Command {
	var (
		migrationName string
		dryRun        bool
		force         bool
	)
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate ORM, GraphQL, and migrations from schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := generator.GenerateOptions{
				MigrationName: migrationName,
				DryRun:        dryRun,
				Force:         force,
			}
			result, err := runGenerator(".", opts)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if dryRun {
				fmt.Fprintln(out, "generator: dry-run - no files were written")
				if len(result.Operations) == 0 {
					fmt.Fprintln(out, "generator: no schema changes detected (dry-run)")
				} else {
					fmt.Fprintln(out, "generator: migration dry-run preview")
					fmt.Fprintln(out, result.SQL)
				}
				fmt.Fprintln(out, "Generation preview complete.")
				return nil
			}
			if result.FilePath != "" {
				fmt.Fprintf(out, "generator: wrote migration %s\n", filepath.Base(result.FilePath))
			} else {
				fmt.Fprintln(out, "generator: no schema changes detected")
			}
			fmt.Fprintln(out, "generator: wrote ORM and GraphQL artifacts")
			fmt.Fprintln(out, "Generation complete.")
			return nil
		},
	}
	cmd.Flags().StringVar(&migrationName, "name", "", "Override the generated migration slug")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview migration SQL without writing files")
	cmd.Flags().BoolVar(&force, "force", false, "Rewrite generated files even if content is unchanged")
	return cmd
}
