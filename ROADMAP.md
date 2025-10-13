# Roadmap & Milestones

> Source of truth for high-level delivery. Each milestone maps to a tracked GitHub Project view.
> All work merges to `main` through reviewed PRs; dependent tasks wait for upstream merges.

## Milestone 0 — Repo bootstrap (Week 0–1)
- Repo skeleton, CI, Makefile, devcontainer, linters, codeowners, templates.
- Minimal `erm` CLI with `init`, `new`, `gen`, `graphql init` stubs.
- Docs: PRD, Specs, Agents, Orchestrator.

## Milestone 1 — ORM core (Week 1–4)
- Schema DSL (Go structs + annotations).
- Codegen templates for entities, queries, mutations, hooks, interceptors, policies.
- Transactions, predicates, eager loading; typed builders; UUID v7 IDs.
- Migrations (versioned), extension enablement (PostGIS/pgvector/TimescaleDB).

## Milestone 2 — GraphQL Relay (Week 3–6, overlaps M1 tail)
- gqlgen config, with guarded autobind support once Node helpers and enum wrappers land.
- Node interface + global object ID encoder/decoder.
- Connections/cursors; PageInfo; generic connection builders.
- Dataloaders; N+1 test.

## Milestone 3 — OIDC (Week 4–6)
- Middleware verifying JWT via JWKS (go-oidc v3 + jwx v2).
- Claims mapping: Keycloak default; pluggable per-provider mappers (Okta/Auth0/custom).
- GraphQL directive `@auth` and context injection.

## Milestone 4 — Extensions & Perf (Week 6–8)
- PostGIS, pgvector, TimescaleDB types + helpers + migrations.
- Benchmarks (query, write, bulk). Index generation from schema hints.
- Connection pooling & pgx tunables; tracing hooks (OpenTelemetry stub).

## Milestone 5 — DX polish (Week 8–9)
- `erm doctor` basic, error messages, recipe docs.
- Example app(s): blog, vector-search.
- Release v0.1.0 with binaries for darwin/linux/windows (amd64+arm64).
