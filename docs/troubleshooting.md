# Troubleshooting

Use this playbook to diagnose issues across the CLI, schema generation, GraphQL runtime, and infrastructure. Each section lists
symptoms, root causes, and remediation steps.

---

## CLI Issues

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| `erm gen` exits with “no module found” | `go.mod` missing or `erm.yaml.module` mismatch | Run `go mod init` and update `erm.yaml` to match. |
| `erm new` fails with permission error | Running outside repository or lacking write access | Execute inside repo root and check filesystem permissions. |
| Generated files keep changing | Editor modifies imports or whitespace | Run `gofmt` on schema files; avoid manual edits to generated files. |
| `erm graphql init` overwrites custom code | Modifications made inside generated files | Move custom logic to `_extension.go` files before re-running command. |

---

## Schema & Migration Errors

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| Migration fails with duplicate column | Field renamed without removing old column | Edit migration to drop old column or rename field using `.StorageKey`. |
| `pq: relation already exists` during `erm gen` | Running migrations against database that already has tables | Point `ERM_DATABASE_URL` to a fresh database for generation checks or skip DB validation with `--no-db-check`. |
| `edge not found` compilation error | Missing `Ref`/`Inverse` in edge definition | Ensure both sides of relationships are defined or use `ManyToMany`. |
| Privacy compilation errors | Invalid expressions or typos in policy | Run `erm gen` to get precise error messages; verify viewer properties exist. |

---

## GraphQL Runtime Problems

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| `node(id:)` returns null | Node registry missing type or ID malformed | Check base64 format `<Type>:<uuidv7>` and ensure entity registered. |
| Connections return empty edges | Pagination filters exclude results | Verify `after`/`before` cursors and orderings; inspect SQL via debug logging. |
| `INTERNAL` errors without detail | Resolver panicked or unexpected DB error | Enable debug logs, wrap custom logic with error handling, inspect traces. |
| Mutations silently fail | Privacy rules deny updates | Inspect policy evaluation logs; adjust `Policy()` or viewer roles. |

---

## Authentication Problems

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| `UNAUTHENTICATED` on every request | Missing/invalid `Authorization` header | Provide `Bearer <token>`; check token issuer/audience. |
| Roles missing in context | Claims mapper not configured | Update `erm.yaml` `claims_mapper` or implement custom mapper. |
| JWKS fetch timeout | OIDC issuer unreachable | Validate network access; configure `jwks_cache_ttl` and fallback to cached keys. |
| Token accepted locally but rejected in prod | Clock skew or TLS issues | Sync clocks (NTP) and verify HTTPS certificates. |

---

## Database & Infrastructure

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| `connection refused` | Postgres not running or DSN incorrect | Start database, verify credentials, ensure network routes open. |
| Slow migrations | Running with small `work_mem` or lacking indexes | Tune Postgres parameters and review migration SQL. |
| Extension creation fails | Database role lacks `CREATE` privilege | Install extensions manually or run migrations as superuser. |

---

## Observability & Logging

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| Missing metrics at `/metrics` | Endpoint disabled or blocked | Check `observability.metrics.enabled` in `erm.yaml` and middleware chain. |
| Traces not exported | OTLP endpoint not configured | Set `OTEL_EXPORTER_OTLP_ENDPOINT` and verify network connectivity. |
| Logs show `<redacted>` unexpectedly | Field marked `.Sensitive()` | Remove `.Sensitive()` or expose via custom resolver when safe. |

---

## Production Incident Playbook

When latency spikes or errors surface after deployment, anchor the investigation around the workspace editorial scenario described in
[`examples/blog/walkthroughs/error-handling.md`](../examples/blog/walkthroughs/error-handling.md):

1. **Reproduce with recorded queries** – Capture the GraphQL payload from structured logs (`request_id`, `viewer_id`) and replay it with
   your chosen CLI (for example `erm gql replay --request-id <id>`) so dataloaders and privacy policies execute exactly as they did in production.
2. **Inspect query plans** – Enable `ERM_OBSERVABILITY_DEBUG=1` and rerun the profiling walkthrough to emit SQL and planner output.
   Compare against the stored baseline in your observability dashboard; divergences usually point to missing indexes or bloated predicates.
3. **Validate edge constraints** – Use the sample `Membership` schema's unique index to confirm no duplicate workspace memberships were inserted.
   If the incident trace shows `FOREIGN KEY` violations, run the walkthrough's validation steps to reapply cascading deletes and regenerate migrations.
4. **Capture remediation** – Update the `examples/blog/README.md` playbook with the findings (new index, privacy tweak, or resolver guard) so future
   responders start with a proven checklist.

This incident template keeps the team aligned on root cause analysis: reproduce with production data, inspect ORM-generated SQL, and verify schema
constraints before shipping hotfixes.

---

## Debugging Tips

- Run `erm gen --verbose` to print file diffs and execution details.
- Use `ERM_LOG_LEVEL=debug` to surface ORM queries and policy evaluations.
- Enable SQL logging in Postgres (`log_statement=all`) temporarily when debugging migrations.
- Capture GraphQL request/response payloads in tests using `testutil.RecordResponse`.

---

If you encounter new issues, update this document and reference logs, stack traces, and remediation steps so the whole team benefits.
