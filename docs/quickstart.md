# Quickstart Guide

This guide will help you get started with erm by creating a new project and generating your first GraphQL API.

## Prerequisites

- Go 1.22+ installed
- PostgreSQL server running (for full functionality)

## Installation

First, install the erm CLI tool:

```bash
go install github.com/deicod/erm/cmd/erm@latest
```

## Creating Your First Project

### 1. Initialize a New Project

```bash
mkdir myproject && cd myproject
go mod init github.com/yourname/myproject
go mod tidy
erm init
```

The `erm init` command will create the basic project structure with configuration files and a default schema.

### 2. Define Your First Schema

Create a new schema definition using the `erm new` command:

```bash
erm new User
```

This creates a schema file in the `schema/` directory. You can edit this file to add fields, edges, and other schema elements:

```go
package schema

import "github.com/deicod/erm/internal/orm/dsl"

type User struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.String("name").Optional(),
        dsl.String("email").Unique().NotEmpty(),
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
        dsl.Idx("idx_users_email").On("email").Unique(),
    }
}
```

### 3. Generate Code

Generate the ORM models, GraphQL resolvers, and other code:

```bash
erm gen
```

This command scans your schema files and generates all necessary backend code including:
- ORM models with full CRUD operations
- Relationship helpers that populate `Edges` fields and mark which associations are loaded
- Database migration files
- GraphQL types and resolvers
- Relay-compliant connections and pagination

### 4. Initialize GraphQL

Set up the GraphQL server:

```bash
erm graphql init
```

This creates the necessary GraphQL configuration and server setup files.

## Running Your Application

After code generation, you can build and run your application:

```bash
go mod tidy  # Ensure all dependencies are resolved
go run main.go
```

Your GraphQL API will be available at the configured endpoint (typically `http://localhost:8080/graphql`) where you can access GraphQL Playground to explore and test your API.

## Connecting to PostgreSQL

Your generated application will connect to PostgreSQL using the configuration in `erm.yaml`. Update this file with your database connection details:

```yaml
database:
  host: localhost
  port: 5432
  user: postgres
  password: yourpassword
  name: myproject
  ssl_mode: disable
```

## Exploring Generated Features

Your generated application includes:

- **Relay-compliant API**: Global Node IDs, connections with cursors, PageInfo
- **CRUD Operations**: Create, read, update, delete operations for all entities
- **Relationship Helpers**: Eager-load related entities with generated `Load<Name>` methods and inspect the populated `Edges` struct on each model
- **Type Safety**: Generated Go types matching your schema definitions
- **Migrations**: Versioned database migration files
- **Authentication**: OIDC middleware with Keycloak integration

### Loading Relationships

After generating the ORM package you can traverse relationships in both directions without hand-written SQL:

```go
client := gen.NewClient(db)

users, err := client.Users().List(ctx, 10, 0)
if err != nil {
    log.Fatal(err)
}

if err := client.Users().LoadPosts(ctx, users...); err != nil {
    log.Fatal(err)
}

for _, u := range users {
    if !u.EdgeLoaded("posts") {
        continue
    }
    for _, post := range u.Edges.Posts {
        fmt.Printf("%s wrote %s\n", u.Email, post.Title)
    }
}
```

The same helpers exist for inverse edges: calling `client.Posts().LoadAuthor(ctx, posts...)` batches the foreign-key lookup and attaches each author to `post.Edges.Author`.

## Next Steps

- [Schema Definition Guide](./schema-definition.md) - Learn how to define complex schemas with relationships
- [GraphQL API Usage](./graphql-api.md) - Explore the generated GraphQL API features
- [OIDC Authentication](./authentication.md) - Configure authentication and authorization
- [Extensions Guide](./extensions.md) - Use PostGIS, pgvector, and TimescaleDB features

## Troubleshooting

If you encounter issues:

1. Verify your Go version meets the requirements (1.22+)
2. Ensure PostgreSQL is running and accessible
3. Check that all dependencies are properly installed with `go mod tidy`
4. Run `erm doctor` for diagnostic information (when available)