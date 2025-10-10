package cli

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/deicod/erm/internal/generator"
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
	)
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate ORM, GraphQL, and migrations from schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, err := normalizeComponents(components)
			if err != nil {
				return wrapError(fmt.Sprintf("gen: %v", err), err, "Use --only with orm, graphql, or migrations", 2)
			}
			shouldGenerate := func(name string) bool {
				if len(targets) == 0 {
					return true
				}
				return slices.Contains(targets, name)
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
			result, err := runGenerator(".", opts)
			if err != nil {
				return wrapError(fmt.Sprintf("gen: generation failed: %v", err), err, "Resolve the schema or configuration issue above and re-run `erm gen`.", 1)
			}
			out := cmd.OutOrStdout()
			if dryRun {
				fmt.Fprintln(out, "generator: dry-run - no files were written")
				if shouldGenerate("orm") || shouldGenerate("graphql") {
					fmt.Fprintln(out, "generator: application artifacts would be refreshed")
				}
				if shouldGenerate("migrations") {
					if len(result.Operations) == 0 {
						fmt.Fprintln(out, "generator: no schema changes detected (dry-run)")
					} else {
						fmt.Fprintln(out, "generator: migration dry-run preview")
						fmt.Fprintln(out, result.SQL)
					}
					if showDiff {
						fmt.Fprintln(out, "generator: diff summary")
						if len(result.Operations) == 0 {
							fmt.Fprintln(out, "  (no schema changes)")
						} else {
							for _, op := range result.Operations {
								fmt.Fprintf(out, "  %s\n", formatOperation(op))
							}
						}
					}
				} else {
					fmt.Fprintln(out, "generator: migrations skipped (--only)")
				}
				fmt.Fprintln(out, "Generation preview complete.")
				return nil
			}
			var generated []string
			if shouldGenerate("orm") {
				generated = append(generated, "ORM")
			}
			if shouldGenerate("graphql") {
				generated = append(generated, "GraphQL")
			}
			if len(generated) > 0 {
				fmt.Fprintf(out, "generator: wrote %s artifacts\n", strings.ToLower(humanizeList(generated)))
			}
			if shouldGenerate("migrations") {
				if result.FilePath != "" {
					fmt.Fprintf(out, "generator: wrote migration %s\n", filepath.Base(result.FilePath))
				} else {
					fmt.Fprintln(out, "generator: no schema changes detected")
				}
			} else {
				fmt.Fprintln(out, "generator: migrations skipped (--only)")
			}
			fmt.Fprintln(out, "Generation complete.")
			return nil
		},
	}
	cmd.Flags().StringVar(&migrationName, "name", "", "Override the generated migration slug")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview migration SQL without writing files")
	cmd.Flags().BoolVar(&force, "force", false, "Rewrite generated files even if content is unchanged")
	cmd.Flags().StringSliceVar(&components, "only", nil, "Restrict generation to one or more components (orm, graphql, migrations)")
	cmd.Flags().BoolVar(&showDiff, "diff", false, "Include a schema diff summary (requires --dry-run)")
	return cmd
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
