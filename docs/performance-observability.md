# Performance & Observability

erm-generated services include built-in hooks for tracing, metrics, logging, and profiling. This guide explains the knobs you can
tune to keep latency low, detect N+1 issues, and observe behavior in production.

---

## Connection Pooling

- ORM uses `pgxpool.Pool`. Configure via `erm.yaml`:

```yaml
database:
  url: postgres://...
  max_conns: 50
  min_conns: 5
  max_conn_lifetime: 30m
  health_check_period: 30s
```

- Use annotations to override per-entity transaction isolation if necessary (`dsl.TransactionIsolation(...)`).
- Monitor pool stats via the `internal/observability/metrics` package (exported to Prometheus by default).

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

---

## Tracing

- OpenTelemetry instrumentation is wired into resolvers, ORM queries, and hooks via `internal/observability/tracing`.
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

Prometheus counters/gauges/histograms live in `internal/observability/metrics` and are registered via the HTTP server.

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
  dataloader:
    log_n_plus_one: true
```

Run `erm gen` to propagate changes into `internal/observability` packages.

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

---

Continue to [testing.md](./testing.md) for strategies to validate performance-sensitive code.
