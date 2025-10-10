# Getting Started with the App Skeleton

Running `erm init` bootstraps a workspace with a runnable HTTP skeleton, schema
stubs, and guardrails to keep your development workflow consistent. This guide
explains what is generated, how to iterate locally, and the expectations
captured in the AGENTS instructions.

## Generated layout

`erm init` is idempotent—re-running the command only adds missing files. A fresh
workspace will contain:

- `erm.yaml` – project configuration with placeholders for the Go module path,
  database URL, and feature flags.
- `README.md` – onboarding guide that walks through the recommended workflow.
- `AGENTS.md` – development expectations: TDD, formatting, and validation
  commands to run before committing.
- `cmd/api/main.go` – minimal HTTP server with a health check and graceful
  shutdown hooks. This is the entrypoint for wiring resolvers once GraphQL is
  generated.
- `schema/` – empty directory with an `AGENTS.md` reminder to regenerate code
  and run tests whenever entities change.
- `internal/graphql/README.md` – instructions for bootstrapping gqlgen with
  `erm graphql init` and maintaining resolvers.
- `migrations/` – empty directory ready for generated SQL migrations.

## Configuring `erm.yaml`

Update `module` to match your Go module path. Point `database.url` at your
Postgres instance, and adjust extension toggles (PostGIS, pgvector,
TimescaleDB) to match your stack. These values drive code generation and CLI
commands like `erm gen` and `erm migrate`.

## Exploring `cmd/api/main.go`

The generated server exposes:

- `GET /healthz` returning `200 OK` with body `ok` for readiness checks.
- Graceful shutdown handling via `signal.NotifyContext`.
- A TODO reminding you to mount the GraphQL handler after running
  `erm graphql init` and `erm gen`.

You can run the service immediately:

```bash
ERM_DATABASE_URL=postgres://user:pass@localhost:5432/app?sslmode=disable \
    go run ./cmd/api
```

Update the port or middleware as your requirements evolve.

## Recommended workflow

1. Initialise the Go module (`go mod init <module>`) and keep dependencies
   tidy with `go mod tidy`.
2. Create entities using `erm new <Entity>` and adjust the generated schema.
3. Regenerate artifacts with `erm gen` whenever the schema changes.
4. Implement resolvers under `internal/graphql` and wire them into the HTTP
   server.
5. Apply migrations locally with `erm migrate` as you iterate.

## Quality gates

The generated `AGENTS.md` files outline required checks:

- Format code with `gofmt -w`.
- Run the full test suite: `go test ./...`.
- Execute race detection: `go test -race ./...`.
- Lint with `go vet ./...`.

These commands keep the skeleton healthy and mirror the automation expected in
CI pipelines.

## Next steps

Once you are comfortable with the app skeleton:

- Follow [Getting Started with the Schema Skeleton](./getting-started-schema.md)
  to model your domain.
- Review the [CLI reference](./cli.md) for additional commands like
  `erm graphql init`, `erm migrate`, and `erm doctor`.
- Explore [Best Practices](./best-practices.md) to align with team conventions.

With the skeleton in place, you can confidently iterate on features while
retaining a clear feedback loop across schema, database, and GraphQL layers.
