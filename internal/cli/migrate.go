package cli

import (
	"context"
	"errors"
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
				return fmt.Errorf("migrate: read project config: %w", err)
			}
			dsn := cfg.Database.URL
			if dsn == "" {
				return errors.New("migrate: database.url is not configured in erm.yaml")
			}
			ctx := cmd.Context()
			conn, err := openMigrationConn(ctx, dsn)
			if err != nil {
				return fmt.Errorf("migrate: connect database: %w", err)
			}
			defer conn.Close(ctx)
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "migrate: applying migrations")
			if err := applyMigrations(ctx, conn, os.DirFS(".")); err != nil {
				return fmt.Errorf("migrate: apply: %w", err)
			}
			fmt.Fprintln(out, "migrate: completed successfully")
			return nil
		},
	}
	return cmd
}
