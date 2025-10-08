# Product Requirements Document — erm

## Vision

**erm** generates an opinionated Go backend that fully implements the **Relay GraphQL Server Specification** (Node interface, global ObjectIDs, connections/edges with opaque cursors), on top of a schema-as-code ORM with code generation (ent-like), targeting **PostgreSQL** via **pgx/v5** with first-class support for **PostGIS**, **pgvector**, and **TimescaleDB**. Security is enforced with **OIDC** middleware (Keycloak-optimized claims mapping by default), with pluggable claim mappers per provider (Okta/Auth0/custom).

The experience must be **great for senior developers** while **AI-friendly** for rapid iteration and parallel agent work.

## Primary Users
- Senior Go/GraphQL backend engineers building new services quickly.
- Teams standardizing on Postgres, Relay-compliant GraphQL, and OIDC with Keycloak.
- AI/agent-enhanced teams that want to farm out parallel tasks safely.

## Goals (Must-have)
1. **Relay completeness**: Global `Node` IDs, cursor pagination (connections/edges), `PageInfo`, `node(id:)`, and type-resolved ID marshalling.
2. **Schema-as-code ORM** with codegen: fields, edges, indexes, views, mixins, annotations; CRUD; hooks; interceptors; privacy policies (allow/deny with reason); transactions; eager loading; graph traversal; predicates; aggregations; paging/ordering.
3. **PostgreSQL-first** with **pgx/v5**: ergonomic query builder and typed scanning; connection pooling; context-aware tracing; sensible timeouts; statement caching.
4. **Migrations**: versioned + (optionally) automatic; extension enablement; zero-downtime guidance.
5. **UUID v7 for IDs** (app-generated) and **global ObjectID** encoding for Relay (base64 `<Type>:<uuidv7>` by default).
6. **OIDC** middleware: RS256/ES256 verification, JWKS refresh, issuer/audience validation; **Keycloak** default claim mapping; pluggable mappers.
7. **CLI (`erm`)**: `init`, `new <Entity>`, `gen`, `graphql init`. Idempotent and incremental.
8. **DX**: linted templates, readable generated code, rich comments, consistent naming, `make` targets, CI (build/test/lint), devcontainer.
9. **AI-friendly**: clear task breakdown, prompts, orchestrator, PR review rules, structured docs for agents.
10. **Performance**: dataloaders; n+1 detection; query hints; indexes from schema; pgx batch/Copy where relevant.

## Non-Goals (v1)
- Multi-DB support (MySQL/SQLite); keep Postgres-only at v1.
- Full-blown admin UI; expose GraphQL Playground only.
- Federation; keep single graph v1 (allow later).

## Success Metrics
- `erm new + gen` to first GraphQL server: **< 10 minutes** from blank repo.
- End-to-end tests generate + run with CI green on default project.
- 95th percentile GraphQL query latency for a typical read-list (100–1,000 rows) under **p95 < 50ms** on local Postgres with indexes present (guidance + benchmarks).

## Constraints
- Go **1.22+** (toolchain)-compatible.
- Stable deps only (pgx v5, gqlgen stable, go-oidc v3, jwx v2).

## Risks & Mitigations
- **Relay correctness**: Provide conformance tests + example repo.
- **Provider variety (OIDC)**: pluggable claim mappers + test fixtures.
- **Migration safety**: versioned migrations + extension checks + advisory locks.
- **DX confusion**: clear docs, examples, templates, `erm doctor` later.
