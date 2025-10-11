# Quickstart Guide

This hands-on tutorial walks you through bootstrapping a new erm project, defining a schema, generating code, and running the
GraphQL API. Follow along to get a working service in minutes.

---

## Prerequisites

- Go 1.22+
- PostgreSQL 14+
- (Optional) Node.js for GraphQL tooling
- An OIDC provider (Keycloak recommended for local development)

Ensure `go`, `psql`, and `erm` binaries are on your `PATH`.

---

## 1. Initialize the Project

```bash
mkdir blog && cd blog
go mod init github.com/example/blog
go mod tidy
erm init --module github.com/example/blog --oidc-issuer http://localhost:8080/realms/erm
```

`erm init` creates:

- `erm.yaml` with module path, database defaults, and OIDC configuration
- `cmd/server` GraphQL entrypoint
- `internal/orm/schema` with sample schema mixins
- `internal/graphql` resolver scaffolding
- `.env.example` and Makefile targets (`gen`, `test`, `serve`)

---

## 2. Configure the Database

Update `erm.yaml`:

```yaml
database:
  url: postgres://postgres:postgres@localhost:5432/blog?sslmode=disable
```

Create the database:

```bash
createdb blog
```

---

## 3. Define Schemas

Generate entity skeletons:

```bash
erm new User
erm new Post
```

Edit `internal/orm/schema/user.go`:

```go
package schema

import "github.com/erm-project/erm/orm/dsl"

type User struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.String("email").NotEmpty().Unique(),
        dsl.String("display_name").Optional(),
    }
}

func (User) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToMany("posts", "Post").Ref("author"),
    }
}
```

Edit `internal/orm/schema/post.go`:

```go
package schema

import "github.com/erm-project/erm/orm/dsl"

type Post struct{ dsl.Schema }

func (Post) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.String("title").NotEmpty(),
        dsl.String("body").Optional(),
        dsl.TimestampTZ("published_at").Optional().Nillable(),
    }
}

func (Post) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("author", "User").Required().Field("author_id").Comment("Author of the post"),
    }
}
```

> **Next steps:** As your app grows beyond this starter schema, review the
> [complex relationship playbook](./schema-definition.md#modeling-complex-relationship-graphs) and the
> [editorial workspace walkthroughs](../examples/blog/README.md) for multi-tenant join tables, scoped query composition, and
> production-grade validation patterns.

---

## 4. Generate Code

```bash
erm gen
```

During development you can run:

```bash
erm gen --dry-run --diff      # Inspect the SQL and a concise change summary
erm gen --only orm,graphql    # Refresh code without touching migrations
```

Outputs:

- ORM packages in `internal/orm/user` and `internal/orm/post`
- GraphQL schema/resolvers in `internal/graphql`
- SQL migrations under `migrations/`

Review migrations:

```bash
ls migrations
cat migrations/20240101010101_create_users.sql
cat migrations/20240101010102_create_posts.sql
```

Apply migrations:

```bash
psql blog < migrations/20240101010101_create_users.sql
psql blog < migrations/20240101010102_create_posts.sql
```

Or use your migration tool of choice.

---

## 5. Seed Data (Optional)

Create a seed script `internal/seed/seed.go`:

```go
package seed

func Run(ctx context.Context, client *orm.Client) error {
    user, err := client.User.Create().SetEmail("hello@example.com").Save(ctx)
    if err != nil {
        return err
    }
    _, err = client.Post.Create().
        SetTitle("Hello erm").
        SetAuthor(user).
        Save(ctx)
    return err
}
```

Call it from `cmd/server/main.go` in dev mode.

---

## 6. Run the GraphQL Server

```bash
go run ./cmd/server
```

Open [http://localhost:8080/graphql](http://localhost:8080/graphql) (Playground enabled if `--playground` flag used during
`erm graphql init`).

Execute a query:

```graphql
query {
  users(first: 10) {
    edges {
      node {
        id
        email
        posts(first: 5) {
          edges { node { title } }
        }
      }
    }
  }
}
```

Create a post:

```graphql
mutation {
  createPost(input: { title: "GraphQL with erm", authorId: "<UserID>" }) {
    post {
      id
      title
    }
  }
}
```

---

## 7. Enable Subscriptions

Opt in by flipping `graphql.subscriptions.enabled` to `true` in `erm.yaml` and annotating entities with `dsl.GraphQLSubscriptions`. The generator adds `userCreated`, `userUpdated`, and `userDeleted` fields (mirroring the triggers you request) plus resolver stubs that publish after ORM mutations succeed. Swap `handler.NewDefaultServer` for `server.NewServer` so websocket transports are registered automatically, or supply your own `subscriptions.Broker` implementation when you need Redis/Kafka fan-out.

```go
func (User) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.GraphQL("User",
            dsl.GraphQLSubscriptions(dsl.SubscriptionEventCreate, dsl.SubscriptionEventUpdate),
        ),
    }
}
```

Background jobs can deliver events manually via `resolvers.Publish(ctx, broker, "User", resolvers.SubscriptionTriggerUpdated, gqlUser)`—consumers receive hydrated GraphQL objects without polling.

---

## 8. Add Authentication

Set up Keycloak (or your provider) and update `erm.yaml` `oidc` block. Restart the server to pick up configuration. Use
`scripts/get-token.sh` to fetch tokens for Playground.

---

## 9. Run Tests

```bash
go test ./...
```

Use `internal/testing` helpers (sandbox + GraphQL harness) to create targeted tests.

---

## 10. Next Steps

- Explore [schema-definition.md](./schema-definition.md) for advanced DSL patterns.
- Configure tracing and metrics per [performance-observability.md](./performance-observability.md).
- Add more entities and rerun `erm gen` as your domain evolves.

Happy shipping!
