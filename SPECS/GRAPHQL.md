# Spec — GraphQL (Relay)

## Targets
- **Relay Server Spec compliance**.
- `gqlgen`-based schema/resolvers, with optional `autobind` to ORM types once Node helpers and enum wrapper shims are available.

## Global Object IDs
- Encode as base64 of `<Type>:<uuidv7>`.
- Provide `DecodeGlobalID` returning `(typeName, nativeID)`.

## Node Interface
```graphql
interface Node {
  id: ID!
}
```

- Root field `node(id: ID!): Node` resolves via registry mapping type → fetcher.
- Every entity implements `Node` and exposes `id` (global).

## Connections & Pagination
- Support `first/after` and `last/before` with opaque cursors.
- `PageInfo { hasNextPage, hasPreviousPage, startCursor, endCursor }`.
- Generators produce standard `<Type>Connection` and `<Type>Edge` structs.
- Builders paginate via ORM predicates and orderings.

## Dataloaders
- Per-entity loaders map ids → rows; batched per request; context-scoped.

## Auth
- Directive `@auth(roles: [String!])` enforces roles in request context.
- Middleware attaches `Claims` onto context.
- Default resolvers guard mutations with `@auth(roles: ["user"])`, customizable.

## gqlgen
- Default `gqlgen.yml` ships without `autobind`; re-enable it via `erm graphql init --autobind` or manual config only after the Node helpers and enum wrappers are generated to satisfy gqlgen's binding requirements.
- Generated `schema.graphqls` + stubs live under `graphql`.
