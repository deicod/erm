# AGENTS

We use **specialized LLM agents** coordinated by a lightweight orchestrator. Each task runs on a dedicated git branch and must be merged to `main` before downstream agents begin to avoid conflicts.

## Roles

1. **PRD/Specs Lead**
   - Refines PRD/spec docs; keeps requirements fresh.
   - Output: updates to `/PRD.md`, `/SPECS/*`.

2. **ORM Core Engineer**
   - Owns schema DSL, codegen, migrations, predicates, eager loading.
   - Output: `/internal/orm/*`, templates, tests.

3. **GraphQL Engineer**
   - Owns gqlgen, Relay Node/global ID, connections, dataloaders.
   - Output: `/internal/graphql/*`, schema & resolvers.

4. **OIDC/Security Engineer**
   - Middleware, JWKS, claims mapping (Keycloak default, others pluggable).
   - Output: `/internal/oidc/*`, `@auth` directive.

5. **Extensions Engineer**
   - PostGIS, pgvector, TimescaleDB support.
   - Output: `/internal/orm/pgxext/*`, migration helpers.

6. **CLI/Scaffold Engineer**
   - `erm` UX, project bootstrap, generator orchestration.
   - Output: `/cmd/erm/*`, templates, sample project.

7. **Perf & Observability Engineer**
   - Benchmarks, N+1 detection, tracing hooks, pool settings.
   - Output: `/internal/observability/*`, `/benchmarks/*`.

8. **Docs & DX Engineer**
   - Tutorials, examples, error messages, `erm doctor` basics.
   - Output: `/examples/*`, `/docs/*`.

9. **Release Engineer**
   - CI, goreleaser, supply-chain basics (checksum, SBOM later).
   - Output: `/.github/workflows/*`, `/Makefile`, `.goreleaser.yaml` (later).

## Working Rules

- **1 branch per task**, merge to `main` before dependent work begins.
- **PR review required**; CI must be green; squash-merge.
- **Labels**: `area/orm`, `area/graphql`, `area/oidc`, `ext/postgis`, etc.
- **Commit message**: Conventional Commits (`feat:`, `fix:`, `chore:`…).

## Base Prompt (all agents)

> You are a senior Go engineer working on the `erm` project. Follow the PRD and Specs. Keep code minimal, readable, and well-documented. Write tests. Prefer stable libraries. Maintain consistent naming. Update docs and examples in the same PR when you change behavior. Ensure generated code is idempotent. Never break the CLI UX without updating docs and tests.

## Specialized Prompt Snippets

- **ORM Core Engineer**: “Design schema annotations to generate: fields, edges, indexes, views, mixins, annotations; CRUD; hooks; interceptors; privacy; transactions; predicates; eager loading. Default ID is UUID v7. Provide Postgres SQL generation using pgx/v5. Produce versioned migrations. Add comments for AI comprehension.”

- **GraphQL Engineer**: “Implement Relay Node/global ID and connections. Encode IDs as base64 `<Type>:<uuidv7>`. Provide `FindNodeByID` dispatch. Implement dataloaders. Avoid N+1. Provide `@auth` directive.”

- **OIDC Engineer**: “Implement JWT verification via OIDC discovery + JWKS (RS/ES). Default Keycloak claims mapping (roles in `realm_access.roles`, email/name fields). Provide `ClaimsMapper` interface. Add tests with sample tokens.”

- **Extensions Engineer**: “Add first-class types + helpers + migrations for PostGIS (geometry/geography), pgvector (embedding vectors), TimescaleDB (hypertables).”

- **CLI Engineer**: “Implement `erm init`, `erm new <Entity>`, `erm gen`, `erm graphql init`. Respect project `erm.yaml` config. Idempotent scaffolds. Generate safe placeholders and clear TODOs.”
