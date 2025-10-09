# Authentication & Authorization

erm ships with first-class OIDC integration, JWT verification, and flexible claims mapping so you can secure your GraphQL API
without writing boilerplate. This guide covers configuration, middleware flow, authorization directives, and common customizations.

---

## Overview

- **OIDC Discovery** – The generated middleware fetches the provider configuration (JWKS URI, token endpoints) from the issuer
  URL defined in `erm.yaml`.
- **JWKS Cache** – Public keys are cached with configurable refresh intervals to minimize network latency.
- **Claims Mapping** – A pluggable interface turns JWT claims into a `Viewer` struct with roles, permissions, and identity
  fields.
- **GraphQL Enforcement** – `@auth` directives and entity privacy rules enforce authorization before hitting the database.

---

## Configuring OIDC

Edit the `oidc` block in `erm.yaml`:

```yaml
oidc:
  issuer: http://localhost:8080/realms/demo
  audience: erm-api
  required_scopes:
    - openid
    - profile
  jwks_cache_ttl: 5m
  claims_mapper: keycloak
```

Supported mappers: `keycloak` (default), `auth0`, `okta`, or custom implementations registered under `internal/oidc/mapper`.

After updating configuration, run `erm gen` so generated middleware picks up new defaults.

---

## Middleware Flow

1. **Token Extraction** – `internal/oidc/middleware.go` pulls the `Authorization: Bearer <token>` header.
2. **Verification** – The token is validated using the provider’s JWKS. Signature, expiration, issuer, and audience checks run
   automatically.
3. **Claims Mapping** – The mapper converts raw claims into a `Viewer` struct (`ID`, `Email`, `Name`, `Roles`, `Permissions`).
4. **Context Injection** – The viewer is stored in the request context; errors short-circuit with `UNAUTHENTICATED` GraphQL
   responses.
5. **Directive Enforcement** – GraphQL resolvers read viewer data to evaluate `@auth` directives and privacy rules.

The middleware is inserted in `internal/graphql/server/server.go` during `erm graphql init`.

---

## Custom Claims Mapping

Create a new mapper by implementing the `ClaimsMapper` interface:

```go
package mapper

type CustomMapper struct{}

func (CustomMapper) Map(ctx context.Context, token *oidc.IDToken, claims map[string]interface{}) (authz.Viewer, error) {
    roles := authz.RolesFromPath(claims, "app_roles")
    return authz.Viewer{
        ID:    claims["sub"].(string),
        Email: claims["email"].(string),
        Name:  claims["preferred_username"].(string),
        Roles: roles,
    }, nil
}
```

Register the mapper in `internal/oidc/mapper/registry.go` and reference it in `erm.yaml`.

### Mapping Tips

- Normalize role names (e.g., uppercase) so comparisons in privacy rules remain consistent.
- Populate `Permissions` when you need fine-grained capability checks beyond coarse roles.
- Add audit logging within the mapper if you need to trace access decisions.

---

## Authorization in GraphQL

Use annotations to declare access requirements:

```go
func (Workspace) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.Authz().Roles("ADMIN", "OWNER"),
        dsl.GraphQL("Workspace").Description("A collaborative space for teams."),
    }
}
```

This generates a GraphQL directive:

```graphql
type Workspace implements Node @auth(require: [ADMIN, OWNER]) {
  id: ID!
  name: String!
}
```

Resolvers call `authz.Check(ctx, authz.RequireRoles("ADMIN", "OWNER"))`. Failures return `PERMISSION_DENIED` before ORM queries
run.

### Field-Level Guards

```go
func (Invoice) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.GraphQL("Invoice").FieldAuth("amount", dsl.RequireRoles("FINANCE")),
    }
}
```

Field guards hide sensitive data while still allowing the node to resolve.

---

## Combining with Privacy Policies

Privacy rules complement `@auth` directives by running inside the ORM layer.

```go
func (Workspace) Policy() dsl.Policy {
    return dsl.Policy{
        Query: dsl.AllowIf("viewer.has_role('ADMIN') || viewer.has_role('OWNER')"),
        Update: dsl.AllowIf("viewer.has_role('OWNER')"),
    }
}
```

If a resolver bypasses GraphQL directives (e.g., via dataloaders), privacy still protects the data.

---

## Handling Machine-to-Machine Tokens

For service accounts without user context, configure an additional mapper that extracts capabilities from JWT claims:

```yaml
oidc:
  service_mappers:
    - name: ci
      match_claim: { field: client_id, equals: erm-ci }
      roles: [SYSTEM]
```

The middleware chooses the mapper based on matching claims, allowing you to grant CI pipelines limited access.

---

## Local Development with Keycloak

1. Run Keycloak via Docker (see `examples/docker-compose/keycloak.yml`).
2. Configure realm, client, and user as described in the README.
3. Update `erm.yaml` with the local issuer URL (`http://localhost:8080/realms/erm`).
4. Start the GraphQL server; the middleware will download JWKS automatically.

Use `scripts/get-token.sh` to fetch a test access token for GraphQL Playground requests.

---

## Refresh and Revocation

- Tokens are verified on every request; if you need revocation, configure short-lived access tokens and rely on refresh tokens in
  your client application.
- JWKS keys rotate automatically—set `jwks_cache_ttl` to a low value (e.g., 1m) in environments with frequent rotations.

---

## Troubleshooting Authentication

| Symptom | Resolution |
|---------|------------|
| `401 Unauthorized` on every request | Confirm the `Authorization` header includes `Bearer`. Check that issuer/audience in
  `erm.yaml` matches the token. |
| Tokens accepted locally but rejected in production | Ensure clocks are synchronized. Consider enabling `leeway` in the middleware
  configuration for small clock skews. |
| Roles missing in resolvers | Verify the mapper extracts them correctly. Add `ERM_LOG_LEVEL=debug` to inspect mapped viewers. |
| Privacy rules always deny | Check that the viewer context is present. If you skip middleware (e.g., for admin scripts), inject a
  viewer manually using `authz.WithViewer(ctx, viewer)`. |

---

## Next Steps

- Continue to [extensions.md](./extensions.md) to leverage PostgreSQL extensions.
- Review [performance-observability.md](./performance-observability.md) to monitor authentication latency and cache stats.
