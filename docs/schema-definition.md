# Schema Definition Guide

erm uses a declarative-but-expressive schema DSL inspired by Ent. You write Go code that describes your domain model, and the
generators synthesize ORM packages, GraphQL types, privacy policies, and migrations. This guide is the canonical reference for
every DSL construct with examples you can copy into `internal/orm/schema`.

> **Terminology:** Throughout this guide, the term *schema* refers to a Go type that embeds `dsl.Schema`. *Entity* describes the
> generated runtime type and database table. The DSL lives in the `github.com/erm-project/erm/internal/orm/dsl` package.

---

## Anatomy of a Schema

Every schema file should follow the same structure: imports, type definition, and optional method overrides.

```go
package schema

import (
    "github.com/erm-project/erm/internal/orm/dsl"
)

type User struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.String("email").NotEmpty().Unique().Comment("Primary login identifier."),
        dsl.String("display_name").Optional().Size(120),
        dsl.Bool("is_admin").Default(false),
        dsl.Time("created_at").DefaultNow(),
        dsl.Time("updated_at").UpdateNow(),
    }
}

func (User) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToMany("posts", "Post").Ref("author").Comment("All posts authored by this user."),
    }
}

func (User) Indexes() []dsl.Index {
    return []dsl.Index{
        dsl.Index().Fields("email").Unique(),
        dsl.Index().Fields("display_name").Where("display_name <> ''"),
    }
}

func (User) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.GraphQL("User").Description("A registered account in the workspace."),
    }
}
```

Key methods you can implement:

| Method | Purpose |
|--------|---------|
| `Fields() []dsl.Field` | Define columns, constraints, defaults, and modifiers. |
| `Edges() []dsl.Edge` | Model relationships between entities and configure inverse edges. |
| `Indexes() []dsl.Index` | Add single or multi-column indexes with ordering and predicates. |
| `Mixins() []dsl.Mixin` | Embed reusable field/edge definitions. |
| `Annotations() []dsl.Annotation` | Provide metadata consumed by generators (GraphQL, privacy, observability). |
| `Hooks() []dsl.Hook` | Register lifecycle hooks that run before/after mutations. |
| `Interceptors() []dsl.Interceptor` | Intercept queries/mutations for cross-cutting concerns. |
| `Policy() dsl.Policy` | Define coarse privacy rules that apply to CRUD operations. |
| `Privacy() dsl.Privacy` | Specify fine-grained allow/deny expressions. |
| `Mutations() []dsl.Mutation` | Attach custom mutation templates to extend the GraphQL API. |

You can override only the methods you need—defaults handle the rest.

---

## Field Types and Modifiers

Fields describe table columns and GraphQL scalar exposure. Each field factory returns a fluent builder that supports modifiers,
validators, and annotations.

### Primitive Fields

| Constructor | Description | Notes |
|-------------|-------------|-------|
| `dsl.String(name)` | Variable length text (defaults to `TEXT`). | Use `.Size(n)` to constrain to `VARCHAR(n)`. |
| `dsl.Int(name)` | 32-bit integer. | Use `.Min()`, `.Max()`, `.Positive()`. |
| `dsl.Int64(name)` | 64-bit integer. | |
| `dsl.Float(name)` | Float64. | `.Precision()` controls decimal scale for SQL. |
| `dsl.Bool(name)` | Boolean flag. | `.Default(false)` recommended. |
| `dsl.Time(name)` | Timestamp with timezone. | `.DefaultNow()` and `.UpdateNow()` integrate with Go `time.Time`. |
| `dsl.Bytes(name)` | Byte slice stored as `BYTEA`. | Useful for binary blobs (e.g., signed documents). |

### Special Fields

| Constructor | Description |
|-------------|-------------|
| `dsl.UUIDv7(name)` | UUID version 7. Default primary key for entities. |
| `dsl.UUID(name)` | Generic UUID (v4 or custom). |
| `dsl.JSON(name)` | JSONB column with optional schema validation. |
| `dsl.Enum(name, values...)` | Enum backed by `TEXT CHECK`. Generates Go constants and GraphQL enum. |
| `dsl.Vector(name, dim)` | pgvector embedding column (requires extension). |
| `dsl.Geometry(name, srid)` | PostGIS geometry field. |
| `dsl.Geography(name, srid)` | PostGIS geography field. |
| `dsl.Decimal(name, precision, scale)` | Numeric column for money/precise values. |

### Common Modifiers

Modifiers can be chained:

```go
dsl.String("username").NotEmpty().Unique().Default("anonymous")
dsl.Time("deleted_at").Optional().Nillable().Comment("Null when active.")
dsl.Float("rating").Range(0, 5).Default(0)
```

Selected modifiers:

- `.Primary()` – Marks field as primary key. Automatically implies `.Immutable()`.
- `.Unique()` – Generates `UNIQUE` constraint and GraphQL `nodeBy<Field>` query.
- `.NotEmpty()` / `.Required()` – Non-null constraint with generated validation.
- `.Optional()` – Field may be omitted on create and update; GraphQL input marks it optional.
- `.Nillable()` – Distinguishes between “not set” and `null` in updates; generates pointer fields in Go builders.
- `.Default(value)` – Static default inserted in Go and SQL.
- `.DefaultFunc(fn)` – Call Go function to set default at runtime (`uuid.New` etc.).
- `.Immutable()` – Prevents updates after initial creation.
- `.Sensitive()` – Excludes from GraphQL outputs and JSON logs (still stored in DB).
- `.Comment(text)` – Adds database comment and docstring for GraphQL schema.

### Validators and Hooks

Attach validation logic directly to fields using `.Validate(func(value T) error)` or the shorthand `.Min()`, `.Max()`, `.Match()`.
Validation runs before hooks and before hitting the database. For cross-field validation, use entity-level hooks instead.

Example:

```go
dsl.String("password").Sensitive().Validate(func(p string) error {
    if len(p) < 14 {
        return errors.New("password must be at least 14 characters")
    }
    return nil
})
```

---

## Relationships (Edges)

Edges define how entities relate. Every edge can declare direction, inverse edges, foreign key fields, and cascade behavior.

### To-One

```go
dsl.ToOne("author", "User").
    Field("author_id").           // optional; defaults to `<edge>_id`
    Unique().                      // ensures 1:1 relationship
    Required().                    // disallow null author_id
    Comment("User that wrote the post")
```

### To-Many

```go
dsl.ToMany("comments", "Comment").
    Ref("post").                  // matches Comment edge
    BatchSize(100).                // dataloader batch size override
    Comment("All comments on this post")
```

### Many-to-Many

```go
dsl.ManyToMany("members", "User").
    ThroughTable("workspace_members").
    ThroughColumns("workspace_id", "user_id").
    Comment("Users belonging to this workspace")
```

When you omit `ThroughTable`, erm generates a join table name using both entity names (e.g., `users_projects`).

### Edge Annotations

- `.OnDeleteCascade()` / `.OnDeleteSetNull()` – Control FK behavior in SQL and GraphQL.
- `.Inverse(name)` – Create an inverse edge without writing a second schema definition. `Ref()` is preferred when referencing a
  concrete field but `.Inverse()` is helpful for convenience edges.
- `.StorageKey(name)` – Override join table or foreign key column names explicitly.
- `.Privacy(expression)` – Apply edge-specific guard in addition to entity policy.

### Generated Helpers

For each edge, the ORM emits:

- `Set<Relation>(ctx context.Context, related *Entity)` – Associate relationships in builders.
- `Add<Relation>(ctx context.Context, related ...*Entity)` – Append to to-many edges.
- `Load<Relation>(ctx context.Context, entity *Entity) error` – Load edges after fetching nodes.
- `EdgeLoaded("relation") bool` – Check if an edge has been populated to avoid duplicate queries.

In GraphQL resolvers, dataloaders automatically batch these loads based on `Edge` definitions.

---

## Indexes and Constraints

Indexes accelerate lookups and enforce uniqueness beyond primary keys.

```go
func (User) Indexes() []dsl.Index {
    return []dsl.Index{
        dsl.Index().Fields("email").Unique(),
        dsl.Index().Fields("created_at").Desc(),
        dsl.Index().Fields("workspace_id", "role").Where("deleted_at IS NULL"),
    }
}
```

Features:

- `.Unique()` – Adds uniqueness constraint and GraphQL query.
- `.Desc()` / `.Asc()` – Ordering per column.
- `.Where(expr string)` – Partial index predicate.
- `.StorageKey(name)` – Override index name.
- `.Using(method)` – Choose index method (e.g., `gin` for JSONB or `gist` for PostGIS).

The generator emits SQL in migrations and attaches helper predicates to Go query builders (e.g., `WhereWorkspaceIDEQ`).

---

## Mixins

Mixins encapsulate reusable field/edge definitions and optional hooks.

```go
type AuditMixin struct{ dsl.Mixin }

func (AuditMixin) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.Time("created_at").DefaultNow(),
        dsl.Time("updated_at").UpdateNow(),
    }
}

func (AuditMixin) Hooks() []dsl.Hook {
    return []dsl.Hook{
        dsl.Hook{
            Type: dsl.HookBeforeUpdate,
            Code: `mutation.SetUpdatedAt(time.Now())`,
        },
    }
}
```

Use mixins in schemas:

```go
func (User) Mixins() []dsl.Mixin {
    return []dsl.Mixin{AuditMixin{}, SoftDeleteMixin{}}
}
```

erm ships common mixins (timestamps, soft delete, workspace scoping) under `internal/orm/schema/mixins`. Copy them when you need
customization.

---

## Annotations and Metadata

Annotations drive cross-cutting features. They are typed for clarity.

```go
func (User) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.GraphQL("User").
            Description("A registered account").
            Implements("Node").
            Expose("email", dsl.GraphQLExposeAlways).
            ExcludeFromMutations("delete"),
        dsl.Observability().
            TraceName("user").
            LogFields("id", "email"),
    }
}
```

Common annotations:

- `dsl.GraphQL(name)` – Override type name, descriptions, expose/hide fields, configure custom payload fragments.
- `dsl.Authz()` – Attach `@auth` directives, role requirements, or viewer capabilities used by GraphQL resolvers.
- `dsl.Observability()` – Emit spans/log fields when the entity is loaded or mutated.
- `dsl.Extension(name)` – Enable Postgres extensions automatically in migrations (`vector`, `postgis`, `timescaledb`).

Annotations cascade into GraphQL schema comments, resolver hints, and CLI diagnostics.

---

## Hooks and Interceptors

Hooks run around mutations (create, update, delete) while interceptors wrap both queries and mutations.

```go
func (User) Hooks() []dsl.Hook {
    return []dsl.Hook{
        dsl.Hook{
            Type: dsl.HookBeforeCreate,
            Code: `if !strings.Contains(strings.ToLower(mutation.Email), "@") {
    return nil, errors.New("email must contain @")
}
return next(ctx, mutation)`
        },
    }
}
```

Interceptors share a similar shape but can target queries:

```go
func (User) Interceptors() []dsl.Interceptor {
    return []dsl.Interceptor{
        dsl.Interceptor{
            Type: dsl.InterceptorQuery,
            Code: `span := tracer.Start(ctx, "orm.user.query")
result, err := next(span.Context(), query)
span.End(err)
return result, err`,
        },
    }
}
```

Generated code places hooks/interceptors in dedicated files so you can extend them without risking merge conflicts.

---

## Privacy and Policies

Privacy rules ensure only authorized viewers can read or mutate data. There are two layers:

### Policy (Coarse)

```go
func (User) Policy() dsl.Policy {
    return dsl.Policy{
        Query:  dsl.AllowIf("viewer.is_admin"),
        Create: dsl.AllowIf("viewer.is_admin"),
        Update: dsl.AllowIf("viewer.id == node.id || viewer.is_admin"),
        Delete: dsl.Deny(),
    }
}
```

### Privacy Expressions (Fine-Grained)

```go
func (User) Privacy() dsl.Privacy {
    return dsl.Privacy{
        Read: []dsl.PrivacyRule{
            dsl.AllowIf("viewer.id == node.id"),
            dsl.AllowIf("viewer.has_role('support')"),
            dsl.Deny(),
        },
        Update: []dsl.PrivacyRule{
            dsl.AllowIf("viewer.id == node.id"),
            dsl.AllowIf("viewer.is_admin"),
            dsl.Deny(),
        },
    }
}
```

Expressions use a simple language with comparison operators, logical AND/OR, and helper functions defined in the privacy engine.
The GraphQL layer injects viewer context so resolvers fail fast before hitting the database.

---

## Custom Mutations and Actions

Extend GraphQL beyond CRUD using mutations defined in schemas.

```go
func (User) Mutations() []dsl.Mutation {
    return []dsl.Mutation{
        dsl.Mutation{
            Name:        "deactivateUser",
            InputFields: []dsl.MutationField{dsl.UUID("id")},
            OutputFields: []dsl.MutationField{
                dsl.Boolean("success"),
            },
            Code: `user, err := client.User.UpdateOneID(input.ID).SetIsActive(false).Save(ctx)
if err != nil {
    return nil, err
}
return &DeactivateUserPayload{Success: true, User: user}, nil`,
        },
    }
}
```

Custom mutations are generated into resolver stubs so you can add logic without editing generated files. The CLI ensures payload
structs live alongside entity packages.

---

## Views and Read Models

Views provide read-only projections backed by SQL queries.

```go
func (User) Views() []dsl.View {
    return []dsl.View{
        dsl.View("active_users").
            Materialized().
            Query(`SELECT * FROM users WHERE deleted_at IS NULL`).
            RefreshOn("users"),
    }
}
```

Materialized views generate `CREATE MATERIALIZED VIEW` migrations and optional refresh triggers.

---

## Generated File Layout

After running `erm gen`, inspect the following structure for each entity:

```
internal/orm/user/
├── user.go               # Core entity definition and getters
├── user_create.go        # Builder for create mutations
├── user_update.go        # Builder for updates
├── user_delete.go        # Delete helper
├── user_query.go         # Query builder with predicates and eager-loading
├── user_policy.go        # Generated privacy enforcement
├── user_hooks.go         # Hook registration (you can extend in *_extension.go)
├── user_annotations.go   # Annotation metadata for GraphQL and observability
├── user_edge.go          # Edge definitions and loader helpers
└── (more files)
```

Do not edit `*_generated.go` files directly; instead, create `_extension.go` or modify the schema DSL.

---

## Migration Output

Each schema change generates SQL migrations with comments pointing back to the schema file.

```sql
-- internal/orm/schema/user.go: Field "email"
ALTER TABLE users ADD COLUMN email TEXT NOT NULL;
COMMENT ON COLUMN users.email IS 'Primary login identifier.';

-- internal/orm/schema/user.go: Index idx_users_email
CREATE UNIQUE INDEX idx_users_email ON users (email);
```

You can rename migration files using `--name` flag on `erm gen` for clarity:

```bash
erm gen --name add-user-auth-fields
```

---

## Schema Authoring Tips

- Keep schema files focused—one entity per file with helper types adjacent.
- Use mixins for shared fields to avoid copy-paste drift.
- Prefer annotations over manual resolver edits; generators ingest annotations to customize GraphQL output.
- Always run `erm gen` after modifying schemas, then review migrations before applying them.
- Update documentation (this portal) and tests when you introduce new patterns or business rules.

Armed with this reference, continue to [graphql-api.md](./graphql-api.md) to see how schemas map to the Relay layer.
