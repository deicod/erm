# Getting Started with the Schema Skeleton

The `erm new <Entity>` command scaffolds a schema file under `schema/<Entity>.schema.go` with a production-ready baseline.
The template mirrors the `User` example shipped in this repository and demonstrates how to combine the DSL, query helpers,
and GraphQL annotations in a single file.【F:internal/cli/new.go†L39-L79】【F:schema/User.schema.go†L1-L44】

```go
package schema

import "github.com/deicod/erm/internal/orm/dsl"

// Entity models the Entity domain entity.
type Entity struct{ dsl.Schema }

func (Entity) Fields() []dsl.Field {
        return []dsl.Field{
                dsl.UUIDv7("id").Primary(),
                dsl.Text("slug").
                        Computed(dsl.Computed(dsl.Expression("id::text"))),
                dsl.TimestampTZ("created_at").DefaultNow(),
                dsl.TimestampTZ("updated_at").UpdateNow(),
        }
}
```

## Default building blocks

| Section | Purpose | What to customize |
| --- | --- | --- |
| `Fields()` | Declares identifiers, lifecycle timestamps, and a computed slug that mirrors the UUID. | Replace or extend with business fields, enforce constraints (e.g. `.NotEmpty()`), or add computed expressions. |
| `Edges()` | Placeholder for relationships. Defaults to `nil` so you can layer in `dsl.ToOne`/`dsl.ToMany` links when other entities exist. | Define ownership, optionality, and cascading behavior once related schemas are scaffolded. |
| `Indexes()` | Entry point for secondary indexes. Starts empty to highlight where to add unique constraints or partial indexes. | Add `dsl.Idx(...).On(...)` declarations for frequently queried columns. |
| `Query()` | Bundles reusable predicates, default ordering, and aggregate helpers for generated repositories and resolvers. | Extend with domain-specific filters (e.g. `SlugEq`, `CreatedAfter`) and adjust `OrderBy` or limits. |
| `Annotations()` | Adds integration metadata—by default GraphQL exposure plus real-time subscription events. | Introduce REST annotations, privacy markers, or disable GraphQL entirely. |

## Iterating with TDD

The schema skeleton is designed to be evolved in tandem with tests. When you change it:

1. Add or update tests under `internal/generator` or your application package to encode the expected behavior.
2. Regenerate artifacts with `erm gen` and review the staged diff.
3. Run the schema workflow commands captured in `schema/AGENTS.md` to keep the feedback loop fast (`go test`, `go test -race`, and `go vet`).【F:schema/AGENTS.md†L5-L11】

## Next steps

- Introduce additional fields that capture your domain and wire up `Edges()` to link related entities.
- Extend the query spec with rich predicates so the generated ORM and GraphQL layers expose ergonomic filters.
- Document the model in your project README to onboard collaborators faster, mirroring the approach used here.
