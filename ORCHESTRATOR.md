# Orchestrator

A lightweight human-in-the-loop workflow that assigns tasks to agents, tracks branches, and sequences dependent work.

## Process

1. **Queue tasks** in GitHub Projects with labels for role and dependency.
2. **Spawn up to 9 agents** in parallel if tasks are independent (no shared files or generator templates). Examples: OIDC middleware and PostGIS helpers can run in parallel; ORM templates and GraphQL connection builders should wait for schema core.
3. **One task → one branch**, PR opened early as draft.
4. **CI runs** (build, lint, test, `erm gen` smoke).
5. **Review & merge** to `main` → unblock dependents.
6. **Autosync** docs/examples; ensure golden snapshots updated.

## Conflict Avoidance

- Templates and generator core are **serialized**.
- Feature branches touching the same template file must be staged as separate PRs and merged sequentially.
- Use **feature flags** in config to keep generated code stable (no churn).

## Required Checks (before merge)

- `go vet`, `staticcheck`, `gofmt -s` clean.
- Unit tests + integration tests (Postgres service).
- `erm gen` reproducible (no diff on second run).

## GitHub Tips

- Protected `main`, required PR review (at least 1), required status checks.
- Labels + CODEOWNERS route to the right reviewers.
