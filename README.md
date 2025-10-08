# erm — GraphQL + ORM codegen for Go (Relay, pgx v5, OIDC)

> **Status:** design + bootstrap skeleton (alpha).  
> **Goal:** Generate an opinionated, production-grade GraphQL backend that fully supports the Relay Server Spec (global `Node` IDs + cursor-based connections) on top of a schema-as-code ORM with code generation—built for PostgreSQL via **`jackc/pgx` v5** and secured by **OIDC** (default mapping for **Keycloak**).

- **DX & AI-friendly:** one-command scaffolds, rich comments, consistent conventions, and LLM-ready docs & prompts.
- **Relay-complete:** `Node` interface, global ObjectID, connections/edges, `PageInfo`, opaque cursors.
- **ORM:** ent-like schema-as-code: fields, edges, indexes, views, mixins, annotations; hooks, interceptors, privacy policies, transactions, predicates, eager loading, traversal, pagination, ordering, aggregations, and migrations.
- **PostgreSQL first-class:** batteries-included support for extensions: **PostGIS**, **pgvector**, **TimescaleDB**.
- **IDs:** **UUID v7** by default at the ORM layer (generated app-side).
- **CLI:** `erm` provides `init`, `new`, `gen`, and `graphql init` workflows.

See: [PRD](PRD.md), [Roadmap](ROADMAP.md), [Agents](AGENTS.md), [Orchestrator](ORCHESTRATOR.md).

---

## Quickstart (local)

```bash
mkdir myproj && cd myproj
go mod init github.com/yourname/myproj
go mod tidy
# Assuming 'erm' binary is on PATH, else: go run ./cmd/erm
erm init
erm new User
erm gen
erm graphql init
```

Generated app uses:
- **gqlgen** for GraphQL (autobinds to generated ORM types).
- **OIDC** middleware (pluggable claims mapping; default Keycloak).
- **pgx/v5** pools + migrations.

## License

MIT © 2025 deicod / contributors
