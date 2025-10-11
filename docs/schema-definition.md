# Schema Definition Guide

erm uses a declarative-but-expressive schema DSL inspired by Ent. You write Go code that describes your domain model, and the
generators synthesize ORM packages, GraphQL types, privacy policies, and migrations. This guide is the canonical reference for
every DSL construct with examples you can copy into `orm/schema`.

> **Terminology:** Throughout this guide, the term *schema* refers to a Go type that embeds `dsl.Schema`. *Entity* describes the
> generated runtime type and database table. The DSL lives in the `github.com/erm-project/erm/orm/dsl` package.

---

## Anatomy of a Schema

Every schema file should follow the same structure: imports, type definition, and optional method overrides.

```go
package schema

import (
    "github.com/erm-project/erm/orm/dsl"
)

type User struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.String("email").NotEmpty().Unique().Comment("Primary login identifier."),
        dsl.String("display_name").Optional().Size(120),
        dsl.Bool("is_admin").Default(false),
        dsl.TimestampTZ("created_at").DefaultNow(),
        dsl.TimestampTZ("updated_at").UpdateNow(),
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

func (User) Query() dsl.QuerySpec {
    return dsl.Query().
        WithPredicates(
            dsl.NewPredicate("id", dsl.OpEqual).Named("IDEq"),
            dsl.NewPredicate("email", dsl.OpILike).Named("EmailILike"),
        ).
        WithOrders(
            dsl.OrderBy("created_at", dsl.SortDesc).Named("CreatedAtDesc"),
        ).
        WithAggregates(
            dsl.CountAggregate("Count"),
        ).
        WithDefaultLimit(25).
        WithMaxLimit(100)
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
| `Query() dsl.QuerySpec` | Describe predicates, sort orders, and aggregates for the fluent query builder. |
| `Mixins() []dsl.Mixin` | Embed reusable field/edge definitions. |
| `Annotations() []dsl.Annotation` | Provide metadata consumed by generators (GraphQL, privacy, observability). |
| `Hooks() []dsl.Hook` | Register lifecycle hooks that run before/after mutations. |
| `Interceptors() []dsl.Interceptor` | Intercept queries/mutations for cross-cutting concerns. |
| `Policy() dsl.Policy` | Define coarse privacy rules that apply to CRUD operations. |
| `Privacy() dsl.Privacy` | Specify fine-grained allow/deny expressions. |
| `Mutations() []dsl.Mutation` | Attach custom mutation templates to extend the GraphQL API. |

You can override only the methods you need—defaults handle the rest.

---

## Modeling Complex Relationship Graphs

Large products frequently require richer relationship graphs than a single `ToOne`/`ToMany` pair. The blog sample under
[`examples/blog`](../examples/blog) now includes a **workspace editorial workflow** that demonstrates how to compose join tables,
scoped predicates, and edge annotations without losing generator ergonomics.

```go
// examples/blog/schema/Workspace.schema.go
func (Workspace) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToMany("memberships", "Membership").Ref("workspace"),
        dsl.ToMany("posts", "Post").Ref("workspace"),
    }
}

// examples/blog/schema/Membership.schema.go
func (Membership) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("workspace", "Workspace").Field("workspace_id"),
        dsl.ToOne("user", "User").Field("user_id"),
    }
}

// examples/blog/schema/Post.schema.go
func (Post) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("author", "User").Field("author_id").Inverse("posts"),
        dsl.ToOne("workspace", "Workspace").Field("workspace_id"),
        dsl.ToMany("comments", "Comment").Ref("post"),
    }
}

// examples/blog/schema/Comment.schema.go
func (Comment) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("post", "Post").Field("post_id"),
        dsl.ToOne("author", "User").Field("author_id"),
        dsl.ToOne("parent", "Comment").Field("parent_id").Optional(),
        dsl.ToMany("replies", "Comment").Ref("parent"),
    }
}
```

Key techniques illustrated by the scenario:

1. **Join tables with explicit models** – `Membership` keeps per-workspace roles and audit timestamps. `dsl.Idx("membership_workspace_user_unique").On("workspace_id", "user_id").Unique()`
   enforces uniqueness so a user cannot have duplicate memberships.
2. **Scoped query composition** – `Post.Query()` constrains default predicates to the active workspace and exposes helper methods like
   `.WhereWorkspaceIDEq()` to keep GraphQL resolvers tenant-aware without manual SQL.
3. **Recursive relationships** – `Comment` demonstrates how self-referencing edges (`parent`/`replies`) combine with optional foreign keys to
   power nested conversations while keeping query counts predictable.

See the [blog walkthroughs](../examples/blog/README.md) for step-by-step validation, profiling, and error-handling flows that exercise
these definitions end-to-end.

---

## Field Types and Modifiers

Fields describe table columns and GraphQL scalar exposure. Each field factory returns a fluent builder that supports modifiers,
validators, and annotations.

### Field Constructors

PostgreSQL column families map to dedicated helpers. Unless you override `GoType` or `Column`, the generators derive SQL, Go, and GraphQL metadata (including custom scalars such as `BigInt`, `Decimal`, `Timestamptz`, and `JSONB`). Legacy shortcuts like `dsl.String`, `dsl.Int`, `dsl.Float`, and `dsl.Bool` remain as aliases for the richer helpers below.

#### Character & UUID

| Constructor | SQL type | Go type | GraphQL scalar |
|-------------|----------|---------|----------------|
| `dsl.Text(name)` | `text` | `string` | `String` |
| `dsl.VarChar(name, size)` | `varchar(size)` (defaults to `varchar`) | `string` | `String` |
| `dsl.Char(name, size)` | `char(size)` | `string` | `String` |
| `dsl.UUID(name)` / `dsl.UUIDv7(name)` | `uuid` | `string` | `ID` |

#### Numeric & Boolean

| Constructor | SQL type | Go type | GraphQL scalar |
|-------------|----------|---------|----------------|
| `dsl.Boolean(name)` | `boolean` | `bool` | `Boolean` |
| `dsl.SmallInt(name)` | `smallint` | `int16` | `Int` |
| `dsl.Integer(name)` | `integer` | `int32` | `Int` |
| `dsl.BigInt(name)` | `bigint` | `int64` | `BigInt` |
| `dsl.SmallSerial(name)` | `smallserial` | `int16` | `Int` |
| `dsl.Serial(name)` | `serial` | `int32` | `Int` |
| `dsl.BigSerial(name)` | `bigserial` | `int64` | `BigInt` |
| `dsl.SmallIntIdentity(name, mode)` | `smallint GENERATED … AS IDENTITY` | `int16` | `Int` |
| `dsl.IntegerIdentity(name, mode)` | `integer GENERATED … AS IDENTITY` | `int32` | `Int` |
| `dsl.BigIntIdentity(name, mode)` | `bigint GENERATED … AS IDENTITY` | `int64` | `BigInt` |
| `dsl.Decimal(name, precision, scale)` | `decimal(precision,scale)` | `string` | `Decimal` |
| `dsl.Numeric(name, precision, scale)` | `numeric(precision,scale)` | `string` | `Decimal` |
| `dsl.Real(name)` | `real` | `float32` | `Float` |
| `dsl.DoublePrecision(name)` | `double precision` | `float64` | `Float` |
| `dsl.Money(name)` | `money` | `string` | `Money` |

Use `dsl.IdentityAlways` or `dsl.IdentityByDefault` as the second parameter when you need identity columns. When omitted, identity helpers default to `BY DEFAULT` semantics.

#### Temporal

| Constructor | SQL type | Go type | GraphQL scalar |
|-------------|----------|---------|----------------|
| `dsl.Date(name)` | `date` | `time.Time` | `Date` |
| `dsl.Time(name)` | `time` | `time.Time` | `Time` |
| `dsl.TimeTZ(name)` | `timetz` | `time.Time` | `Timetz` |
| `dsl.Timestamp(name)` | `timestamp` | `time.Time` | `Timestamp` |
| `dsl.TimestampTZ(name)` | `timestamptz` | `time.Time` | `Timestamptz` |
| `dsl.Interval(name)` | `interval` | `string` | `Interval` |

#### Binary, JSON, and XML

| Constructor | SQL type | Go type | GraphQL scalar |
|-------------|----------|---------|----------------|
| `dsl.Bytea(name)` | `bytea` | `[]byte` | `String` |
| `dsl.JSON(name)` | `json` | `json.RawMessage` | `JSON` |
| `dsl.JSONB(name)` | `jsonb` | `json.RawMessage` | `JSONB` |
| `dsl.XML(name)` | `xml` | `string` | `XML` |

#### Network & Bit Strings

| Constructor | SQL type | Go type | GraphQL scalar |
|-------------|----------|---------|----------------|
| `dsl.Inet(name)` | `inet` | `string` | `Inet` |
| `dsl.CIDR(name)` | `cidr` | `string` | `CIDR` |
| `dsl.MACAddr(name)` | `macaddr` | `string` | `MacAddr` |
| `dsl.MACAddr8(name)` | `macaddr8` | `string` | `MacAddr8` |
| `dsl.Bit(name, length)` | `bit(length)` | `string` | `BitString` |
| `dsl.VarBit(name, length)` | `varbit(length)` | `string` | `VarBitString` |

#### Text Search & Range Types

| Constructor | SQL type | Go type | GraphQL scalar |
|-------------|----------|---------|----------------|
| `dsl.TSVector(name)` | `tsvector` | `string` | `TSVector` |
| `dsl.TSQuery(name)` | `tsquery` | `string` | `TSQuery` |
| `dsl.Int4Range(name)` | `int4range` | `string` | `Int4Range` |
| `dsl.Int8Range(name)` | `int8range` | `string` | `Int8Range` |
| `dsl.NumRange(name)` | `numrange` | `string` | `NumRange` |
| `dsl.TSRange(name)` | `tsrange` | `string` | `TSRange` |
| `dsl.TSTZRange(name)` | `tstzrange` | `string` | `TSTZRange` |
| `dsl.DateRange(name)` | `daterange` | `string` | `DateRange` |

#### Arrays & Extensions

- `dsl.Array(name, elementType)` creates PostgreSQL arrays (`elementType[]`). Pass constants like `dsl.TypeText` or `dsl.TypeInteger`; the generators derive the appropriate Go slice (for example `[]string` or `[]int32`) and GraphQL list (`[String!]!`).
- `dsl.Geometry(name)` / `dsl.Geography(name)` require the PostGIS extension. Columns are exposed as `[]byte` in Go and the `JSON` scalar in GraphQL.
- `dsl.Vector(name, dim)` requires the `vector` extension and maps to `[]float32` in Go and `[Float!]!` in GraphQL.
- `dsl.Enum(name, values...)` still generates GraphQL enums backed by a `TEXT CHECK` constraint.

### Common Modifiers

Modifiers can be chained:

```go
dsl.VarChar("username", 64).NotEmpty().Unique().Default("anonymous")
dsl.TimestampTZ("deleted_at").Optional().Comment("Null when active.")
dsl.Decimal("rating", 4, 2).WithDefault("0.00")
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
- `.Length(n)` / `.Precision(n)` / `.Scale(n)` – Override size metadata for `VARCHAR`, numeric, and temporal columns.
- `.Identity(mode)` – Toggle identity generation (`dsl.IdentityAlways` or `dsl.IdentityByDefault`) on supported integer types.

### Computed Columns

Generated (computed) columns let PostgreSQL derive values from other fields while keeping the ORM models in sync. Use `dsl.Expression` to describe the SQL fragment and optional dependency list, then wrap it with `dsl.Computed` on the field:

```go
dsl.Text("display_name").
    Computed(dsl.Computed(dsl.Expression(
        "COALESCE(first_name || ' ' || last_name, email)",
        "first_name", "last_name", "email",
    )))
```

- The generator emits `GENERATED ALWAYS AS (...) STORED` columns and records dependencies in the schema snapshot so diffs are stable.
- Computed fields hydrate automatically from queries, but they are read-only: generated clients and GraphQL mutations reject inputs where a computed field is non-zero.
- Because PostgreSQL cannot alter a generated expression in-place, the migration planner will drop and recreate the column if the definition changes. Plan for a brief lock or add a manual migration when large tables are involved.
- Computed columns cannot declare defaults or be targeted by `.UpdateNow()`. If you need application-level fallbacks, keep a separate writable column instead.

### Validators and Hooks

Attach validation logic directly to fields using `.Validate(func(value T) error)` or the shorthand `.Min()`, `.Max()`, `.Match()`.
Validation runs before hooks and before hitting the database. Cross-field checks and complex predicates can be registered through the runtime validation registry that ships with the ORM generator.

#### Runtime rules

Generated packages export `gen.ValidationRegistry`, an instance of `orm/runtime/validation.Registry`. Register rules during package init (or application startup) to apply additional constraints without editing generated files. Rules run prior to hitting the database for both `Create` and `Update` mutations.

```go
var emailRegex = regexp.MustCompile(`^[^@]+@example.com$`)

func init() {
    gen.ValidationRegistry.Entity("User").
        OnCreate(
            validation.String("Email").Required().Matches(emailRegex).Rule(),
        ).
        OnUpdate(validation.RuleFunc(func(_ context.Context, subject validation.Subject) error {
            created, _ := subject.Record.Time("CreatedAt")
            updated, _ := subject.Record.Time("UpdatedAt")
            if updated.Before(created) {
                return validation.FieldError{Field: "UpdatedAt", Message: "must be after CreatedAt"}
            }
            return nil
        }))
}
```

- `validation.String(field)` supplies fluent helpers for string length and regex checks.
- `validation.RuleFunc` supports custom logic (including cross-field access through `subject.Record`).
- `subject.Input` exposes the raw struct pointer if you prefer typed assertions.

To override rules in tests or sandboxes, reassign `gen.ValidationRegistry = validation.NewRegistry()` before registering replacements.

---

## Relationships (Edges)

Edges define how entities relate. Every edge can declare direction, inverse edges, foreign key fields, and cascade behavior.

### Direction: `To*` vs. inverse edges

Edges are declared from the *source* schema. The `dsl.ToOne`, `dsl.ToMany`, and `dsl.ManyToMany` helpers describe how the
source links to a *target* schema. Use `.Ref("<edge>")` or `.Inverse("<edge>")` on one side to connect the inverse edge so
generators know both ends describe the same relationship.

```go
// user.schema.go
func (User) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToMany("posts", "Post").                      // source: User
            Ref("author").                                 // matches Post's ToOne edge
            Comment("All posts authored by this user."),
    }
}

// post.schema.go
func (Post) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("author", "User").                       // source: Post
            Field("author_id").                            // FK column on posts table
            Required().
            Comment("Account that created the post"),
    }
}
```

`Ref("author")` tells erm that the `User.posts` edge is the inverse of `Post.author`. During codegen this produces join helper
methods, GraphQL field resolvers, and migrates a single `author_id` column on the `posts` table.

### Base edge helpers

- `dsl.ToOne(name, target)` – Foreign key column stored on the source table. Chain `.Field("column")` to override the default
  `<edge>_id` column. Use `.Unique()` to upgrade the relationship to one-to-one.
- `dsl.ToMany(name, target)` – Inverse side of a `ToOne` or `ManyToMany`. Call `.Ref("edge")` to point at the owning
  relationship, or `.Inverse("edge")` for convenience edges without a physical column.
- `dsl.ManyToMany(name, target)` – Declares a join table. Accepts `.ThroughTable()` / `.ThroughColumns()` when you need custom
  names.

### Relationship recipes

The following snippets illustrate common modeling patterns. You can paste them into schemas by replacing the type names and
field identifiers with your domain.

#### One-to-one (two types)

```go
// profile.schema.go
func (Profile) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("user", "User").
            Field("user_id").
            Unique().
            Required(),
    }
}

// user.schema.go
func (User) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("profile", "Profile").
            Ref("user").
            Unique(),
    }
}
```

The `Profile` table owns the `user_id` column. Marking both sides `Unique()` guarantees a true one-to-one link.

#### One-to-one (same type)

```go
func (User) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("manager", "User").
            Field("manager_id").
            Comment("Direct manager for reporting hierarchy"),
    }
}
```

This self-referential relationship stores a `manager_id` column on the `users` table. You can expose the inverse with
`dsl.ToMany("reports", "User").Ref("manager")` when needed.

#### One-to-one (bidirectional convenience)

```go
// workspace.schema.go
func (Workspace) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("billing_account", "BillingAccount").
            Field("billing_account_id").
            Unique().
            Inverse("workspace"),
    }
}

// billingaccount.schema.go
func (BillingAccount) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("workspace", "Workspace").
            Unique(),
    }
}
```

`.Inverse("workspace")` synthesizes a read-only convenience edge on `BillingAccount` without defining a second foreign key.

#### One-to-many (two types)

```go
// post.schema.go
func (Post) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToMany("comments", "Comment").
            Ref("post").
            Comment("Comments left on this post"),
    }
}

// comment.schema.go
func (Comment) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("post", "Post").
            Field("post_id").
            Required(),
    }
}
```

The `Comment.post` edge owns the `post_id` column. Loading `Post.comments` batches automatically thanks to the `.Ref("post")`
link.

#### One-to-many (same type)

```go
func (Task) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("parent", "Task").Field("parent_id"),
        dsl.ToMany("children", "Task").Ref("parent"),
    }
}
```

Parents and children are stored in the same table. erm enforces referential integrity via the generated foreign key.

#### Many-to-many (two types)

```go
// workspace.schema.go
func (Workspace) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ManyToMany("members", "User").
            ThroughTable("workspace_members").
            ThroughColumns("workspace_id", "user_id"),
    }
}

// user.schema.go
func (User) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ManyToMany("workspaces", "Workspace").
            Ref("members"),
    }
}
```

Providing custom join table metadata is optional. Without it, erm generates `users_workspaces` (alphabetical) and creates the
necessary foreign keys.

#### Many-to-many (same type)

```go
func (User) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ManyToMany("followers", "User").
            ThroughTable("user_followers").
            ThroughColumns("user_id", "follower_id"),
    }
}
```

Self-referential many-to-many edges are ideal for social graphs. Use `.Inverse("following")` if you want a convenience edge on
the same schema.

#### Many-to-many (bidirectional convenience)

```go
// project.schema.go
func (Project) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ManyToMany("tags", "Tag").Inverse("projects"),
    }
}

// tag.schema.go
func (Tag) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ManyToMany("projects", "Project"),
    }
}
```

Using `.Inverse("projects")` keeps the `Tag` schema minimal while still generating loaders for both directions.

### Edge annotations

- `.OnDeleteCascade()` / `.OnDeleteSetNull()` / `.OnDeleteRestrict()` – Control FK behavior in SQL and GraphQL. Pair with the matching `.OnUpdate*` helpers to keep constraints symmetric when you expect updates to cascade, restrict, or set null.
- `.Polymorphic(targets...)` – Attach discriminated unions to edges. Combine with `dsl.PolymorphicTarget("<Entity>", "<sql condition>")` to describe how records should be typed at runtime.
- `.Inverse(name)` – Create an inverse edge without writing a second schema definition. `Ref()` is preferred when referencing a concrete field but `.Inverse()` is helpful for convenience edges.
- `.StorageKey(name)` – Override join table or foreign key column names explicitly.
- `.Privacy(expression)` – Apply edge-specific guard in addition to entity policy.

#### Polymorphic unions

```go
func (Post) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("workspace", "Workspace").
            Field("workspace_id").
            OnDeleteCascade().
            Polymorphic(
                dsl.PolymorphicTarget("Workspace", "kind = 'team'"),
                dsl.PolymorphicTarget("Workspace", "kind = 'personal'"),
            ),
    }
}
```

`Polymorphic` metadata flows through the generated registry and clients so resolvers can materialise GraphQL interfaces or unions without hand-written switches.

### Generated helpers

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

## Query Specifications

`Query() dsl.QuerySpec` teaches the ORM about supported filters, sort orders, and aggregates. The generator turns each descriptor
into a fluent helper on the `<Entity>Client`.

```go
func (Post) Query() dsl.QuerySpec {
    return dsl.Query().
        WithPredicates(
            dsl.NewPredicate("id", dsl.OpEqual).Named("IDEq"),
            dsl.NewPredicate("author_id", dsl.OpEqual).Named("AuthorIDEq"),
            dsl.NewPredicate("title", dsl.OpILike).Named("TitleILike"),
        ).
        WithOrders(
            dsl.OrderBy("created_at", dsl.SortDesc).Named("CreatedAtDesc"),
            dsl.OrderBy("title", dsl.SortAsc).Named("TitleAsc"),
        ).
        WithAggregates(
            dsl.CountAggregate("Count"),
        ).
        WithDefaultLimit(20).
        WithMaxLimit(100)
}
```

Generates helpers such as:

```go
posts, err := client.Posts().
    Query().
    WhereAuthorIDEq(userID).
    WhereTitleILike("%launch%").
    OrderByCreatedAtDesc().
    Limit(10).
    All(ctx)

total, err := client.Posts().Query().Count(ctx)
```

Under the hood these descriptors are translated into parametrised SQL by `runtime.BuildSelectSQL` and `runtime.BuildAggregateSQL`,
and executed via the pgx-backed `pg.DB` helpers (`Select`, `Aggregate`).

---

## Mixins

Mixins encapsulate reusable field/edge definitions and optional hooks.

```go
type AuditMixin struct{ dsl.Mixin }

func (AuditMixin) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.TimestampTZ("created_at").DefaultNow(),
        dsl.TimestampTZ("updated_at").UpdateNow(),
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

erm ships common mixins (timestamps, soft delete, workspace scoping) under `orm/schema/mixins`. Copy them when you need
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
orm/user/
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
-- orm/schema/user.go: Field "email"
ALTER TABLE users ADD COLUMN email TEXT NOT NULL;
COMMENT ON COLUMN users.email IS 'Primary login identifier.';

-- orm/schema/user.go: Index idx_users_email
CREATE UNIQUE INDEX idx_users_email ON users (email);
```

You can rename migration files using `--name` flag on `erm gen` for clarity:

```bash
erm gen --name add-user-auth-fields
```

Behind the scenes, the generator records the post-generation schema in `migrations/schema.snapshot.json`. Future runs compare the
current DSL output against this snapshot to derive incremental migrations—only the delta between snapshots is emitted. Use
`erm gen --dry-run` to inspect the computed SQL without touching disk, and pair it with `erm migrate` to apply changes once
you are satisfied.

---

## Schema Authoring Tips

- Keep schema files focused—one entity per file with helper types adjacent.
- Use mixins for shared fields to avoid copy-paste drift.
- Prefer annotations over manual resolver edits; generators ingest annotations to customize GraphQL output.
- Always run `erm gen` after modifying schemas, then review migrations before applying them.
- Update documentation (this portal) and tests when you introduce new patterns or business rules.

Armed with this reference, continue to [graphql-api.md](./graphql-api.md) to see how schemas map to the Relay layer.
