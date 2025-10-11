# CI/CD Guidance

This document explains how to integrate the `erm` tooling into automated pipelines. It focuses on schema validation, safe migration rollouts, and smoke testing using the shared testing utilities.

## Pipeline Layout

The default GitHub Actions workflow (`.github/workflows/ci.yml`) now runs two jobs:

1. **build** – executes formatting checks, `go build`, unit tests, and verifies that generated artifacts are checked in.
2. **migrations** – provisions a disposable Postgres instance, runs `erm migrate` in `plan` and `apply` modes, and executes smoke tests built on `testing`.

Re-use the workflow by copying the job definitions into your own repositories or by calling it from reusable workflows.

## Schema Validation

`erm migrate --mode plan` performs a non-destructive inspection of the migrations directory. It validates that:

- every applied migration recorded in `erm_schema_migrations` still exists on disk;
- pending migrations are reported in execution order; and
- optional batch limits (`--batch-size`) are respected.

Always run the plan step in CI and block merges if schema drift is detected.

## Applying Migrations

Use `--mode apply` to execute pending migrations inside a single transaction guarded by an advisory lock. The CLI prints the number of migrations that will run after the plan step completes successfully. Apply mode respects the selected environment profile and will honour `ERM_DATABASE_URL` secrets when present.

## Rollbacks

The `rollback` mode replays the matching `*_down.sql` script for the most recently applied migration. It will refuse to run if the database contains migrations whose SQL files are missing. Keep rollback scripts adjacent to the forward migrations.

In CI you can validate rollback scripts with:

```bash
ERM_DATABASE_URL=postgres://erm:erm@127.0.0.1:5432/erm?sslmode=disable \
  go run ./cmd/erm migrate --mode rollback --env ci
```

Run this after smoke tests when you need to exercise rollback logic, or add a nightly job that verifies rollback coverage.

## Smoke Tests

The `testing` package exposes a `Sandbox` helper backed by `pgxmock` and the generated ORM client. The CI workflow includes a `go test ./testing/...` step which executes `TestSandboxSmoke` to ensure the helpers wire up correctly. Extend this suite with domain-specific smoke tests for your project.

## Secrets and Configuration

- Declare environment profiles in `erm.yaml` under `database.environments`.
- Use `ERM_ENV` to choose the target profile at runtime (defaults to `dev`).
- Inject secrets through `ERM_DATABASE_URL` so the config file can remain checked in without credentials.
- For GitHub Actions, store the DSN in an encrypted secret and export it before running plan/apply/rollback steps.
