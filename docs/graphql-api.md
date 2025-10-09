# GraphQL API

erm generates a complete, Relay-compliant GraphQL server using gqlgen. This document explains the schema topology, resolver
implementation patterns, dataloader integration, and how to extend the generated API with custom operations.

---

## Relay Compliance at a Glance

- **Global Node IDs** – Every entity implements the `Node` interface. IDs are encoded as `base64("<Type>:<uuidv7>")`. Helpers
  live in `internal/graphql/node` and the ORM automatically decodes/encodes IDs in builders.
- **Connections & Edges** – Pagination follows the Relay spec using `first`, `last`, `after`, and `before`. Connections expose
  `edges`, `pageInfo`, and `totalCount`.
- **Mutations** – Generated CRUD mutations use input objects and payload objects that include `clientMutationId` for optimistic
  UI patterns.
- **Viewer Context** – Resolvers receive viewer context (from OIDC middleware) and pass it through to privacy rules before
  hitting the database.

---

## Generated Schema Structure

For each entity, erm produces GraphQL types in `internal/graphql/schema.graphqls` (or split across modules if you enable module
mode). Using a `User` entity as an example:

```graphql
type User implements Node {
  id: ID!
  email: String!
  displayName: String
  isAdmin: Boolean!
  createdAt: Time!
  updatedAt: Time!
  posts(after: Cursor, first: Int, before: Cursor, last: Int, orderBy: PostOrder): PostConnection!
}

type UserEdge {
  node: User!
  cursor: Cursor!
}

type UserConnection {
  totalCount: Int!
  edges: [UserEdge!]!
  pageInfo: PageInfo!
}

input UserFilter {
  email: String
  emailContains: String
  isAdmin: Boolean
  createdAtAfter: Time
  createdAtBefore: Time
}

input UserOrder {
  direction: OrderDirection! = ASC
  field: UserOrderField!
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
      email
      displayName
    }
  }
}
```

The `node` resolver dispatches to `FindNodeByID` generated in `internal/graphql/node/registry.go`. You can register additional
node types (e.g., view projections) via annotations.

### Entity Queries

```graphql
query Users($first: Int!, $after: Cursor) {
  users(first: $first, after: $after, orderBy: { field: CREATED_AT, direction: DESC }) {
    edges {
      cursor
      node {
        id
        email
        posts(first: 5) {
          totalCount
        }
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

For fields marked `.Unique()`, the generator exposes `userByEmail`, `workspaceBySlug`, etc. Example:

```graphql
query UserByEmail($email: String!) {
  userByEmail(email: $email) {
    id
    email
    displayName
  }
}
```

### Mutations

Every entity gets `create<Entity>`, `update<Entity>`, and `delete<Entity>` mutations. They follow the Relay payload pattern:

```graphql
mutation CreatePost($input: CreatePostInput!) {
  createPost(input: $input) {
    post {
      id
      title
      body
      author {
        id
        email
      }
    }
    clientMutationId
  }
}
```

`clientMutationId` is echoed back automatically to support optimistic updates. Partial updates use `Update<Entity>Input` where
optional fields map to pointer types in Go, allowing you to distinguish between "null" and "not provided".

### Subscriptions (Optional)

If you enable subscriptions in `erm.yaml`, the generator adds stub definitions and resolver scaffolding. Wire them up using your
preferred pub/sub implementation.

---

## Resolver Implementation

Generated resolvers live in `internal/graphql/resolver`. Files ending in `_generated.go` should not be edited. For custom logic,
create extension files (e.g., `user.resolvers_extension.go`).

### Query Flow

1. **Argument Parsing** – gqlgen decodes incoming arguments into Go structs.
2. **Privacy Check** – ORM policy/privay rules run before executing SQL.
3. **Dataloader Registration** – To-many edges automatically register dataloaders defined in `internal/graphql/dataloader`.
4. **ORM Execution** – Query builders fetch data via `pgx` with context propagation.
5. **Response Mapping** – Entities convert to GraphQL models, applying field-level annotations (e.g., rename `display_name` →
   `displayName`).

### Dataloaders

The `dataloader` package prevents N+1 queries by batching loads. Each edge has a loader with configurable batch size and
caching strategy. Override settings via schema annotations:

```go
dsl.ToMany("posts", "Post").
    Ref("author").
    Dataloader(dsl.Loader{Batch: 200, Wait: 2 * time.Millisecond})
```

### Error Handling

- Validation errors bubble up as GraphQL errors with `BAD_USER_INPUT` codes.
- Privacy denials use `PERMISSION_DENIED`.
- Unexpected errors propagate as `INTERNAL` with sanitized messages. The observability package logs full error context.

Custom resolvers can wrap errors using helpers in `internal/graphql/errors`.

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
`internal/graphql/resolver/<entity>_mutation_extension.go` so you can author business logic there.

### Query Helpers

Add reusable filters in annotations:

```go
dsl.GraphQL("Post").
    FilterPreset("published", `published_at IS NOT NULL`).
    FilterPreset("draft", `published_at IS NULL`)
```

The CLI generates helper args so clients can call `posts(preset: PUBLISHED)`.

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
internal/graphql/
├── schema.graphqls              # Generated SDL (split when module mode enabled)
├── resolver/
│   ├── generated.go             # Boilerplate wiring – do not edit
│   ├── user.resolvers.go        # Generated resolvers per entity
│   ├── user.resolvers_extension.go  # Safe to edit
│   └── ...
├── dataloader/
│   ├── registry.go              # Loader registration invoked per request
│   └── user_loader.go           # Generated loader for edges
├── node/
│   └── registry.go              # Global Node fetch dispatch
└── server/
    └── server.go                # gqlgen HTTP server setup (Playground, CORS, logging, OIDC)
```

---

## Testing the GraphQL Layer

Use the generated test helpers in `internal/graphql/testutil`:

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
- Use GraphQL Inspector or Apollo Rover in CI to catch breaking schema changes; export SDL from `internal/graphql/schema.graphqls`.

---

## Troubleshooting

| Issue | Resolution |
|-------|------------|
| `node(id:)` returns null | Ensure the ID is base64-encoded `<Type>:<uuidv7>` and that the Node registry includes the type. |
| Duplicate edges in connection | Confirm dataloader caching is not disabled and review `.BatchSize()` annotations. |
| Mutation denies access | Check entity `Policy()`/`Privacy()` definitions and `@auth` roles; inspect logs for policy evaluation. |
| Playground missing custom headers | Update `internal/graphql/server/server.go` to inject defaults or configure your HTTP client. |

For resolver-specific errors, turn on verbose logging (`ERM_LOG_LEVEL=debug`) to inspect ORM queries and dataloader batches.

---

With the GraphQL layer understood, proceed to [authentication.md](./authentication.md) to secure your API.
