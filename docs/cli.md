# Command Line Interface (CLI)

The `erm` CLI is the orchestrator for every workflow—bootstrapping new services, generating code, applying migrations,
auto-configuring GraphQL, and verifying project health. This reference covers each command, the files it touches, and the
practical examples you can adapt to your team’s workflow.

---

## Installation

```bash
go install github.com/erm-project/erm/cmd/erm@latest
```

You can also run the CLI directly from source:

```bash
go run ./cmd/erm <command>
```

The CLI reads configuration from `erm.yaml` located at the project root. Commands emit actionable logs that describe what changed
and which files you may edit next.

---

## Global Options

All commands support a consistent set of flags:

| Flag | Description |
|------|-------------|
| `--config <path>` | Override the default `erm.yaml` location. |
| `--dry-run` | Print the actions that would be taken without writing files. |
| `--verbose` | Emit detailed logs, including resolved paths and diff summaries. |
| `--no-color` | Disable ANSI colors for CI pipelines. |

Environment variables:

- `ERM_ENV` – Switch between `development`, `staging`, or `production` contexts defined in `erm.yaml`.
- `ERM_DATABASE_URL` – Override the Postgres DSN for generation-time checks (useful in CI).
- `ERM_OIDC_ISSUER` – Override issuer discovery URL without editing configuration files.

---

## Command Reference

### `erm init`

Bootstraps a new service or re-initializes configuration in an existing repository.

```bash
erm init --module github.com/acme/payment --oidc-issuer http://localhost:8080/realms/demo
```

Actions performed:

- Creates `erm.yaml` populated with module path, database defaults, and OIDC issuer.
- Generates base folders: `cmd/server`, `internal/graphql`, `internal/orm/schema`, `docs/`, `migrations/`.
- Writes starter mixins (`time`, `soft delete`), GraphQL resolver stubs, and `.env.example` with connection strings.
- Adds Makefile targets for `gen`, `lint`, `test`, and `migrate`.

### `erm new <Entity>`

Generates a new schema skeleton under `internal/orm/schema/<entity>.go`.

```bash
erm new Invoice --table invoices --description "Customer invoices with line items"
```

The generated file includes TODO comments for fields, edges, indexes, annotations, hooks, interceptors, and privacy policies.
Use `.Mixins()` to embed reusable behaviors like `AuditMixin` or `SoftDeleteMixin`.

### `erm gen`

Runs the full generation pipeline.

```bash
erm gen --dry-run       # Inspect changes without touching the filesystem
erm gen --force         # Regenerate even if no changes were detected (useful after library upgrades)
```

Generation outputs:

- ORM packages under `internal/orm/<entity>` with fluent builders, query types, and `Edges` structs.
- GraphQL schema (`internal/graphql/schema.graphqls`), gqlgen config (`gqlgen.yml`), resolver implementations, and dataloader
  registration.
- Migration files under `migrations/<timestamp>_<name>.sql`, including extension management and comment DDL.
- Updated documentation comments that help AI tooling understand generated code.

`erm gen` is idempotent; you can run it repeatedly without creating inconsistent diffs.

### `erm graphql init`

Configures gqlgen and GraphQL server scaffolding.

```bash
erm graphql init --playground --listen :8080
```

Outputs include:

- `internal/graphql/server/server.go` – HTTP server with middleware chain (OIDC auth, tracing, logging).
- `internal/graphql/resolver/resolver.go` – Resolver root that delegates to generated ORM builders.
- `internal/graphql/node/registry.go` – Global Node lookup with base64 `<Type>:<uuidv7>` encoding helpers.
- `cmd/server/main.go` – Entry point that wires gqlgen handlers, dataloaders, and health endpoints.

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
# Edit internal/orm/schema/comment.go to add fields/edges
erm gen
# Edit internal/graphql/resolver/comment.resolvers.go for custom logic
make test
```

After `erm gen`, inspect the generated migration to confirm indexes and foreign keys match expectations. Update the
`docs/` portal with any domain-specific considerations.

### 2. Enabling pgvector for Recommendations

```bash
erm new Recommendation
# Add dsl.Vector("embedding", 1536) in Fields()
# Annotate schema with dsl.EnableExtension("vector")
erm gen
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
