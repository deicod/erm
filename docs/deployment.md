# Deployment Playbooks

This playbook documents the recommended process for promoting schema changes and application builds across environments. It builds on the `erm migrate` workflow and the Postgres-first deployment model used by the project.

## Environment Profiles

The CLI understands database profiles declared in `erm.yaml` under `database.environments`. Each profile overrides the base connection URL and pool settings. We standardise on three profiles:

- `dev` for personal sandboxes and preview stacks.
- `staging` for pre-production verification.
- `prod` for customer-facing workloads.

Select a profile at runtime with `--env` (or `ERM_ENV`). The flag transparently falls back to `database.url` when a dedicated profile is not declared, allowing gradual adoption of the new structure. Secrets supplied via `ERM_DATABASE_URL` override every profile so pipelines can inject credentials without editing the config file.

```bash
# Preflight the staging database without writing any data.
go run ./cmd/erm migrate --mode plan --env staging

# Apply a batch of migrations against production.
go run ./cmd/erm migrate --mode apply --env prod

# Roll back the most recent migration in dev (requires *_down.sql files).
go run ./cmd/erm migrate --mode rollback --env dev
```

## Read Replicas

- Add replica connection strings under `database.replicas` in `erm.yaml`. Each entry can declare `name`, `url`, `read_only`, and `max_follower_lag` to describe the target node.
- Optional `database.routing.policies` map friendly names to routing preferences (lag thresholds, fallback behaviour). Call `db.UseReplicaPolicies(defaultPolicy, policies)` after `pg.ConnectCluster` to register them.
- At call sites, use `pg.WithReplicaRead(ctx, pg.ReplicaReadOptions{MaxLag: 5 * time.Second})` or `pg.WithReplicaPolicy(ctx, "reporting")` to opt in to read scaling. Mutations and explicit transactions continue to hit the primary via `db.Writer()`.
- CI and local workflows that do not provision replicas can keep the section empty—`ConnectCluster` gracefully operates in primary-only mode.

## Release Checklist

1. **Cut a release branch.** Regenerate code (`erm gen`) and ensure `go test ./...` passes.
2. **Preflight schema changes.** Run `erm migrate --mode plan` in dev and staging. Resolve any reported schema drift before continuing.
3. **Back up production.** Create a physical or logical backup before running destructive changes. Automate this inside your orchestration platform where possible.
4. **Apply migrations.** Execute `erm migrate --mode apply --env <profile>` from a trusted CI job or release engineer workstation.
5. **Run smoke tests.** Use the `testing` sandbox helpers to validate high-value queries and inserts. The GitHub Actions workflow under `.github/workflows/ci.yml` contains a reusable example.
6. **Observe and verify.** Monitor database telemetry (locks, replication lag) and application health dashboards for at least one full billing cycle.
7. **Promote artifacts.** Tag the release, publish binaries, and update downstream services.

## Rollback Procedure

1. **Trigger rollback mode.** Execute `erm migrate --mode rollback --env <profile>`. The CLI will halt if it cannot find the matching `*_down.sql` script or if schema drift is detected.
2. **Re-run smoke tests.** Validate critical flows using the same helpers you used post-deploy.
3. **Restore backups when rollback is insufficient.** If a destructive migration lacks a rollback script, restore from the pre-deployment backup and re-run migrations up to the desired safe point.
4. **Post-mortem.** Document the root cause, amend the migration playbooks, and add missing rollback scripts before retrying the deployment.

## Secrets Management

- Never commit plaintext credentials. Use the `ERM_DATABASE_URL` environment variable (or a secrets manager such as GitHub Actions encrypted secrets, Hashicorp Vault, or AWS Secrets Manager) to supply runtime credentials.
- Grant the CI runner a dedicated, least-privilege database role with the ability to run migrations and smoke tests.
- For production, prefer short-lived credentials issued by your secrets platform. Refresh them before running plan/apply/rollback to avoid runtime failures.

## Operational Tips

- Keep rollback scripts (`*_down.sql`) in the same `migrations/` directory so that automated checks can validate their presence.
- Use `--mode plan` during code review to attach the pending migration list to pull requests.
- Configure database connection pool settings per environment (e.g., smaller pools for staging). The CLI honours the profile-specific settings before falling back to `database.pool`.
- Always run `erm migrate --mode plan` in CI before `--mode apply` to detect drift early.
