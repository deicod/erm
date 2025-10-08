package cli

import (
	"fmt"

	"github.com/deicod/erm/internal/generator"
	"github.com/spf13/cobra"
)

func newGenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate ORM, GraphQL, and migrations from schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := generator.Run("."); err != nil {
				return err
			}
			fmt.Println("Generation complete.")
			return nil
		},
	}
	return cmd
}
