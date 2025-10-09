# Schema Definition Guide

The erm framework uses a schema-as-code approach similar to Facebook's ent or Go's GORM. Schemas are defined in Go files with a specific DSL (Domain Specific Language) that generates all the corresponding database models, GraphQL types, and operations.

## Basic Schema Structure

Each schema is defined as a Go struct that embeds the `dsl.Schema` type:

```go
package schema

import "github.com/deicod/erm/internal/orm/dsl"

type User struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.String("name").Size(255),
        dsl.Time("created_at").DefaultNow(),
        dsl.Time("updated_at").UpdateNow(),
    }
}

func (User) Edges() []dsl.Edge { 
    return nil 
}

func (User) Indexes() []dsl.Index { 
    return nil 
}
```

## Field Types

The framework supports various field types for your schema:

### Primitive Types
- `dsl.String(name)` - Variable character field
- `dsl.Int(name)` - Integer field
- `dsl.Int64(name)` - 64-bit integer field
- `dsl.Float(name)` - Floating point field
- `dsl.Bool(name)` - Boolean field
- `dsl.Time(name)` - Timestamp field
- `dsl.Bytes(name)` - Binary data field

### Special Types
- `dsl.UUIDv7(name)` - UUID v7 identifier (default for primary keys)
- `dsl.JSON(name)` - JSON data field
- `dsl.UUID(name)` - Generic UUID field

### Field Constraints and Modifiers

Fields can have multiple constraints and modifiers:

```go
dsl.String("email").Size(255).Unique().Required()
dsl.String("name").Size(100).Required()
dsl.Time("created_at").DefaultNow()
dsl.Time("updated_at").UpdateNow()
dsl.Bool("is_active").Default(true)
```

Common modifiers include:
- `.Primary()` - Mark as primary key (typically used with UUIDv7)
- `.Unique()` - Enforce uniqueness constraint
- `.Required()` - Make field non-nullable
- `.Size(n)` - Set maximum size for string fields
- `.Default(value)` - Set default value
- `.DefaultNow()` - Set default to current timestamp
- `.UpdateNow()` - Update to current timestamp on modification
- `.Comment(text)` - Add database comment

## Relationships (Edges)

Schemas can define relationships between entities using edges:

```go
type User struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
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

type Post struct{ dsl.Schema }

func (Post) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.UUIDv7("author_id"), // Foreign key column
        dsl.String("title").NotEmpty(),
        dsl.String("body").Optional(),
        dsl.Time("created_at").DefaultNow(),
        dsl.Time("updated_at").UpdateNow(),
    }
}

func (Post) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("author", "User").Field("author_id").Inverse("posts"),
    }
}
```

### Edge Types

- `dsl.ToOne(name, target)` - Defines a required/optional foreign-key to another entity
- `dsl.ToMany(name, target)` - Defines a collection of targets keyed by a foreign column
- `dsl.ManyToMany(name, target)` - Defines many-to-many relationships via a join table (auto-generated unless overridden with `.ThroughTable()`)

The generator produces helpers such as `LoadAuthor`, `LoadPosts`, and `LoadGroups` so that you can batch eager-load relationships and populate each model's `Edges` struct. Callers can set edges manually with the generated `Set<Name>` helpers or check if an edge has been loaded with `EdgeLoaded("name")`.

## Indexes

Database indexes can be defined for performance optimization:

```go
func (User) Indexes() []dsl.Index {
    return []dsl.Index{
        dsl.Index().Fields("email"),                    // Single field index
        dsl.Index().Fields("first_name", "last_name"),  // Composite index
        dsl.Index().Fields("created_at").Desc(),        // Descending index
        dsl.Index().Unique().Fields("email"),           // Unique index
    }
}
```

## Views

Views provide read-only access patterns:

```go
func (User) Views() []dsl.View {
    return []dsl.View{
        dsl.View("active_users").Query("SELECT * FROM users WHERE is_active = true"),
    }
}
```

## Mixins

Mixins allow sharing common fields across multiple schemas:

```go
type AuditMixin struct{ dsl.Mixin }

func (AuditMixin) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.Time("created_at").DefaultNow(),
        dsl.Time("updated_at").UpdateNow(),
    }
}

type User struct{ dsl.Schema }

func (User) Mixins() []dsl.Mixin {
    return []dsl.Mixin{
        AuditMixin{},  // Include audit fields
    }
}

func (User) Fields() []dsl.Field {
    // Additional fields specific to User
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.String("name").Size(255),
    }
}
```

## Annotations

Annotations provide metadata and configuration for code generation:

```go
func (User) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.Annotation{"graphql": map[string]interface{}{
            "description": "A user in the system",
            "exclude_from": []string{"mutation"},
        }},
    }
}
```

## Interceptors and Hooks

For advanced use cases, you can define hooks that execute during operations:

```go
func (User) Hooks() []dsl.Hook {
    return []dsl.Hook{
        dsl.Hook{
            Type: "before_create",
            Code: `// Custom validation logic before creation`,
        },
    }
}

func (User) Interceptors() []dsl.Interceptor {
    return []dsl.Interceptor{
        dsl.Interceptor{
            Type: "mutation",
            Code: `// Custom business logic for mutations`,
        },
    }
}
```

## Privacy Policies

Privacy policies control data access:

```go
func (User) Privacy() dsl.Privacy {
    return dsl.Privacy{
        // Define who can read/write this entity
        Read:   "user.id == context.user_id || context.is_admin",
        Write:  "user.id == context.user_id",
        Create: "context.is_authenticated",
        Delete: "context.is_admin",
    }
}
```

## Generating Code

After defining your schema, run:

```bash
erm gen
```

This will generate:
- Go models with full CRUD operations
- Database migration files
- GraphQL types and resolvers
- Relay-compliant connections and pagination
- Validation and business logic