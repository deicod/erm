package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var (
	goListPackages = realGoList
	goTestCommand  = realGoTest
)

func newTestCmd() *cobra.Command {
	var (
		race      bool
		batchSize int
	)
	cmd := &cobra.Command{
		Use:   "test [packages]",
		Short: "Run go test with ergonomics for the erm repository",
		Long: "test wraps `go test` with the defaults we rely on in CI. " +
			"Enable --race to batch packages so the detector finishes in a reasonable time.",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if batchSize <= 0 {
				return wrapError("test: --batch-size must be greater than zero", errors.New("invalid batch size"),
					"Pick a positive batch size when running with --race.", 2)
			}
			patterns := args
			if len(patterns) == 0 {
				patterns = []string{"./..."}
			}
			ctx := cmd.Context()
			if race {
				return runRaceBatches(ctx, cmd, patterns, batchSize)
			}
			goArgs := append([]string{"test"}, patterns...)
			fmt.Fprintf(cmd.OutOrStdout(), "==> go %s\n", strings.Join(goArgs, " "))
			if err := goTestCommand(ctx, goArgs, cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
				return wrapError(fmt.Sprintf("test: go test failed: %v", err), err, "Review the go test output above for details.", 1)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&race, "race", false, "Enable the Go race detector (packages run in batches)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 8, "Number of packages per go test invocation when --race is enabled")
	return cmd
}

func runRaceBatches(ctx context.Context, cmd *cobra.Command, patterns []string, batchSize int) error {
	packages, err := goListPackages(ctx, patterns)
	if err != nil {
		return wrapError(fmt.Sprintf("test: go list failed: %v", err), err, "Resolve the package listing error and retry.", 1)
	}
	if len(packages) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "test: no packages matched the provided patterns")
		return nil
	}
	for start := 0; start < len(packages); start += batchSize {
		end := start + batchSize
		if end > len(packages) {
			end = len(packages)
		}
		chunk := packages[start:end]
		goArgs := append([]string{"test", "-race"}, chunk...)
		fmt.Fprintf(cmd.OutOrStdout(), "==> go %s\n", strings.Join(goArgs, " "))
		if err := goTestCommand(ctx, goArgs, cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
			return wrapError(fmt.Sprintf("test: go test -race failed: %v", err), err, "Review the failing package output above and address the race.", 1)
		}
	}
	return nil
}

func realGoList(ctx context.Context, patterns []string) ([]string, error) {
	args := append([]string{"list"}, patterns...)
	cmd := exec.CommandContext(ctx, "go", args...)
	output, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(exitErr.Stderr))
		}
		if stderr != "" {
			return nil, fmt.Errorf("%w: %s", err, stderr)
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}
	return lines, nil
}

func realGoTest(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
