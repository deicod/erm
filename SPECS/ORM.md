# Spec — ORM (schema-as-code)

## Goals
- ent-like developer model with code generation from Go schema definitions.
- Strict Postgres via pgx/v5.
- UUID v7 for entity IDs by default.

## Concepts

### Entity Schema (Go)
Each entity defines:
- **Fields**: name, type, nullability, default, annotations (index, unique, vector, geometry, timeseries).
- **Edges**: `O2O`, `O2M`, `M2M`, with inverse names and join options.
- **Indexes**: single or composite; expression indexes; partials.
- **Views**: read-only projections with custom SQL.
- **Mixins**: common fields (timestamps, soft-delete).
- **Annotations**: arbitrary labels for codegen and GraphQL.

```go
// examples/blog/schema/User.schema.go
package schema

import "github.com/deicod/erm/internal/orm/dsl"

type User struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.String("email").Unique().NotEmpty(),
        dsl.String("name").Optional(),
        dsl.Time("created_at").DefaultNow(),
        dsl.Time("updated_at").UpdateNow(),
    }
}

func (User) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToMany("posts", "Post").Ref("author_id"),
    }
}

func (User) Indexes() []dsl.Index {
    return []dsl.Index{
        dsl.Idx("idx_user_email").On("email").Unique(),
    }
}
```

### Codegen
- Builders: `UserCreate`, `UserQuery`, `UserUpdate`, `UserDelete`.
- CRUD with context; transactions with `Tx` wrapper; eager loading via `WithPosts()` etc.
- Predicates: `user.EmailEQ`, `user.NameContains`. 
- Pagination: `user.Paginate(ctx, after, first, before, last)` returns Relay connection.
- Aggregations: `Count`, `Max`, `Min`, `Avg`, `Sum` with filters.
- Hooks/Interceptors/Policies pluggable via generated registries.

### Migrations
- Versioned SQL files placed in `/migrations`.
- Extensions: `CREATE EXTENSION IF NOT EXISTS postgis;` etc.
- Hypertables: helpers to create TimescaleDB hypertables for time-series entities.

### Postgres Extensions (first-class)
- **PostGIS**: field types `Geometry`, `Geography`, helpers for SRID, WKB/WKT.
- **pgvector**: `Vector(dim int)` field; index `ivfflat`; cosine/L2/IP distance.
- **TimescaleDB**: `TimeSeries()` annotation generates hypertable migration + policies.

### Composite Keys & Advanced Constraints
- Support composite unique/indexes; optional composite primary keys (opt-in).

### UUID v7
- App-generated via `github.com/google/uuid` (v7). Stored as `uuid` in Postgres.
