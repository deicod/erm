package cli

import (
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
				return CommandError{
					Message:    "doctor: one or more checks failed",
					Suggestion: "Resolve the failing checks above and rerun `erm doctor`.",
					ExitCode:   3,
				}
			}
			return nil
		},
	}
	return cmd
}
