package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/deicod/erm/internal/orm/migrate"
	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
)

type migrationConn interface {
	migrate.TxStarter
	Close(ctx context.Context) error
}

var (
	openMigrationConn = func(ctx context.Context, url string) (migrationConn, error) {
		return pgx.Connect(ctx, url)
	}
	applyMigrations = migrate.Apply
	planMigrations  = migrate.Plan
	rollbackMig     = migrate.Rollback
)

func newMigrateCmd() *cobra.Command {
	var (
		mode    string
		envName string
	)
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Manage SQL migrations for the configured database",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadProjectConfig(".")
			if err != nil {
				return wrapError("migrate: read project config", err, "Ensure erm.yaml exists in the project root.", 1)
			}
			profile := envName
			if profile == "" {
				profile = os.Getenv("ERM_ENV")
			}
			if profile == "" {
				profile = "dev"
			}
			dsn := cfg.Database.URL
			if envCfg, ok := cfg.Database.Environments[profile]; ok && envCfg.URL != "" {
				dsn = envCfg.URL
			}
			if override := os.Getenv("ERM_DATABASE_URL"); override != "" {
				dsn = override
			}
			if dsn == "" {
				return CommandError{
					Message:    "migrate: database.url is not configured in erm.yaml",
					Suggestion: "Set database.url in erm.yaml, configure database.environments, or export ERM_DATABASE_URL before running the command.",
					ExitCode:   2,
				}
			}
			execMode := strings.ToLower(mode)
			if execMode == "" {
				execMode = "apply"
			}
			switch execMode {
			case "plan", "apply", "rollback":
			default:
				return CommandError{
					Message:    fmt.Sprintf("migrate: unsupported mode %q", execMode),
					Suggestion: "Use one of plan, apply, or rollback.",
					ExitCode:   2,
				}
			}
			ctx := cmd.Context()
			conn, err := openMigrationConn(ctx, dsn)
			if err != nil {
				return wrapError(fmt.Sprintf("migrate: connect database %s", dsn), err, "Verify the database is reachable and credentials are correct.", 1)
			}
			defer conn.Close(ctx)
			out := cmd.OutOrStdout()
			fsys := os.DirFS(".")
			plan, err := planMigrations(ctx, conn, fsys)
			if err != nil {
				var driftErr migrate.SchemaDriftError
				if errors.As(err, &driftErr) {
					return CommandError{
						Message:    fmt.Sprintf("migrate: schema drift detected for %s", strings.Join(driftErr.Missing, ", ")),
						Suggestion: "Review applied migrations, restore the missing SQL files, or reconcile the database state before continuing.",
						ExitCode:   1,
					}
				}
				return wrapError("migrate: plan migrations", err, "Resolve the planning error before retrying.", 1)
			}

			switch execMode {
			case "plan":
				fmt.Fprintf(out, "migrate: plan for %s\n", profile)
				if len(plan.Pending) == 0 {
					fmt.Fprintln(out, "migrate: database is up-to-date")
					return nil
				}
				versions := make([]string, 0, len(plan.Pending))
				for _, mig := range plan.Pending {
					versions = append(versions, fmt.Sprintf("%s (%s)", mig.Version, mig.Name))
				}
				sort.Strings(versions)
				for _, v := range versions {
					fmt.Fprintf(out, "  pending: %s\n", v)
				}
				return nil
			case "apply":
				if len(plan.Pending) == 0 {
					fmt.Fprintln(out, "migrate: database is up-to-date")
					return nil
				}
				fmt.Fprintf(out, "migrate: applying %d migration(s)\n", len(plan.Pending))
				if err := applyMigrations(ctx, conn, fsys); err != nil {
					return wrapError("migrate: apply migrations", err, "Review the SQL error, fix the migration, and re-run `erm migrate --mode apply`.", 1)
				}
				fmt.Fprintln(out, "migrate: completed successfully")
				return nil
			case "rollback":
				reverted, err := rollbackMig(ctx, conn, fsys)
				if err != nil {
					if errors.Is(err, migrate.ErrNoAppliedMigrations) {
						return CommandError{
							Message:    "migrate: no applied migrations to rollback",
							Suggestion: "Ensure at least one migration has been applied before running rollback.",
							ExitCode:   1,
						}
					}
					return wrapError("migrate: rollback", err, "Ensure a corresponding *_down.sql exists and the database is reachable.", 1)
				}
				fmt.Fprintf(out, "migrate: rolled back %s (%s)\n", reverted.Version, reverted.Name)
				return nil
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "apply", "Select plan, apply, or rollback execution mode")
	cmd.Flags().StringVar(&envName, "env", "", "Target environment profile (dev, staging, prod)")
	return cmd
}
