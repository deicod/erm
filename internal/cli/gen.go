package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/deicod/erm/internal/generator"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var runGenerator = generator.Run

func newGenCmd() *cobra.Command {
	var (
		migrationName string
		dryRun        bool
		force         bool
		components    []string
		showDiff      bool
		watchMode     bool
	)
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate ORM, GraphQL, and migrations from schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, err := normalizeComponents(components)
			if err != nil {
				return wrapError(fmt.Sprintf("gen: %v", err), err, "Use --only with orm, graphql, or migrations", 2)
			}
			if showDiff && !dryRun {
				return wrapError("gen: --diff requires --dry-run", errors.New("diff requires dry-run"), "Re-run with --dry-run --diff to preview schema changes.", 2)
			}
			if watchMode && dryRun {
				return wrapError("gen: --watch cannot be combined with --dry-run", errors.New("invalid flag combination"), "Remove --dry-run to enable watch mode.", 2)
			}
			componentDesc := "all components"
			if len(targets) > 0 {
				componentDesc = humanizeList(targets)
			}
			opts := generator.GenerateOptions{
				MigrationName: migrationName,
				DryRun:        dryRun,
				Force:         force,
				Components:    targets,
			}
			logVerbose(cmd, "running generator with targets: %s", componentDesc)
			if watchMode {
				opts.StagingDir = filepath.Join(".", ".erm", "staging")
				return runWatch(cmd, opts, showDiff)
			}
			return executeGeneration(cmd, opts, showDiff)
		},
	}
	cmd.Flags().StringVar(&migrationName, "name", "", "Override the generated migration slug")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview migration SQL without writing files")
	cmd.Flags().BoolVar(&force, "force", false, "Rewrite generated files even if content is unchanged")
	cmd.Flags().StringSliceVar(&components, "only", nil, "Restrict generation to one or more components (orm, graphql, migrations)")
	cmd.Flags().BoolVar(&showDiff, "diff", false, "Include a schema diff summary (requires --dry-run)")
	cmd.Flags().BoolVar(&watchMode, "watch", false, "Watch schema files and regenerate impacted artifacts")
	return cmd
}

func executeGeneration(cmd *cobra.Command, opts generator.GenerateOptions, showDiff bool) error {
	result, err := runGenerator(".", opts)
	if err != nil {
		return wrapError(fmt.Sprintf("gen: generation failed: %v", err), err, "Resolve the schema or configuration issue above and re-run `erm gen`.", 1)
	}
	out := cmd.OutOrStdout()
	if opts.DryRun {
		printDryRunSummary(out, opts, result, showDiff)
		return nil
	}
	printComponentSummary(out, result.Components)
	printMigrationSummary(out, opts, result)
	fmt.Fprintln(out, "Generation complete.")
	return nil
}

func printDryRunSummary(out io.Writer, opts generator.GenerateOptions, result generator.RunResult, showDiff bool) {
	fmt.Fprintln(out, "generator: dry-run - no files were written")
	if includes(opts.Components, string(generator.ComponentORM)) || includes(opts.Components, string(generator.ComponentGraphQL)) || len(opts.Components) == 0 {
		fmt.Fprintln(out, "generator: application artifacts would be refreshed")
	}
	if includes(opts.Components, string(generator.ComponentMigrations)) || len(opts.Components) == 0 {
		if len(result.Migration.Operations) == 0 {
			fmt.Fprintln(out, "generator: no schema changes detected (dry-run)")
		} else {
			fmt.Fprintln(out, "generator: migration dry-run preview")
			fmt.Fprintln(out, result.Migration.SQL)
		}
		if showDiff {
			fmt.Fprintln(out, "generator: diff summary")
			if len(result.Migration.Operations) == 0 {
				fmt.Fprintln(out, "  (no schema changes)")
			} else {
				for _, op := range result.Migration.Operations {
					fmt.Fprintf(out, "  %s\n", formatOperation(op))
				}
			}
		}
	} else {
		fmt.Fprintln(out, "generator: migrations skipped (--only)")
	}
	fmt.Fprintln(out, "Generation preview complete.")
}

func printComponentSummary(out io.Writer, components []generator.ComponentResult) {
	for _, comp := range components {
		if comp.Name == generator.ComponentMigrations {
			continue
		}
		name := string(comp.Name)
		switch {
		case comp.Skipped && comp.Reason == "filtered (--only)":
			fmt.Fprintf(out, "generator: %s skipped (--only)\n", name)
		case comp.Skipped:
			fmt.Fprintf(out, "generator: %s %s\n", name, comp.Reason)
		case comp.Staged:
			fmt.Fprintf(out, "generator: %s staged -> %s\n", name, humanizeFiles(comp.Files))
		case len(comp.Files) > 0:
			fmt.Fprintf(out, "generator: %s updated %s\n", name, humanizeFiles(comp.Files))
		case comp.Reason != "":
			fmt.Fprintf(out, "generator: %s %s\n", name, comp.Reason)
		default:
			fmt.Fprintf(out, "generator: %s completed\n", name)
		}
	}
}

func printMigrationSummary(out io.Writer, opts generator.GenerateOptions, result generator.RunResult) {
	if !includes(opts.Components, string(generator.ComponentMigrations)) && len(opts.Components) != 0 {
		fmt.Fprintln(out, "generator: migrations skipped (--only)")
		return
	}
	if result.Migration.FilePath != "" {
		fmt.Fprintf(out, "generator: wrote migration %s\n", filepath.Base(result.Migration.FilePath))
		return
	}
	if len(result.Migration.Operations) == 0 {
		fmt.Fprintln(out, "generator: no schema changes detected")
		return
	}
	fmt.Fprintln(out, "generator: migration operations pending (dry-run or staged)")
}

func runWatch(cmd *cobra.Command, opts generator.GenerateOptions, showDiff bool) error {
	if err := os.MkdirAll(opts.StagingDir, 0o755); err != nil {
		return wrapError(fmt.Sprintf("gen: unable to prepare staging dir: %v", err), err, "Check permissions for .erm/staging and retry.", 1)
	}
	if err := executeGeneration(cmd, opts, showDiff); err != nil {
		return err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return wrapError(fmt.Sprintf("gen: watch failed: %v", err), err, "Install inotify/fsevents support and retry.", 1)
	}
	defer watcher.Close()

	schemaDir := filepath.Join(".", "schema")
	if err := watcher.Add(schemaDir); err != nil {
		return wrapError(fmt.Sprintf("gen: unable to watch schema directory: %v", err), err, "Ensure the schema directory exists before using --watch.", 1)
	}

	logVerbose(cmd, "watching %s for schema changes", schemaDir)

	debounce := time.NewTimer(0)
	if !debounce.Stop() {
		<-debounce.C
	}
	pending := false
	for {
		select {
		case <-cmd.Context().Done():
			return nil
		case event := <-watcher.Events:
			if !isSchemaEvent(event) {
				continue
			}
			pending = true
			if !debounce.Stop() {
				select {
				case <-debounce.C:
				default:
				}
			}
			debounce.Reset(200 * time.Millisecond)
		case err := <-watcher.Errors:
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "generator: watch error: %v\n", err)
			}
		case <-debounce.C:
			if !pending {
				continue
			}
			pending = false
			_ = os.RemoveAll(opts.StagingDir)
			if err := executeGeneration(cmd, opts, showDiff); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "generator: watch run failed: %v\n", err)
			}
		}
	}
}

func isSchemaEvent(event fsnotify.Event) bool {
	if event.Name == "" {
		return false
	}
	return strings.HasSuffix(event.Name, ".schema.go")
}

func includes(components []string, name string) bool {
	if len(components) == 0 {
		return true
	}
	return slices.Contains(components, name)
}

func humanizeFiles(files []string) string {
	if len(files) == 0 {
		return "(no file changes)"
	}
	if len(files) == 1 {
		return files[0]
	}
	return strings.Join(files, ", ")
}

func normalizeComponents(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	valid := map[string]struct{}{
		"orm":        {},
		"graphql":    {},
		"migrations": {},
	}
	set := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if _, ok := valid[normalized]; !ok {
			return nil, fmt.Errorf("unknown component %q", value)
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		set = append(set, normalized)
	}
	slices.Sort(set)
	return set, nil
}

func humanizeList(values []string) string {
	if len(values) == 0 {
		return ""
	}
	if len(values) == 1 {
		return values[0]
	}
	if len(values) == 2 {
		return values[0] + " and " + values[1]
	}
	return strings.Join(values[:len(values)-1], ", ") + ", and " + values[len(values)-1]
}

func formatOperation(op generator.Operation) string {
	prefix := "~"
	switch op.Kind {
	case generator.OpCreateExtension, generator.OpCreateTable, generator.OpAddColumn,
		generator.OpAddForeignKey, generator.OpAddIndex, generator.OpCreateHypertable:
		prefix = "+"
	case generator.OpDropExtension, generator.OpDropTable, generator.OpDropColumn,
		generator.OpDropForeignKey, generator.OpDropIndex, generator.OpDropHypertable:
		prefix = "-"
	case generator.OpAlterColumn:
		prefix = "~"
	}
	target := op.Target
	if target == "" {
		target = "(global)"
	}
	return fmt.Sprintf("%s %s %s", prefix, op.Kind, target)
}
