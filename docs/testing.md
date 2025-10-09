# Testing Guide

Reliable tests make it safe to evolve schemas, generated clients, and GraphQL resolvers. The `erm` repository ships with helpers
in `internal/testing` that keep suites fast while still exercising the generated layers end-to-end. This guide outlines the
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

## CI Recommendations

1. Run `go test ./...` to execute both sandbox-based unit tests and live integration suites.
2. Add a job that runs `erm gen --dry-run` to detect schema drift before migrations fail.
3. Cache the Go build and module downloads to keep feedback loops tight.
4. Surface GraphQL examples (queries/mutations) in CI logs when failures occur to make debugging easier.

Following these patterns keeps the test pyramid balanced: fast feedback from mocked helpers, confidence from targeted integration
coverage, and reproducible workflows in continuous integration.
