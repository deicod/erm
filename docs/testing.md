# Testing Guide

Testing ensures generated code and custom logic behave as expected. erm provides helpers for unit, integration, and end-to-end
tests so teams can ship confidently. This guide outlines strategies, tooling, and patterns.

---

## Test Pyramid

1. **Unit Tests** – Validate schema mixins, custom hooks, resolver extensions, and utility packages.
2. **Integration Tests** – Exercise generated ORM builders against a real Postgres database (use Docker or Testcontainers).
3. **GraphQL Tests** – Use the generated GraphQL client to run queries/mutations and assert responses.
4. **End-to-End Tests** – Optional; run the full server, issue HTTP requests, and inspect side effects.

---

## Setup Utilities

The repository includes helpers under `internal/testutil`:

- `testutil.NewClient(t *testing.T)` – Returns a fully configured ORM client connected to a temporary database.
- `testutil.WithTransaction(t, client, func(ctx context.Context, tx *orm.Tx) { ... })` – Runs code inside a transaction rolled back
  after the test.
- `testutil.GraphQL(t)` – Spins up an in-memory GraphQL server for request tests.
- `testutil.ContextWithViewer` – Injects authenticated viewer context for authorization scenarios.

Example integration test harness:

```go
func TestUserCreate(t *testing.T) {
    client := testutil.NewClient(t)
    ctx := context.Background()

    user, err := client.User.Create().
        SetEmail("test@example.com").
        Save(ctx)
    require.NoError(t, err)
    require.NotNil(t, user)
}
```

---

## Testing Generated Hooks and Privacy

Hooks live in generated files but extension points allow testing custom logic:

```go
func TestUserBeforeCreateHook(t *testing.T) {
    client := testutil.NewClient(t)
    ctx := context.Background()

    _, err := client.User.Create().
        SetEmail("invalid").
        Save(ctx)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "must contain @")
}
```

Privacy rules can be tested by injecting viewers:

```go
viewer := authz.Viewer{ID: "user-1"}
ctx := authz.WithViewer(context.Background(), viewer)
_, err := client.User.Query().Only(ctx)
require.ErrorIs(t, err, privacy.ErrUnauthorized)
```

---

## GraphQL Testing

Use the generated GraphQL test client:

```go
func TestGraphQLQuery(t *testing.T) {
    srv := testutil.GraphQLServer(t)
    ctx := testutil.ContextWithViewer(t, testutil.Viewer{ID: "user-1"})

    var resp struct {
        Users struct {
            TotalCount int
        }
    }

    err := testutil.Query(ctx, srv, `query { users(first: 1) { totalCount } }`, nil, &resp)
    require.NoError(t, err)
    require.Equal(t, 0, resp.Users.TotalCount)
}
```

For mutations:

```go
mutation := `mutation($input: CreateUserInput!) {
  createUser(input: $input) {
    user { id email }
  }
}`

vars := map[string]any{"input": map[string]any{"email": "qa@example.com"}}
err := testutil.Mutate(ctx, srv, mutation, vars, &resp)
```

Use snapshots (`github.com/bradleyjkemp/cupaloy`) if you want to compare full GraphQL responses.

---

## Database Migrations in Tests

- `testutil.NewClient` applies migrations automatically to a temporary database.
- For long-running integration suites, use Docker Compose to provide a persistent Postgres instance and run migrations with
  `erm gen` followed by `erm migrate` to apply the generated SQL.
- Combine `erm gen --dry-run` with CI to ensure the schema snapshot is in sync before executing migrations.
- Remember to clean up data between tests using transactions or TRUNCATE statements.

Example local workflow when developing against a shared test database:

```bash
erm gen --dry-run           # Inspect upcoming migration SQL
erm gen --name add_flag     # Materialize the migration and update the snapshot
erm migrate                 # Apply migrations using database.url from erm.yaml
go test ./internal/...      # Execute integration suite against the migrated schema
```

---

## Benchmarking

Benchmark both ORM and GraphQL layers using Go’s `testing` package.

```go
func BenchmarkGraphQLUsers(b *testing.B) {
    srv := testutil.GraphQLServer(b)
    ctx := context.Background()

    for i := 0; i < b.N; i++ {
        testutil.Query(ctx, srv, `query { users(first: 10) { totalCount } }`, nil, &struct{}{})
    }
}
```

Run benchmarks:

```bash
go test -bench=. ./internal/... ./benchmarks/...
```

---

## Testing CLI Workflows

For advanced automation, write golden tests that run CLI commands and compare output:

```go
func TestCLIGenDryRun(t *testing.T) {
    out := cli.Run(t, "gen", "--dry-run")
    require.Contains(t, out, "No changes detected")
}
```

Store golden files under `testdata/` and use `require.Equal(t, string(expected), out)` to detect regressions.

---

## Continuous Integration

Suggested CI steps:

1. `go test ./...` – Runs all unit and integration tests (set `ERM_DATABASE_URL` to a disposable DB).
2. `erm gen --dry-run` – Ensures no schema drift.
3. `erm migrate` – Applies pending migrations against the CI database before tests run.
4. `golangci-lint run` – Lint generated and handwritten code.
5. `erm doctor` (when available) – Confirms migration ordering and configuration sanity.

---

## Troubleshooting Tests

| Issue | Fix |
|-------|-----|
| Tests fail due to missing database | Export `ERM_DATABASE_URL` or use `testutil.NewClient` which spins up a temporary DB. |
| Data leakage between tests | Wrap tests in transactions or use `testutil.ResetTables`. |
| GraphQL errors obscure stack traces | Set `ERM_LOG_LEVEL=debug` to surface resolver logs during tests. |
| Benchmarks inconsistent | Disable CPU frequency scaling (set `GOMAXPROCS`), pin dependencies, and warm caches before measurement. |

---

Next, explore [best-practices.md](./best-practices.md) for workflow conventions that keep tests reliable.
