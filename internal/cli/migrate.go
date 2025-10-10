package cli

import (
	"context"
	"fmt"
	"os"

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
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Apply SQL migrations to the configured database",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadProjectConfig(".")
			if err != nil {
				return wrapError("migrate: read project config", err, "Ensure erm.yaml exists in the project root.", 1)
			}
			dsn := cfg.Database.URL
			if dsn == "" {
				return CommandError{
					Message:    "migrate: database.url is not configured in erm.yaml",
					Suggestion: "Set database.url in erm.yaml or export ERM_DATABASE_URL before running the command.",
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
			fmt.Fprintln(out, "migrate: applying migrations")
			if err := applyMigrations(ctx, conn, os.DirFS(".")); err != nil {
				return wrapError("migrate: apply migrations", err, "Review the SQL error, fix the migration, and re-run `erm migrate`.", 1)
			}
			fmt.Fprintln(out, "migrate: completed successfully")
			return nil
		},
	}
	return cmd
}
