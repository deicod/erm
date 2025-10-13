# Command Line Interface (CLI)

The `erm` CLI is the orchestrator for every workflow—bootstrapping new services, generating code, applying migrations,
auto-configuring GraphQL, and verifying project health. This reference covers each command, the files it touches, and the
practical examples you can adapt to your team’s workflow.

---

## Installation

Download a pre-built binary from the [GitHub Releases page](https://github.com/deicod/erm/releases) and place it on your `PATH`
for the quickest setup on macOS, Linux, or Windows. Each release includes checksums so you can verify the download before
executing it.

To build from source instead (recommended when contributing changes), run:

```bash
go install github.com/deicod/erm/cmd/erm@latest
```

You can also run the CLI directly from source without installing it globally:

```bash
go run ./cmd/erm <command>
```

The CLI reads configuration from `erm.yaml` located at the project root. Commands emit actionable logs that describe what changed
and which files you may edit next.

---

## Global Options

The root command exposes a single persistent flag that you can use with any sub-command:

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Print diagnostic logs and include wrapped error details alongside remediation hints. |

---

## Command Reference

### `erm init`

Bootstraps a new service or re-initializes configuration in an existing repository.

```bash
erm init --module github.com/acme/payment --oidc-issuer http://localhost:8080/realms/demo
```

Actions performed:

- Creates `erm.yaml` populated with module, database defaults, and OIDC issuer.
- Seeds workspace docs, a starter `cmd/api/main.go`, and empty `schema/` + `migrations/` folders.
- Materialises GraphQL and observability scaffolds the generator relies on:
  - `graphql/dataloaders/loader.go`
  - `graphql/directives/auth.go`
  - `graphql/relay/id.go`
  - `graphql/resolvers/resolver.go`
  - `graphql/resolvers/entities_hooks.go`
  - `graphql/server/schema.go` & `graphql/server/server.go`
  - `graphql/subscriptions/bus.go`
  - `observability/metrics/metrics.go`
  - `oidc/claims.go`
  - `graphql/scalars.go` & `graphql/types.go`

These files are only written when absent, so rerunning `erm init` preserves edits. After running `erm gen`, extend the scaffolds by editing:

- `graphql/server/schema.go` to register middleware, transports, or directives.
- `graphql/directives/auth.go` for RBAC helpers that consume your `oidc` claims.
- `graphql/resolvers/resolver.go` to tune pagination defaults, subscription topics, or expose additional health probes.
- `graphql/resolvers/entities_hooks.go` for pre/post mutation hooks and resolver-level instrumentation.
- `graphql/types.go` to add scalar wrappers or adapters required by gqlgen.
- `observability/metrics` to plug in real collectors.
- `oidc/claims.go` to adapt identity providers.

### `erm new <Entity>`

Generates a new schema skeleton under `schema/<entity>.schema.go` with TODOs for fields, edges, indexes, annotations, hooks, interceptors, and privacy policies.

### `erm gen`

Runs the full generation pipeline with explicit controls for migrations and output writes.

```bash
erm gen --dry-run                  # Print the migration SQL without touching the filesystem
erm gen --only orm,graphql         # Regenerate application code without creating migrations
erm gen --dry-run --diff           # Show a summarized schema diff along with the SQL preview
erm gen --name add_users_email     # Override the generated migration slug
erm gen --force                    # Rewrite generated files even if contents are unchanged
```

Key flags:

- `--only` limits generation to `orm`, `graphql`, or `migrations`. Omit the flag to run the full pipeline.
- `--dry-run` previews the work without writing to disk. Combine it with `--diff` to emit a concise change summary.
- `--diff` formats each migration operation with `+`, `-`, or `~` prefixes so you can skim structural changes quickly.
- `--name` customizes the slug appended to the timestamp in the generated migration filename.
- `--force` bypasses on-disk equality checks, rewriting artifacts when you need to regenerate after upgrading dependencies.
- Ensure the project module path is set in `erm.yaml` (or inferred from `go.mod`). GraphQL resolvers and dataloaders import
  generated packages under `graphql/*` and `orm/*` using that module path; generation fails if it cannot be determined.
- The generator hydrates missing runtime scaffolds (GraphQL server, dataloaders, directives, observability, OIDC helpers)
  before invoking gqlgen so subsequent `go mod tidy` or `go test ./graphql/...` runs succeed without copying templates.

Generation outputs when not running in dry-run mode:

- ORM packages under `orm/<entity>` with fluent builders, query types, and `Edges` structs.
- GraphQL schema (`graphql/schema.graphqls`), gqlgen config (`gqlgen.yml`), resolver implementations, and dataloader
  registration.
- Migration files under `migrations/<timestamp>_<name>.sql`, including extension management and comment DDL.
- Updated documentation comments that help AI tooling understand generated code.

`erm gen` is idempotent; you can run it repeatedly without creating inconsistent diffs, and combine `--dry-run` with CI to
guard against accidental schema drift.

### `erm migrate`

Manages SQL migrations using the Postgres DSN defined in `erm.yaml` or via runtime overrides.

Execution modes:

- `--mode plan` performs a non-destructive inspection of the migrations directory, verifying that every applied version still exists on disk and reporting pending migrations in execution order.
- `--mode apply` (default) runs unapplied migrations inside a single transaction protected by an advisory lock. The CLI prints the number of migrations it is about to execute.
- `--mode rollback` replays the most recent `*_down.sql` script and removes the corresponding row from `erm_schema_migrations`. The command aborts if it detects schema drift or missing rollback files.

Environment targeting:

- `--env <profile>` selects a profile from `database.environments` in `erm.yaml` (defaults to `dev`).
- `ERM_ENV` provides the same selection via environment variable.
- `ERM_DATABASE_URL` overrides all profile values so CI/CD systems can supply ephemeral credentials.

```bash
erm migrate --mode plan              # Preflight migrations using the dev profile
erm migrate --mode apply --env prod  # Execute pending migrations against production
ERM_ENV=staging erm migrate          # Shortcut: apply using the staging profile
```

The command streams progress to stdout and wraps errors from the underlying executor, making it safe to wire into CI or local scripts. It reuses the schema snapshot generated by `erm gen` so migrations remain incremental and deterministic.

### `erm graphql init`

Configures gqlgen and refreshes runtime scaffolds without overwriting local customizations.

```bash
erm graphql init --playground --listen :8080
```

Outputs include:

- `graphql/gqlgen.yml` and `graphql/schema.graphqls` to seed gqlgen.
- The runtime packages listed under `erm init`, written only when missing.
- `graphql/scalars.go`, a helper file that `erm gen` will extend when new custom scalars are detected.

Anything ending in `_gen.go` remains generator-owned; the runtime scaffolds above are safe to edit and will survive future runs.

### `erm doctor` *(experimental)*

Performs project health checks. This command is under active development but the scaffold ships today so you can wire it into CI.
Current diagnostics include:

- Detecting schema files that have changed without a corresponding `erm gen` run.
- Validating that migration timestamps are monotonically increasing.
- Checking that `erm.yaml` matches the module path in `go.mod`.
- Verifying that GraphQL schema definitions match resolver implementations.

Run it locally before committing, or add it to GitHub Actions once the `doctor` command graduates from preview.

---

## Workflow Examples

### 1. Creating a Feature Slice

```bash
erm new Comment
# Edit orm/schema/comment.go to add fields/edges
erm gen
erm migrate                   # Apply the new migration to your local database
# Edit graphql/resolver/comment.resolvers.go for custom logic
make test
```

After `erm gen`, inspect the generated migration to confirm indexes and foreign keys match expectations. Update the
`docs/` portal with any domain-specific considerations.

### 2. Enabling pgvector for Recommendations

```bash
erm new Recommendation
# Add dsl.Vector("embedding", 1536) in Fields()
# Annotate schema with dsl.EnableExtension("vector")
erm gen --name enable_vector_extension
erm migrate
```

The generator adds `CREATE EXTENSION IF NOT EXISTS vector;` to the migration and emits helper methods for similarity search.
Consult [extensions.md](./extensions.md#pgvector) for usage patterns and query examples.

### 3. Upgrading Schema with Privacy Constraints

```bash
# Add privacy rules to the schema
func (User) Privacy() dsl.Privacy {
    return dsl.Privacy{
        Read:   "viewer.id == node.id || viewer.is_admin",
        Update: "viewer.id == node.id",
        Delete: "viewer.is_admin",
    }
}
erm gen
```

Generated resolvers automatically enforce these policies before hitting the database. The GraphQL guide includes examples of how
the rules translate into runtime checks.

---

## Integrating with Tooling

- **Go Generate:** Add `//go:generate erm gen` directives to schema packages so `go generate ./...` keeps code fresh.
- **CI Pipelines:** Run `erm gen --dry-run` to detect drift and `erm doctor` to enforce health checks before merging.
- **Migrations:** Wire `erm gen` into your migration workflow; apply SQL files using `migrate`, `goose`, or the tool of your
  choice. The generated SQL includes comments describing the originating schema field for traceability.

---

## Troubleshooting CLI Issues

| Symptom | Resolution |
|---------|------------|
| CLI reports “no module found” | Ensure `go.mod` exists and `erm.yaml.module` matches `module` declaration. |
| `erm gen` fails with Postgres error | The CLI validates migrations using the configured DSN. Check connectivity and ensure the
  target database allows the extension/migration being applied. |
| Files regenerate on every run | Confirm your editor preserves Go formatting in schema files and that you are not introducing
  nondeterministic timestamps in annotations. |
| `erm graphql init` overwrites manual changes | Custom business logic should live in files marked `// Code generated` – keep custom
  code in `_extension.go` files or resolver stubs outside generated blocks. |

---

## Next Steps

- Continue to the [Schema Definition Guide](./schema-definition.md) to master the DSL.
- Review [Performance & Observability](./performance-observability.md) to tune the runtime immediately after scaffolding.
- Run `erm doctor` (preview) regularly to stay ahead of configuration drift.
