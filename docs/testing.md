# Testing Guide

Reliable tests make it safe to evolve schemas, generated clients, and GraphQL resolvers. The `erm` repository ships with helpers
in `testing` that keep suites fast while still exercising the generated layers end-to-end. This guide outlines the
available utilities, common patterns, and CI recommendations.

---

## Testing Utilities

### Postgres Sandbox

`testkit.NewPostgresSandbox(tb)` provisions a cancellable context, a pgxmock-backed connection, and a thin `*pg.DB` wrapper so you
can execute generated ORM code without a running database.

```go
sandbox := testkit.NewPostgresSandbox(t)
defer sandbox.ExpectationsWereMet(t)

ctx := sandbox.Context()
client := sandbox.ORM(t)
mock := sandbox.Mock()

mock.ExpectQuery(`INSERT INTO users ... RETURNING id, created_at, updated_at`).
    WithArgs("user-1", pgxmock.AnyArg(), pgxmock.AnyArg()).
    WillReturnRows(mock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("user-1", time.Now(), time.Now()))

user, err := client.Users().Create(ctx, &gen.User{ID: "user-1"})
require.NoError(t, err)
require.Equal(t, "user-1", user.ID)
```

The sandbox stays entirely in memory, making it suitable for unit tests and quick feedback loops. When you need to exercise
multi-row results, pagination, or error branches, set up the appropriate expectations on the returned mock.

### Validation Registry

Validation rules live on the generated `gen.ValidationRegistry`. Reset it inside tests so assertions remain isolated:

```go
gen.ValidationRegistry = validation.NewRegistry()
t.Cleanup(func() { gen.ValidationRegistry = validation.NewRegistry() })

gen.ValidationRegistry.Entity("User").
    OnCreate(validation.String("Email").Required().Matches(emailRegex).Rule()).
    OnUpdate(validation.RuleFunc(func(_ context.Context, subject validation.Subject) error {
        created, _ := subject.Record.Time("CreatedAt")
        updated, _ := subject.Record.Time("UpdatedAt")
        if updated.Before(created) {
            return validation.FieldError{Field: "UpdatedAt", Message: "must be after CreatedAt"}
        }
        return nil
    }))

_, err := client.Users().Create(ctx, &gen.User{Email: "bad"})
require.Error(t, err)
```

Use the `orm/runtime/validation` helpers to build string, regex, and cross-field checks; they run automatically inside `Create`/`Update` once registered.

### GraphQL Harness

`testkit.NewGraphQLHarness(tb, testkit.GraphQLHarnessOptions{ORM: client})` wraps the generated executable schema with gqlgen's test
client and automatically wires request-scoped dataloaders.

```go
orm := sandbox.ORM(t)
harness := testkit.NewGraphQLHarness(t, testkit.GraphQLHarnessOptions{ORM: orm})

var resp struct {
    Users struct {
        TotalCount int
    }
}

mock.ExpectQuery(`SELECT COUNT(*) FROM users`).WillReturnRows(mock.NewRows([]string{"count"}).AddRow(0))
mock.ExpectQuery(`SELECT id, created_at, updated_at FROM users ORDER BY id LIMIT $1 OFFSET $2`).
    WithArgs(1, 0).
    WillReturnRows(mock.NewRows([]string{"id", "created_at", "updated_at"}))

harness.MustExec(t, context.Background(), `query { users(first: 1) { totalCount } }`, &resp)
require.Zero(t, resp.Users.TotalCount)
```

Because `MustExec` injects dataloaders for every request, the harness mirrors production behaviour without spinning up an HTTP
server.

---

> **Looking for real-world patterns?** The [editorial workspace walkthroughs](../examples/blog/walkthroughs/validation.md)
> pair these helpers with concrete validation, profiling, and error-handling suites so you can copy the structure into your own
> projects.

---

## Mocking Patterns

- Use `pgxmock.AnyArg()` for timestamps or automatically generated UUIDs.
- Return multiple rows via `mock.NewRows(...).AddRow(...).AddRow(...)` to test pagination.
- Call `sandbox.ExpectationsWereMet(t)` at the end of the test (or rely on `t.Cleanup`) to ensure all expectations executed.
- For error branches, configure `WillReturnError` on the relevant query or exec expectation and assert that the ORM method returns
  the expected error.

---

## Integration Tests with Real Postgres

When you need to validate migrations or query planners, point the ORM at a real database instead of the sandbox:

```go
func TestUserRoundTrip(t *testing.T) {
    ctx := context.Background()
    db, err := pg.Connect(ctx, os.Getenv("TEST_DATABASE_URL"))
    require.NoError(t, err)
    t.Cleanup(db.Close)

    client := gen.NewClient(db)
    // Apply migrations or insert fixtures as needed...
}
```

Combine this with Docker or Testcontainers in CI to spin up disposable Postgres instances. Keep the fast pgxmock-backed tests as
your primary safety net and reserve live database tests for cross-cutting concerns (migrations, connection pool tuning, etc.).

---

## Race Detector Workflow

`go test -race` instruments every load and store, so plan for the suite to take substantially longer than a normal run. On an
8-core Apple Silicon laptop the full repository usually finishes in **7–9 minutes**, while our CI runners land closer to
**12–15 minutes**. Use it as an overnight or pre-merge check for concurrency-heavy changes instead of running it on every edit.

### When to run it

- Before merging changes that touch goroutine orchestration, background workers, or shared caches.
- After large refactors that move code between packages (the detector will also flush the build cache).
- At least once per workday while iterating on the generator or runtime primitives so subtle data races do not slip through.

### How to scope it

- Prefer `erm test --race ./pkg/...` to focus on the packages you touched. The command expands patterns, batches packages eight at a time,
  and reuses Go's build cache so reruns skip work that already passed.
- For one-off investigations you can pass explicit package lists: `erm test --race ./orm/... ./graphql/...`.
- Need the legacy workflow? `erm test` without `--race` simply delegates to `go test` with whatever package patterns you provide.

### Caching tips

- Because the helper runs in batches you can re-run only the failing chunk—subsequent batches that are still cached finish instantly.
- CI jobs should set `GOCACHE` to a persistent volume so the expensive instrumented builds are reused across branches.
- If you need to invalidate state, run `go clean -cache` before invoking `erm test --race`.

### Recommended commands

- `erm test --race` – Batches the entire repository and prints each `go test -race` invocation so you can copy/paste reruns.
- `make test-race` – Convenience wrapper around `go run ./cmd/erm test --race` for developers who prefer the Makefile entry points.

---

## CI Recommendations

1. Run `go test ./...` to execute both sandbox-based unit tests and live integration suites.
2. Add a job that runs `erm gen --dry-run` to detect schema drift before migrations fail.
3. Cache the Go build and module downloads to keep feedback loops tight.
4. Surface GraphQL examples (queries/mutations) in CI logs when failures occur to make debugging easier.

Following these patterns keeps the test pyramid balanced: fast feedback from mocked helpers, confidence from targeted integration
coverage, and reproducible workflows in continuous integration.
