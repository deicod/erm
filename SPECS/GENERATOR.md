# Spec — Generator

## CLI
- `erm init`: scaffold project files (Makefile, .github, orm dirs, example config).
- `erm new <Entity>`: create `schema/<Entity>.schema.go` with boilerplate.
- `erm gen`: parse schema, generate ORM and GraphQL artifacts, migrations, registries.
- `erm graphql init`: add gqlgen config, schema, resolvers; wire server main.

## Parsing
- Use Go AST to load `schema/*.schema.go`.
- DSL package types drive semantics (fields, edges, indexes, annotations).
- Emit code via `text/template` with embedded templates; idempotent file writes.

## Outputs
- `/orm/gen/*`: entity structs, builders, registries.
- `/graphql/gen/*`: gql types, connection types, resolvers stubs.
- `/migrations/*`: versioned SQL.

## Idempotency
- Writes only if content changed; keep generated blocks behind markers.
