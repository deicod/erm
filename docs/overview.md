# Overview

This chapter introduces the philosophy, components, and lifecycle of **erm** so that architects and senior engineers can reason
about how the generated pieces fit together. Treat it as the conceptual foundation before diving into the DSL or CLI specifics.

---

## Why erm Exists

Building a production-grade GraphQL backend in Go is traditionally a multi-week effort that spans ORM modeling, database
migrations, dataloaders, Relay compliance, authentication, and testing. erm compresses that effort into a schema-first workflow
where a single Go DSL definition becomes the source of truth for:

- PostgreSQL schema evolution (DDL, indexes, triggers, extension enablement).
- Type-safe ORM packages that expose fluent builders, predicate helpers, eager loading, and privacy enforcement.
- Relay-compliant GraphQL schemas, resolvers, Node registry, connections, and pagination utilities.
- Operational glue: migrations, observability hooks, tracing spans, request logging, and CLI automation.

By standardizing architecture and code layout, erm makes it easy for humans and AI assistants to collaborate on large-scale
features without diverging from proven patterns.

---

## Architectural Layers

The erm runtime separates concerns across predictable directories and packages:

1. **Schema DSL (`internal/orm/schema`)** – Hand-authored Go files describe entities, fields, edges, indexes, mixins,
   annotations, hooks, interceptors, privacy policies, and constraints.
2. **ORM Packages (`internal/orm/<entity>`)** – Generated code that includes builders, query APIs, mutation helpers,
   transaction scaffolding, predicates, eager-loading support, `Edges` structs, and instrumentation hooks.
3. **GraphQL Layer (`internal/graphql`)** – Generated gqlgen schema/resolver glue, dataloader registrations, Node resolver, and
   GraphQL-specific annotations from the DSL.
4. **Security Layer (`internal/oidc`)** – OIDC middleware verifying JWTs via discovery documents, JWKS caching, and claims
   mapping; integrates with GraphQL via `@auth` directives and context injection.
5. **Observability (`internal/observability`)** – Tracing, metrics, structured logging, N+1 detection, and connection pool
   monitors that the generators wire into resolvers and ORM interceptors.
6. **Extensions (`internal/orm/pgxext`)** – Specialized mixins and field types for PostGIS geometries, pgvector embeddings, and
   TimescaleDB hypertables, along with migration helpers.

The generated CLI tasks (`erm gen`, `erm graphql init`, etc.) orchestrate these layers so a change in the schema DSL fans out to
every dependent artifact deterministically.

---

## Request Lifecycle: End-to-End Example

The following flow demonstrates how a GraphQL query travels through an erm service. Understanding this pipeline helps when
customizing hooks or debugging performance issues.

1. **HTTP Entry (cmd/server)** – The generated HTTP server receives a GraphQL request. Middleware validates JWTs and attaches a
   `context.Context` enriched with identity information and request metadata.
2. **GraphQL Resolver** – The gqlgen resolver (generated in `internal/graphql/resolver`) translates the selection set into ORM
   operations. Field resolvers automatically register dataloaders to prevent N+1 issues.
3. **Privacy Evaluation** – Before executing ORM queries/mutations, privacy rules generated from schema annotations evaluate the
   viewer and operation type, short-circuiting unauthorized access.
4. **ORM Execution** – Query builders leverage `pgx/v5` via connection pools configured in `internal/orm/runtime`. Hooks and
   interceptors instrument spans, apply auditing mixins, or enforce domain invariants.
5. **Result Assembly** – Loaded entities populate `Edges` structs so resolvers can reuse data without additional database hits.
   Global IDs are encoded as `base64("<Type>:<uuidv7>")` before returning to the client.
6. **Response Emission** – Observability middleware records metrics (timings, request/response sizes, dataloader cache stats)
   and returns the JSON payload.

---

## Project Layout Deep Dive

```
.
├── cmd/
│   └── server/              # GraphQL HTTP server entrypoint (generated + extendable)
├── docs/                    # This portal
├── internal/
│   ├── graphql/             # gqlgen config, resolvers, dataloaders, Node registry
│   ├── observability/       # Logging/tracing integrations, metrics collectors
│   ├── oidc/                # JWT verification, claims mapper interfaces, context helpers
│   ├── orm/                 # Generated ORM packages with schema subdirectory for DSL definitions
│   │   └── schema/          # Hand-authored entity schemas, mixins, annotations
│   └── pkg/                 # Optional custom packages used by both generated and handwritten code
├── migrations/              # Versioned SQL migrations generated during `erm gen`
├── schema/                  # (Optional) GraphQL SDL if you author custom SDL alongside generation
└── erm.yaml                 # Project configuration consumed by the CLI
```

Key layout guarantees that AI tooling relies on:

- Generated files include a prominent header discouraging manual edits; customizing behavior happens via schema annotations or
  resolver extension stubs.
- Every entity lives in its own package under `internal/orm`, and the CLI updates package imports automatically when you add
  mixins or fields.
- `erm.yaml` stores module path, database DSN, OIDC issuer, and code generation toggles; CLI commands read it to avoid repeated
  prompts.

---

## Configuration Surfaces

All environment-specific values are centralized in `erm.yaml` and the generated `.env.example`. Highlights include:

- **Database Block** – DSN, migration directory, connection pool sizes, SSL mode.
- **GraphQL Block** – Listen address, enablement of GraphQL Playground in development, tracing sampling rate.
- **OIDC Block** – Issuer URL, audience, JWKS refresh intervals, custom claim mapper names.
- **Extensions Block** – Flags controlling PostGIS/pgvector/TimescaleDB mixins so migrations include extension DDL automatically.

Changes in configuration trigger regeneration so that runtime packages pick up new defaults. Always run `erm gen` after editing
`erm.yaml` to keep generated code synchronized.

---

## Roadmap Snapshot

erm is evolving quickly. The roadmap tracked in `ROADMAP.md` highlights milestones like enhanced migration tooling, schema lint
commands, and richer `erm doctor` diagnostics. Each milestone feeds back into this documentation with change logs and upgrade
instructions. When a new feature lands, the corresponding guide (CLI, schema, or observability) adds sections marked with `🚀 New`
so teams can spot updates at a glance.

---

## Additional Resources

- **[Quickstart Guide](./quickstart.md)** for a hands-on introduction.
- **[Schema Definition Guide](./schema-definition.md)** for the full DSL reference.
- **[CLI Reference](./cli.md)** for every subcommand and flag.
- **[Troubleshooting](./troubleshooting.md)** when you encounter errors during generation or deployment.

Use this overview as a map to keep context while diving deep into each specialized chapter.
