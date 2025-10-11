# GraphQL API

erm generates a complete, Relay-compliant GraphQL server using gqlgen. This document explains the schema topology, resolver
implementation patterns, dataloader integration, and how to extend the generated API with custom operations.

---

## Relay Compliance at a Glance

- **Global Node IDs** – Every entity implements the `Node` interface. IDs are encoded as `base64("<Type>:<uuidv7>")`. Helpers
  live in `graphql/node` and the ORM automatically decodes/encodes IDs in builders.
- **Connections & Edges** – Pagination follows the Relay spec using `first`, `last`, `after`, and `before`. Connections expose
  `edges`, `pageInfo`, and `totalCount`.
- **Mutations** – Generated CRUD mutations use input objects and payload objects that include `clientMutationId` for optimistic
  UI patterns.
- **Viewer Context** – Resolvers receive viewer context (from OIDC middleware) and pass it through to privacy rules before
  hitting the database.

---

## Generated Schema Structure

For each entity, erm produces GraphQL types in `graphql/schema.graphqls` (or split across modules if you enable module
mode). Using a `User` entity as an example:

```graphql
type User implements Node {
  id: ID!
  createdAt: Timestamptz!
  updatedAt: Timestamptz!
}

type UserEdge {
  cursor: String!
  node: User
}

type UserConnection {
  edges: [UserEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

input CreateUserInput {
  clientMutationId: String
  id: ID
  createdAt: Timestamptz
  updatedAt: Timestamptz
}

type CreateUserPayload {
  clientMutationId: String
  user: User
}

input UpdateUserInput {
  clientMutationId: String
  id: ID!
  createdAt: Timestamptz
  updatedAt: Timestamptz
}

type UpdateUserPayload {
  clientMutationId: String
  user: User
}

input DeleteUserInput {
  clientMutationId: String
  id: ID!
}

type DeleteUserPayload {
  clientMutationId: String
  deletedUserID: ID!
}
```

Filtering inputs mirror the ORM predicate helpers. The generator converts field modifiers (e.g., `.Unique()`, `.Enum()`) into
GraphQL capabilities automatically.

---

## Root Operations

### Node Lookup

```graphql
query NodeQuery($id: ID!) {
  node(id: $id) {
    id
    ... on User {
      createdAt
      updatedAt
    }
  }
}
```

The `node` resolver dispatches to `FindNodeByID` generated in `graphql/node/registry.go`. You can register additional
node types (e.g., view projections) via annotations.

### Entity Queries

```graphql
query Users($first: Int!, $after: String) {
  users(first: $first, after: $after) {
    edges {
      cursor
      node {
        id
        createdAt
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
    totalCount
  }
}
```

The generated resolver constructs an ORM query with the appropriate filters, orderings, and pagination parameters. Connection
cursors are opaque (base64 encoded) and derived from primary key ordering to ensure deterministic pagination.

### Unique Accessors

For fields marked `.Unique()`, the generator exposes accessors such as `workspaceBySlug` or `teamByHandle`. Example (assuming `Workspace.slug` is marked `Unique()`):

```graphql
query WorkspaceBySlug($slug: String!) {
  workspaceBySlug(slug: $slug) {
    id
    slug
  }
}
```

### Mutations

Every entity gets `create<Entity>`, `update<Entity>`, and `delete<Entity>` mutations. They follow the Relay payload pattern:

```graphql
mutation CreateUser($input: CreateUserInput!) {
  createUser(input: $input) {
    user {
      id
      createdAt
    }
    clientMutationId
  }
}
```

`clientMutationId` is echoed back automatically to support optimistic updates. Partial updates use `Update<Entity>Input` where
optional fields map to pointer types in Go, allowing you to distinguish between "null" and "not provided". The resolver stubs call directly into the ORM client so you inherit hooks, interceptors, and transactions without extra wiring.

### Subscriptions (Optional)

If you enable subscriptions in `erm.yaml`, the generator adds schema fields and resolver scaffolding for each entity annotated with `dsl.GraphQLSubscriptions`. Declare the triggers you care about (create/update/delete) and erm wires typed channels to your broker:

```go
func (User) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.GraphQL("User",
            dsl.GraphQLSubscriptions(
                dsl.SubscriptionEventCreate,
                dsl.SubscriptionEventUpdate,
                dsl.SubscriptionEventDelete,
            ),
        ),
    }
}
```

Subscribers receive strongly typed payloads—`userCreated` and `userUpdated` streams yield `User` objects, while `userDeleted` emits the global Relay ID that was removed. The generated mutation resolvers publish to the broker automatically once ORM mutations succeed, so subscribers stay in sync without extra code.

Configure transports and the backing broker in `erm.yaml`:

```yaml
graphql:
  path: "/graphql"
  subscriptions:
    enabled: true
    broker: inmemory
    transports:
      websocket: true
      graphql_ws: false
```

The default in-memory broker works for tests and local development. For production plug in your own adapter by passing a custom `subscriptions.Broker` to `server.NewServer` / `resolvers.NewWithOptions` (Redis, NATS, Kafka, etc.). The helpers in `graphql/resolvers/subscriptions.go` expose consistent topic naming (`user:created`, `user:updated`, `user:deleted`) so your producer can emit events independently if needed.

---

## Resolver Implementation

Generated resolvers live in `graphql/resolvers`. Files ending in `_gen.go` (`entities_gen.go`) should not be edited. For custom logic, create extension files (e.g., `user.resolvers_extension.go`) in the same package. The generated stubs already handle:

* Decoding global IDs via the Relay helpers.
* Falling back to the ORM client when a dataloader is unavailable.
* Priming per-entity dataloaders after reads and writes.

### Query Flow

1. **Argument Parsing** – gqlgen decodes incoming arguments into Go structs (pointers for optional input fields so you can detect "not provided").
2. **Privacy Check** – ORM policy/privacy rules run before executing SQL.
3. **Dataloader Registration** – Entity list resolvers register dataloaders generated in `graphql/dataloaders/entities_gen.go` to prevent N+1 patterns.
4. **ORM Execution** – Query builders fetch data via `pgx` with context propagation.
5. **Response Mapping** – Entities convert to GraphQL models using helper functions such as `toGraphQL<User>`. 

### Dataloaders

The `dataloader` package prevents N+1 queries by batching loads. Each entity gets a generated loader in `graphql/dataloaders/entities_gen.go`, and `Resolver.WithLoaders` wires them into the request context. Override settings via schema annotations:

```go
dsl.ToMany("sessions", "LoginSession").
    Ref("user").
    Dataloader(dsl.Loader{Batch: 200, Wait: 2 * time.Millisecond})
```

### Error Handling

- Validation errors bubble up as GraphQL errors with `BAD_USER_INPUT` codes.
- Privacy denials use `PERMISSION_DENIED`.
- Unexpected errors propagate as `INTERNAL` with sanitized messages. The observability package logs full error context.

Custom resolvers can wrap errors using helpers in `graphql/errors`.

---

## Extending the Schema

### Custom Fields

Annotations allow you to expose computed fields without writing manual resolvers:

```go
func (User) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.GraphQL("User").
            ComputedField("profileUrl", dsl.ComputedField{
                GoType: "string",
                Resolver: `return fmt.Sprintf("https://acme.io/users/%s", obj.ID)`,
            }),
    }
}
```

The generator creates a resolver stub for `profileUrl` that you can customize.

### Custom Mutations

Defined via `dsl.Mutation` in the schema (see the schema guide). Generated payloads live in
`graphql/resolvers/<entity>_mutation_extension.go` so you can author business logic there.

### Query Helpers

Add reusable filters in annotations:

```go
dsl.GraphQL("LoginSession").
    FilterPreset("active", `revoked_at IS NULL`).
    FilterPreset("revoked", `revoked_at IS NOT NULL`)
```

The CLI generates helper args so clients can call `loginSessions(preset: ACTIVE)`.

---

## Authorization Integration

Resolvers respect the `@auth` directive generated from schema annotations. Example GraphQL snippet:

```graphql
type Workspace implements Node @auth(require: [ADMIN]) {
  id: ID!
  name: String!
}
```

The directive references claims extracted by the OIDC middleware. In Go, you can fetch viewer info via
`authz.ViewerFromContext(ctx)` and branch on roles or capabilities.

---

## File Layout Overview

```
graphql/
├── schema.graphqls                  # Generated SDL (split when module mode enabled)
├── resolvers/
│   ├── entities_gen.go              # Generated CRUD + connection resolvers – do not edit
│   ├── query.resolvers.go           # Hand-authored entry points (extend as needed)
│   └── user.resolvers_extension.go  # Safe place for custom logic
├── dataloaders/
│   ├── entities_gen.go              # Generated batch loaders per entity
│   └── loader.go                    # Request-scoped loader registration
├── node/
│   └── registry.go                  # Global Node fetch dispatch
└── server/
    └── server.go                    # gqlgen HTTP server setup (Playground, CORS, logging, OIDC)
```

---

## Testing the GraphQL Layer

Use the generated test helpers in `graphql/testutil`:

```go
func TestQueryUsers(t *testing.T) {
    ctx := testutil.ContextWithViewer(t, testutil.Viewer{ID: uuid.MustParse("...")})
    client := testutil.NewGraphQLClient(t)

    resp := struct {
        Users struct {
            TotalCount int
        }
    }{}

    testutil.Query(t, ctx, client, `query { users(first: 1) { totalCount } }`, nil, &resp)
    require.Equal(t, 1, resp.Users.TotalCount)
}
```

The testing guide dives deeper into integration tests, mock dataloaders, and snapshotting GraphQL responses.

---

## Playground & Tooling

- `erm graphql init --playground` enables GraphQL Playground in development; disable it in production via `erm.yaml`.
- `erm gen --dry-run` prints SDL diffs so you can review schema changes before committing.
- Use GraphQL Inspector or Apollo Rover in CI to catch breaking schema changes; export SDL from `graphql/schema.graphqls`.

---

## Troubleshooting

| Issue | Resolution |
|-------|------------|
| `node(id:)` returns null | Ensure the ID is base64-encoded `<Type>:<uuidv7>` and that the Node registry includes the type. |
| Duplicate edges in connection | Confirm dataloader caching is not disabled and review `.BatchSize()` annotations. |
| Mutation denies access | Check entity `Policy()`/`Privacy()` definitions and `@auth` roles; inspect logs for policy evaluation. |
| Playground missing custom headers | Update `graphql/server/server.go` to inject defaults or configure your HTTP client. |

For resolver-specific errors, turn on verbose logging (`ERM_LOG_LEVEL=debug`) to inspect ORM queries and dataloader batches.

---

With the GraphQL layer understood, proceed to [authentication.md](./authentication.md) to secure your API.
