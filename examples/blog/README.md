# Editorial Workspace Blog Example

This example extends the baseline blog schema into a multi-tenant editorial workspace. It introduces join tables, scoped query
composition, and observability practices that match production workflows. Use it alongside the walkthroughs in this directory
to rehearse the full lifecycle: schema validation, performance profiling, and error handling.

## Scenario Overview

- **Workspace** – Top-level tenant with slug-based routing, descriptive metadata, and aggregate queries.
- **Membership** – Join table that stores per-workspace roles and enforces uniqueness on `(workspace_id, user_id)`.
- **Post** – Editorial content scoped to a workspace, eager-loading comments via tuned dataloaders.
- **Comment** – Threaded replies with optional parent references and pagination defaults.

Each schema lives under [`schema/`](./schema) so you can copy/paste into a real project before running `erm gen`.

## Walkthroughs

1. **[Validation](./walkthroughs/validation.md)** – Generate migrations, assert schema constraints, and automate guardrails that
   keep tenant boundaries intact.
2. **[Performance Profiling](./walkthroughs/performance-profiling.md)** – Compose timeline queries, benchmark dataloader
   behavior, and collect `EXPLAIN ANALYZE` output for regressions.
3. **[Error Handling](./walkthroughs/error-handling.md)** – Reproduce production incidents locally, replay GraphQL payloads, and
   capture remediation notes for future responders.

Each walkthrough provides commands you can run directly from the repository root. Adjust DSNs, flags, or mock data as needed
for your environment.

## ORM Observability Wiring

`erm.yaml` now carries feature flags under `observability.orm` so you can toggle query logs, tracing spans, and correlation IDs without editing Go code. The example project enables all three so the pgx driver streams telemetry into your logging/metrics pipeline:

```go
observer := runtime.QueryObserver{
    Logger: runtime.QueryLoggerFunc(func(ctx context.Context, entry runtime.QueryLog) {
        fields := []any{
            "table", entry.Table,
            "operation", entry.Operation,
            "duration", entry.Duration,
        }
        if entry.CorrelationID != "" {
            fields = append(fields, "correlation_id", entry.CorrelationID)
        }
        if entry.Err != nil {
            fields = append(fields, "error", entry.Err)
        }
        slog.Log(ctx, slog.LevelInfo, entry.SQL, fields...)
    }),
    Tracer:    tracing.WithTracer(tracing.NewOTelTracer(provider, "examples/blog")),
    Collector: metrics.WithCollector(promCollector),
    Correlator: runtime.CorrelationProviderFunc(func(ctx context.Context) string {
        if id, ok := requestid.FromContext(ctx); ok {
            return id
        }
        return ""
    }),
}
db.UseObserver(observer)
```

When `observability.orm.query_logging` is disabled the logger can be left `nil`. Likewise, set `emit_spans` or `correlation_ids` to `false` to skip span emission or context lookups. The defaults keep behaviour backwards compatible for teams that prefer minimal telemetry.
