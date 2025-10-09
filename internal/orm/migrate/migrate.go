package migrate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	defaultDirectory    = "migrations"
	defaultAdvisoryLock = int64(0x65726d)
)

// TxStarter abstracts pgx connections capable of starting a transaction.
type TxStarter interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

var _ TxStarter = (*pgx.Conn)(nil)

// Options configures how migrations are discovered and applied.
type Options struct {
	// Directory indicates the root within the supplied fs.FS that contains the
	// migration files. Defaults to "migrations".
	Directory string
	// BatchSize controls how many unapplied migrations are executed in a single
	// invocation. Zero means all pending migrations run in one transaction.
	BatchSize int
	// AdvisoryLockID overrides the pg_advisory_xact_lock key that guards
	// migrations. Defaults to a deterministic key derived from the project
	// prefix.
	AdvisoryLockID int64
}

// Option mutates Options.
type Option func(*Options)

// WithDirectory instructs Apply to look for migration files under dir.
func WithDirectory(dir string) Option {
	return func(o *Options) {
		if dir != "" {
			o.Directory = dir
		}
	}
}

// WithBatchSize limits the number of unapplied migrations executed per Apply
// invocation.
func WithBatchSize(size int) Option {
	return func(o *Options) {
		if size > 0 {
			o.BatchSize = size
		}
	}
}

// WithAdvisoryLock overrides the advisory lock identifier used to serialize
// migration runs.
func WithAdvisoryLock(id int64) Option {
	return func(o *Options) {
		if id != 0 {
			o.AdvisoryLockID = id
		}
	}
}

// FileMigration represents a single SQL migration discovered on disk.
type FileMigration struct {
	// Version is the parsed migration identifier recorded in erm_schema_migrations.
	Version string
	// Name is the base filename (e.g. 0001_init.sql).
	Name string
	// Path is the path relative to the root of the provided fs.FS.
	Path string
}

// ParseVersion extracts the version identifier from a migration filename. The
// identifier is derived from the portion of the name that precedes the first
// "__", "_", or "-" separator, falling back to the stem when none are present.
func ParseVersion(name string) (string, error) {
	if name == "" {
		return "", errors.New("migrate: empty filename")
	}
	base := strings.TrimSuffix(name, path.Ext(name))
	if base == "" {
		return "", fmt.Errorf("migrate: could not derive version from %q", name)
	}

	for _, sep := range []string{"__", "_", "-"} {
		if idx := strings.Index(base, sep); idx > 0 {
			return base[:idx], nil
		}
	}
	return base, nil
}

// Discover locates .sql migrations within dir and returns them in lexical order.
// A missing directory results in an empty slice.
func Discover(ctx context.Context, fsys fs.FS, dir string) ([]FileMigration, error) {
	if fsys == nil {
		return nil, errors.New("migrate: filesystem cannot be nil")
	}
	if dir == "" {
		dir = defaultDirectory
	}

	if _, err := fs.Stat(fsys, dir); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("migrate: inspect %s: %w", dir, err)
	}

	var files []FileMigration
	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".sql") {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		version, err := ParseVersion(d.Name())
		if err != nil {
			return fmt.Errorf("migrate: %s: %w", p, err)
		}
		files = append(files, FileMigration{Version: version, Name: d.Name(), Path: p})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].Version == files[j].Version {
			return files[i].Path < files[j].Path
		}
		return files[i].Version < files[j].Version
	})

	versions := make(map[string]string, len(files))
	for _, f := range files {
		if prev, ok := versions[f.Version]; ok {
			return nil, fmt.Errorf("migrate: duplicate version %q in %s and %s", f.Version, prev, f.Path)
		}
		versions[f.Version] = f.Path
	}

	return files, nil
}

// Apply discovers SQL migration files in fsys, executes unapplied migrations, and
// records them in erm_schema_migrations. All work occurs inside a single
// transaction protected by pg_advisory_xact_lock.
func Apply(ctx context.Context, conn TxStarter, fsys fs.FS, opts ...Option) error {
	if conn == nil {
		return errors.New("migrate: nil connection")
	}
	if fsys == nil {
		return errors.New("migrate: nil filesystem")
	}

	settings := Options{
		Directory:      defaultDirectory,
		AdvisoryLockID: defaultAdvisoryLock,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&settings)
		}
	}

	migrations, err := Discover(ctx, fsys, settings.Directory)
	if err != nil {
		return err
	}

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("migrate: begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", settings.AdvisoryLockID); err != nil {
		return fmt.Errorf("migrate: acquire advisory lock: %w", err)
	}

	if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS erm_schema_migrations (
        version    text PRIMARY KEY,
        applied_at timestamptz NOT NULL DEFAULT now()
    )`); err != nil {
		return fmt.Errorf("migrate: ensure tracking table: %w", err)
	}

	applied := make(map[string]struct{})
	rows, err := tx.Query(ctx, "SELECT version FROM erm_schema_migrations")
	if err != nil {
		return fmt.Errorf("migrate: list applied versions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var version string
		if scanErr := rows.Scan(&version); scanErr != nil {
			return fmt.Errorf("migrate: read applied versions: %w", scanErr)
		}
		applied[version] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("migrate: read applied versions: %w", err)
	}

	var toApply []FileMigration
	for _, mig := range migrations {
		if _, ok := applied[mig.Version]; ok {
			continue
		}
		toApply = append(toApply, mig)
		if settings.BatchSize > 0 && len(toApply) == settings.BatchSize {
			break
		}
	}

	for _, mig := range toApply {
		raw, readErr := fs.ReadFile(fsys, mig.Path)
		if readErr != nil {
			return fmt.Errorf("migrate: %s: %w", mig.Path, readErr)
		}
		if _, execErr := tx.Exec(ctx, string(raw)); execErr != nil {
			return wrapExecError(mig.Path, string(raw), execErr)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO erm_schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING", mig.Version); err != nil {
			return fmt.Errorf("migrate: record %s: %w", mig.Version, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("migrate: commit transaction: %w", err)
	}
	committed = true
	return nil
}

func wrapExecError(path, sql string, execErr error) error {
	var pgErr *pgconn.PgError
	if errors.As(execErr, &pgErr) {
		if pgErr.Line > 0 {
			return fmt.Errorf("%s:%d: %w", path, pgErr.Line, execErr)
		}
		if pgErr.Position > 0 {
			line, column := positionToLineColumn(sql, int(pgErr.Position))
			return fmt.Errorf("%s:%d:%d: %w", path, line, column, execErr)
		}
	}
	return fmt.Errorf("%s: %w", path, execErr)
}

func positionToLineColumn(sql string, position int) (int, int) {
	if position <= 0 {
		return 1, 1
	}
	// position is 1-indexed byte offset. Convert to rune iteration to compute
	// line and column.
	line, column := 1, 1
	counted := 0
	for _, r := range sql {
		counted++
		if counted == position {
			break
		}
		if r == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return line, column
}
