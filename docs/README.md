# erm Documentation Portal

Welcome to the authoritative guide for **erm**, the opinionated GraphQL + ORM code generation toolkit for Go. This portal brings
together the previously fragmented guides into a single curated knowledge base that covers every surface area of the platform—
from the schema DSL to production operations. Whether you are onboarding a new team member, wiring erm into an existing codebase,
or asking an AI pair-programmer to synthesize features, this portal provides the depth and examples required to ship with
confidence.

---

## How to Use This Portal

The documentation is organized into focused guides that can be consumed linearly or consulted à la carte. Each page mirrors the
actual workflows you will perform with erm and includes runnable snippets, CLI transcripts, and schema patterns that align with
the generated code. A recommended learning path is:

1. **[Overview](./overview.md)** – Understand the goals, architecture, and mental model of erm.
2. **[Quickstart Guide](./quickstart.md)** – Bootstrap a fresh project and take it from schema to running server.
3. **[Schema Definition Guide](./schema-definition.md)** – Explore the DSL in depth with annotated examples.
4. **[GraphQL API](./graphql-api.md)** – Learn how Relay compliance is enforced, including global Node IDs and pagination.
5. **[Authentication & Authorization](./authentication.md)** – Configure OIDC, map claims, and protect resolvers.
6. **[Extensions Support](./extensions.md)** – Add PostGIS, pgvector, or TimescaleDB capabilities to your domain.
7. **[CLI Reference](./cli.md)** – Master the ergonomics of `erm` commands and generated automation.
8. **[Performance & Observability](./performance-observability.md)** – Tune connection pools, dataloaders, and tracing.
9. **[Testing Guide](./testing.md)** – Validate generated code, custom logic, and integration flows.
10. **[Best Practices](./best-practices.md)** – Adopt conventions that keep generated and handwritten code in sync.
11. **[Troubleshooting](./troubleshooting.md)** – Diagnose schema, GraphQL, database, and deployment issues quickly.

Each guide explicitly calls out which files are generated, which are hand-authored, and how to keep AI tooling aligned with the
expected project structure.

---

## Product Overview

erm generates a full Relay-compliant GraphQL server backed by a schema-as-code ORM targeting PostgreSQL through `jackc/pgx` v5.
Instead of manually stitching resolvers, migrations, and middleware, you describe the domain model once using Go code and let
the generators synthesize:

- ORM entities with CRUD helpers, eager loading, transactions, hooks, interceptors, and privacy policies.
- GraphQL schemas, resolvers, dataloaders, Relay Node wiring, and cursor pagination utilities.
- A Postgres-first persistence layer with deterministic, versioned migrations and extension helpers.
- OIDC authentication middleware with JWKS discovery, caching, and pluggable claims mapping (Keycloak defaults included).
- Ergonomic CLI workflows (`erm init`, `erm new`, `erm gen`, `erm graphql init`) that orchestrate schema evolution and
  application scaffolding.

The design principles behind erm are:

- **DX First:** Generated code is intentionally commented and lint-friendly, with deterministic formatting to reduce diff noise.
- **Single Source of Truth:** The schema DSL drives ORM types, GraphQL surfaces, migrations, and documentation metadata.
- **Secure by Default:** Authentication hooks and privacy policies are scaffolded with sane defaults and clear extension points.
- **Production Ready:** Features like dataloaders, connection pooling, tracing hooks, and structured logging are wired in from
day one.
- **AI-Friendly:** Comments, docs, and generated TODOs are optimized so AI copilots can reason about the codebase without
  stumbling over hidden conventions.

---

## Architecture at a Glance

erm projects are structured to keep inputs (schema, config) separate from generated artifacts while maintaining predictable
paths for tooling:

```
.
├── cmd/                # Entry points (e.g. GraphQL server) generated and/or extended by developers
├── docs/               # This portal, synced with CLI scaffolding for onboarding
├── cli/                # CLI implementation backing `erm init`, `erm gen`, and friends
├── generator/          # Code generation engine shared by the CLI and tests
├── graphql/            # Generated resolvers, dataloaders, Relay node registry, gqlgen configuration
├── observability/      # Tracing integrations, logging adapters, request metrics
├── oidc/               # OIDC verifier, JWKS cache, claims mappers, @auth directive helpers
├── orm/                # Generated ORM packages; `schema/` subdirectory contains the hand-authored DSL
├── testing/            # Integration harnesses for ORM + GraphQL flows
├── migrations/         # SQL migration files emitted during `erm gen`
├── schema/             # (Optional) GraphQL SDL stubs when authoring manually managed schemas
└── erm.yaml            # Project configuration consumed by the CLI and generators
```

The layered architecture consists of:

1. **Schema Definitions** – Go files in `orm/schema` describe entities, fields, edges, indexes, mixins, and annotations.
2. **Code Generation** – `erm gen` materializes ORM packages, GraphQL types/resolvers, privacy rules, loader registries, and
   migrations.
3. **Runtime Services** – A gqlgen-powered HTTP server exposes the Relay API with dataloaders, Node ID helpers, and metrics.
4. **Security Layer** – Middleware verifies OIDC tokens, translates claims into application roles, and applies directive guards.
5. **Extension Modules** – Mixins and field types that unlock PostGIS geometries, pgvector embeddings, or TimescaleDB hypertables.

---

## Documentation Map with Highlights

| Guide | Why You Should Read It | Example Spotlight |
|-------|------------------------|-------------------|
| [Overview](./overview.md) | Mental model, architecture, roadmap | Timeline of a request from GraphQL to Postgres |
| [Quickstart](./quickstart.md) | Hands-on creation of a project | Generates `User` and `Post` entities end-to-end |
| [Schema Definition](./schema-definition.md) | DSL reference and advanced patterns | Composable mixins, annotations, privacy policies |
| [GraphQL API](./graphql-api.md) | Relay compliance, Node lookup, connections | Mutation payload walkthrough with optimistic UI hooks |
| [Authentication](./authentication.md) | Configure OIDC, map claims, secure resolvers | Keycloak dev setup + custom claims mapper example |
| [Extensions](./extensions.md) | Enable PostGIS/pgvector/TimescaleDB | Geospatial query + embedding similarity examples |
| [CLI](./cli.md) | Command cheat sheet and automation hooks | Using `erm doctor` (preview) to detect drift |
| [Performance & Observability](./performance-observability.md) | Tuning and diagnostics | pprof integration + dataloader cache tuning |
| [Testing](./testing.md) | Strategy for generated and handwritten code | Table-driven resolver tests with mock loaders |
| [Best Practices](./best-practices.md) | Team conventions and guardrails | Pull request checklist for schema + migration changes |
| [Troubleshooting](./troubleshooting.md) | Rapid fixes for common failures | JWKS fetch failures and migration conflicts |
| [End-to-End Examples](./examples.md) | Complete feature walkthroughs | Project management board + analytics pipeline patterns |

---

## Staying Current

erm is actively developed with a public roadmap. The documentation in this portal mirrors the CLI scaffolding and generator
output at HEAD. When features graduate from experimental flags, the relevant guides are updated with migration notes and
callouts (see the changelog sections within each page). If you are updating an existing project, run `erm gen --dry-run` to
inspect changes and consult the [Testing](./testing.md) and [Troubleshooting](./troubleshooting.md) guides for upgrade
checklists.

---

## Contributing to the Documentation

- **Propose Edits:** Open a pull request that updates the relevant Markdown files under `docs/`. Each page is scoped to a
  particular area; keep changes focused and include before/after CLI or schema snippets when applicable.
- **Automate with AI:** When using AI assistants, point them to the appropriate sections (for example, "Schema DSL: Field
  Modifiers" in `schema-definition.md`) to keep generated code consistent with conventions.
- **Report Gaps:** If a workflow is missing coverage, file an issue tagged `area/docs` so we can prioritize additional guides or
  examples.

Thanks for building with erm. Happy shipping!
