# Performance & Observability

erm-generated services include built-in hooks for tracing, metrics, logging, and profiling. This guide explains the knobs you can
tune to keep latency low, detect N+1 issues, and observe behavior in production.

---

## Migration caveats

- Foreign keys with explicit cascade policies (`OnDeleteCascade`, `OnUpdateRestrict`, etc.) are rendered directly into the SQL migrations. Plan for the additional write amplification on large tables and ensure background workers account for the cascaded deletes.
- Polymorphic edges only annotate runtime metadata. The generator still emits concrete foreign keys, so keep discriminator predicates lightweight (usually an indexed column).
- Regenerate migrations after changing cascade semantics—existing databases require manual `ALTER TABLE ... DROP CONSTRAINT ...` before the new clause can be applied.
- Generated columns are treated as read-only; the diff engine drops and recreates them when the expression changes because PostgreSQL cannot alter the definition in-place. Schedule migrations during maintenance windows or provide a hand-written migration if the table is large.

## Connection Pooling

- ORM uses `pgxpool.Pool`. Configure via `erm.yaml`:

```yaml
database:
  url: postgres://...
  pool:
    max_conns: 50
    min_conns: 5
    max_conn_lifetime: 30m
    max_conn_idle_time: 5m
    health_check_period: 30s
```

- Use annotations to override per-entity transaction isolation if necessary (`dsl.TransactionIsolation(...)`).
- Monitor pool stats via the `observability/metrics` package (exported to Prometheus by default).

The pgx wrapper now exposes a consolidated `pg.WithPoolConfig` option, so CLI consumers can translate the YAML block into runtime tuning without sprinkling individual setters throughout the codebase.

## Replica Routing & Health Checks

- Declare replicas in `erm.yaml` under `database.replicas` and optionally name routing policies under `database.routing`. The CLI loads the definitions so your bootstrap code can call `db.UseReplicaPolicies(default, policies)` after connecting.
- Use `pg.WithReplicaRead(ctx, pg.ReplicaReadOptions{MaxLag: 5 * time.Second})` to opt an individual request into replica reads. Combine with `pg.WithReplicaPolicy(ctx, "reporting")` to reuse policy definitions, or `pg.WithPrimary(ctx)` to pin a specific call to the writer even when a default policy is active.
- The driver keeps distinct pools for writer and replicas. Health probes (`SELECT pg_is_in_recovery()...`) run on demand and respect both the replica-level `max_follower_lag` and per-read `MaxLag` settings. Override the interval with `db.SetReplicaHealthInterval` or the probe implementation with `db.UseReplicaHealthCheck` when targeting managed services that expose custom views.
- Telemetry now includes span/log attributes: `orm.target` (primary/replica name), `orm.replica` (boolean), `orm.failover`, `orm.failover_reason`, and `orm.health_check`. Metrics fan out through the existing `runtime.QueryObserver`, so failovers show up as additional query events tagged with `orm.failover=true`.
- When a replica errors or violates lag/read-only guarantees, the driver retries against the writer unless `ReplicaReadOptions.DisableFallback` is set. Aggregates handle retry within the returned row wrapper so callers simply invoke `Scan`.

---

## Dataloaders & N+1 Detection

- Every to-many edge registers a dataloader with batching and caching. Override defaults:

```go
dsl.ToMany("comments", "Comment").
    Ref("post").
    Dataloader(dsl.Loader{Batch: 200, Wait: 2 * time.Millisecond, Cache: true})
```

- Enable N+1 logging by setting `ERM_OBSERVABILITY_DEBUG=1`. The resolver layer reports when an edge is loaded without using the
  dataloader.
- Use the generated benchmark stubs in `benchmarks/` to profile dataloader behavior under load.

## Bulk Operations & Streaming

- Generated ORM clients include `BulkCreate`, `BulkUpdate`, and `BulkDelete` helpers. They rely on new runtime builders to issue
  `INSERT ... VALUES (...)` and `UPDATE ... FROM data` statements with predictable placeholder ordering.
- Pair the helpers with the streaming iterator API: `Query().Stream(ctx)` returns a `runtime.Stream[T]` that scans rows lazily so
  long-running exports can process records without holding the entire result set in memory.
- Benchmarks under `benchmarks/orm` cover the SQL builders—run `go test -bench BuildBulk ./benchmarks/orm` to measure planner
  throughput as you tune batch sizes.

## Cache Pluggability

- ORM clients expose `Client.UseCache(cache.Store)`. Implement the `Store` interface from `orm/runtime/cache` to plug
  in Redis, in-memory LRU, or a no-op adapter. Primary-key lookups automatically populate or invalidate the cache across
  create/update/delete (including bulk APIs).
- Keep TTL logic inside your `Store` implementation; the ORM only deals with keys shaped like `orm:<Entity>:<id>`.

---

## Tracing

- OpenTelemetry instrumentation is wired into resolvers, ORM queries, and hooks via `observability/tracing`.
- Configure exporters in `cmd/server/main.go` or via environment variables (`OTEL_EXPORTER_OTLP_ENDPOINT`).
- Spans include attributes: entity name, operation (`create`, `query`, `update`), row counts, and dataloader batch sizes.

Example instrumentation snippet (generated):

```go
ctx, span := tracing.Start(ctx, "orm.user.query")
defer span.End()
res, err := next(ctx)
if err != nil {
    span.RecordError(err)
}
return res, err
```

---

## Metrics

Prometheus counters/gauges/histograms live in `observability/metrics` and are registered via the HTTP server.

Key metrics:

| Metric | Description |
|--------|-------------|
| `erm_graphql_requests_total` | Total GraphQL requests labeled by operation type. |
| `erm_graphql_request_duration_seconds` | Histogram of GraphQL request latency. |
| `erm_dataloader_batch_size` | Histogram of dataloader batch sizes per edge. |
| `erm_db_queries_total` | Count of ORM queries executed, labeled by entity and operation. |
| `erm_db_query_duration_seconds` | Histogram for SQL execution time. |

Enable metrics endpoint by default at `/metrics`. Lock it down via reverse proxy or middleware in production.

---

## ORM Query Instrumentation

`orm/runtime.QueryObserver` fans structured logs, tracing spans, and metrics events into the observability stack. The observer cooperates with `observability/metrics` and `observability/tracing`, so you can opt into telemetry features by wiring the components together in one place:

```go
observer := runtime.QueryObserver{
    Logger: runtime.QueryLoggerFunc(func(ctx context.Context, entry runtime.QueryLog) {
        logger.Infow("orm query", "table", entry.Table, "operation", entry.Operation, "duration", entry.Duration, "error", entry.Err, "correlation_id", entry.CorrelationID)
    }),
    Tracer:    tracing.WithTracer(tracing.NewOTelTracer(provider, "my-service")),
    Collector: metrics.WithCollector(appCollector),
    Correlator: runtime.CorrelationProviderFunc(func(ctx context.Context) string {
        return requestid.FromContext(ctx)
    }),
}
db.UseObserver(observer)
```

- **Query logging** – Controlled by `observability.orm.query_logging`. Leave the logger `nil` when disabled to avoid allocations.
- **Span emission** – Toggle via `observability.orm.emit_spans`. When `false`, the tracer can remain `nil` and no spans are produced.
- **Correlation IDs** – Enable `observability.orm.correlation_ids` to enrich logs/spans with request identifiers. Provide a `CorrelationProvider` that extracts the value from your context chain.

Because the observer copies SQL arguments before logging, downstream mutations cannot leak secrets or identifiers by accident. The span attributes always include `orm.table`, `orm.operation`, and `orm.arg_count` so trace backends can filter by entity without parsing log lines.

---

## Logging

- Structured logging uses `zap` with fields for request ID, viewer ID, and GraphQL operation name.
- Set `ERM_LOG_FORMAT=json` for structured logs or leave blank for console output.
- Sensitive fields (marked `.Sensitive()`) are redacted automatically in logs.
- Use `authz.WithViewer` to inject synthetic viewers in admin scripts to ensure logs contain proper metadata.

---

## Profiling & Benchmarks

- `cmd/server` exposes `pprof` endpoints when `ERM_ENABLE_PPROF=1`. Access via `/debug/pprof/`.
- Benchmarks live in `benchmarks/` with examples for ORM queries, dataloader loads, and GraphQL operations. Run via `go test -bench=. ./benchmarks/...`.

Example benchmark snippet:

```go
func BenchmarkUserQuery(b *testing.B) {
    client := benchmarks.NewClient(b)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = client.User.Query().Limit(10).All(context.Background())
    }
}
```

---

## Query Composition & Optimization Scenario

The blog walkthrough adds a **workspace timeline** view that renders recent posts with nested comments. The generated ORM API lets you
compose the query in one place while still controlling SQL shape:

```go
// examples/blog/walkthroughs/performance-profiling.md
func loadTimeline(ctx context.Context, client *gen.Client, workspaceID string) ([]*gen.Post, error) {
    return client.Posts().
        Query().
        WhereWorkspaceIDEq(workspaceID).
        WithAuthor().
        WithComments(func(q *gen.CommentQuery) {
            q.WhereParentIDEq(parentID).
                Limit(10).
                WithReplies(func(rq *gen.CommentQuery) { rq.Limit(5) })
        }).
        OrderByCreatedAtDesc().
        Limit(20).
        All(ctx)
}
```

In the example above, `parentID` can be set to an empty string to request top-level comments or to a specific comment ID when
you want to expand a nested thread.

### Optimize the Call Stack

1. **Validate limits** – The `Post.Query()` definition in the sample enforces `WithDefaultLimit(20)` and `WithMaxLimit(200)` so an unexpected
   GraphQL argument cannot request an unbounded timeline.
2. **Profile dataloader batches** – Follow the profiling walkthrough's recipe (`ERM_OBSERVABILITY_DEBUG=1 go test ./examples/blog -run TestUserORMCRUDFlow`) to
   confirm that comment replies are batch loaded and no `SELECT` statements repeat per node.
3. **Capture planner output** – Run `EXPLAIN ANALYZE` for the generated SQL (see the walkthrough for commands) and store the plan alongside flamegraphs for
   later regression analysis.

The same workflow scales to other aggregates: push heavy predicates into the schema-level `Query()` specification, rely on eager-loading hooks to prefetch
edges, and restrict `WithMaxLimit` or `WithPredicates` to guide resolver authors toward cache-friendly usage.

---

## Observability Configuration in `erm.yaml`

```yaml
observability:
  tracing:
    enabled: true
    sample_rate: 0.1
  logging:
    level: info
    format: json
  metrics:
    enabled: true
  orm:
    query_logging: true
    emit_spans: true
    correlation_ids: true
  dataloader:
    log_n_plus_one: true
```

Run `erm gen` to propagate changes into `observability` packages.

---

## Alerting & Dashboards

- **Prometheus Alerts** – Use `erm_graphql_request_duration_seconds` to alert on p95 latency spikes.
- **Grafana Dashboards** – The repository includes JSON dashboards under `examples/grafana`. Import them to visualize resolver
  latency, database throughput, and dataloader efficiency.
- **Tracing Backends** – Ship spans to Jaeger, Tempo, or Honeycomb for request-level insights.

---

## Performance Tuning Checklist

1. **Indexing** – Confirm schema indexes cover frequent filters. Use `EXPLAIN ANALYZE` to validate.
2. **Batch Sizes** – Adjust dataloader batch sizes per edge based on GraphQL usage patterns.
3. **Connection Pools** – Tune `max_conns` and `max_conn_lifetime` to match workload concurrency.
4. **Caching** – Enable caching at the application layer (Redis) if certain queries are expensive and static.
5. **Extensions** – Use pgvector approximate indexes, PostGIS gist indexes, or Timescale compression to reduce IO.
6. **Profiling** – Run `go tool pprof` against CPU/heap profiles to identify hotspots.

---

## Troubleshooting

| Symptom | Mitigation |
|---------|------------|
| High latency with small workloads | Check for synchronous logging to stdout; consider async logging or lower log level. |
| Database connection exhaustion | Increase `max_conns`, ensure dataloaders do not spawn concurrent goroutines without limits. |
| Persistent N+1 warnings | Verify resolvers call the generated dataloader helpers and avoid manual ORM queries inside loops. |
| Missing metrics | Ensure `/metrics` endpoint is registered and not blocked by middleware. |
| Query logs missing correlation IDs | Enable `observability.orm.correlation_ids` and provide a `runtime.CorrelationProvider` that reads your request ID from context. |
| Spans not appearing in backend | Check that `observability.orm.emit_spans` is true and a tracer (e.g. `tracing.NewOTelTracer`) is supplied to the query observer. |

---

Continue to [testing.md](./testing.md) for strategies to validate performance-sensitive code.
