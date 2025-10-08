package migrate

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Apply is a placeholder for versioned migration runner.
func Apply(ctx context.Context, conn pgx.Tx) error {
	// In v0, this will scan /migrations and apply.
	fmt.Println("migrate: (stub) apply migrations")
	return nil
}
