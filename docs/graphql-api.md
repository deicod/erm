# GraphQL API

The erm framework generates a fully Relay-compliant GraphQL API that follows the Relay Server Specification. This includes global object IDs, cursor-based pagination, connections/edges, and more.

## Relay Specification Compliance

The generated GraphQL API implements the following Relay features:

### Global Object IDs
- Each entity is assigned a globally unique ID following the format: `base64("<Type>:<uuidv7>")`
- The `node(id:)` query resolver can fetch any entity by its global ID
- Automatic encoding/decoding between UUID v7 and global object IDs

### Connections and Pagination
- Cursor-based pagination using `first`, `last`, `after`, `before` parameters
- `Connection`, `Edge`, and `PageInfo` types for consistent pagination
- Opaque cursors that don't expose internal database structure

## Generated GraphQL Types

For each schema, the framework generates:

1. **Object Type** - Represents the entity with all its fields
2. **Input Types** - For mutations and filtering
3. **Connection Type** - For paginated lists
4. **Edge Type** - Represents a connection between two nodes
5. **Filter Types** - For querying with conditions

### Example Generated Types

Given a User schema:

```go
type User struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.String("name").Size(255),
        dsl.String("email").Size(255).Unique(),
        dsl.Time("created_at").DefaultNow(),
    }
}
```

The framework generates GraphQL types like:

```graphql
type User {
    id: ID!
    name: String!
    email: String!
    createdAt: Time!
}

type UserEdge {
    node: User
    cursor: String!
}

type UserConnection {
    edges: [UserEdge!]!
    pageInfo: PageInfo!
    totalCount: Int!
}

input UserFilter {
    name: String
    email: String
    emailContains: String
    createdAtAfter: Time
    createdAtBefore: Time
}
```

## Query Operations

The framework generates several types of queries:

### Single Entity Queries
```graphql
# Fetch a single user by ID
query {
    user(id: "VXNlcjoxMjM0NTY3ODktMTIzNC0xMjM0LTEyMzQtMTIzNDU2Nzg5MDEy") {
        id
        name
        email
    }
}

# Fetch by unique field
query {
    userByEmail(email: "example@example.com") {
        id
        name
        email
    }
}
```

### List Queries (Connections)
```graphql
# Fetch users with pagination
query {
    users(first: 10, after: "cursor") {
        edges {
            node {
                id
                name
                email
            }
            cursor
        }
        pageInfo {
            hasNextPage
            hasPreviousPage
            startCursor
            endCursor
        }
        totalCount
    }
}

# Fetch with filters
query {
    users(filter: { name: "John" }) {
        edges {
            node {
                id
                name
                email
            }
        }
        pageInfo {
            hasNextPage
            hasPreviousPage
        }
    }
}
```

### Global Node Interface
```graphql
# Fetch any node by global ID
query {
    node(id: "VXNlcjoxMjM0NTY3ODktMTIzNC0xMjM0LTEyMzQtMTIzNDU2Nzg5MDEy") {
        id
        ... on User {
            name
            email
        }
    }
}
```

## Mutation Operations

The framework generates Create, Update, and Delete mutations:

### Create Mutations
```graphql
mutation {
    createUser(input: {
        name: "John Doe"
        email: "john@example.com"
    }) {
        user {
            id
            name
            email
        }
    }
}
```

### Update Mutations
```graphql
mutation {
    updateUser(id: "VXNlcjoxMjM0NTY3ODktMTIzNC0xMjM0LTEyMzQtMTIzNDU2Nzg5MDEy", input: {
        name: "John Smith"
    }) {
        user {
            id
            name
            email
        }
    }
}
```

### Delete Mutations
```graphql
mutation {
    deleteUser(id: "VXNlcjoxMjM0NTY3ODktMTIzNC0xMjM0LTEyMzQtMTIzNDU2Nzg5MDEy") {
        success
    }
}
```

## Edge and Connection Queries

For relationships defined in schemas, the framework generates edge queries:

```graphql
query {
    user(id: "VXNlcjoxMjM0NTY3ODktMTIzNC0xMjM0LTEyMzQtMTIzNDU2Nzg5MDEy") {
        id
        name
        posts(first: 5) {  # Assuming User has many Posts
            edges {
                node {
                    id
                    title
                    content
                }
                cursor
            }
            pageInfo {
                hasNextPage
                hasPreviousPage
            }
        }
    }
}
```

## Filtering and Sorting

Filtering is available on connection queries:

```graphql
query {
    users(
        first: 10
        filter: {
            name: "John"
            emailContains: "@example.com"
        }
        orderBy: { field: "createdAt", direction: "DESC" }
    ) {
        edges {
            node {
                id
                name
                email
                createdAt
            }
            cursor
        }
        pageInfo {
            hasNextPage
            hasPreviousPage
        }
    }
}
```

## Dataloader Integration

The generated code includes dataloader integration to prevent N+1 query problems:

- Automatic batching of queries
- Caching of results within a request context
- Efficient resolution of relationships

## Error Handling

The GraphQL API follows GraphQL error handling conventions:

- Validation errors with descriptive messages
- Authorization errors when privacy policies are violated
- Database constraint errors mapped to GraphQL errors

## Performance Features

### Dataloaders
- Batch related queries to prevent N+1 problems
- Cache results within request context

### Query Optimization
- Automatic index creation based on schema relationships
- Efficient query generation for complex joins
- Connection-based pagination to avoid loading large datasets at once

### Tracing
- OpenTelemetry integration for observability
- Query execution timing and performance metrics

## Customization

While the generated API covers most use cases, you can customize:

1. **Additional Resolvers** - Add custom queries/mutations in the GraphQL generator templates
2. **Validation Logic** - Add custom validation in hooks or interceptors
3. **Authorization Logic** - Implement custom privacy policies based on context
4. **Query Complexity** - Configure query depth limits and complexity analysis