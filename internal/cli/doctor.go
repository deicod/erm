package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/deicod/erm/internal/cli/doctor"
)

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Inspect the environment for common erm setup issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			results := doctor.Run()
			printer := doctor.NewPrinter(cmd.OutOrStdout())
			printer.PrintHeader("erm doctor")
			printer.PrintSystem(runtime.GOOS, runtime.GOARCH, runtime.Version())
			for _, res := range results {
				printer.PrintCheck(res)
			}
			printer.Summary(results)
			if doctor.HasFailures(results) {
				return fmt.Errorf("one or more checks failed")
			}
			return nil
		},
	}
	return cmd
}
