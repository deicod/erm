# Spec — OIDC Middleware

## Requirements
- Validate JWT access tokens via OIDC discovery + JWKS (RS256/ES256).
- Validate `iss` and `aud`.
- Cache keys and refresh per cache-control hints.
- Extract claims into a stable internal struct `Claims` via a pluggable `ClaimsMapper`.

## Default Mapper — Keycloak
- Roles from `realm_access.roles`.
- Standard fields: `sub`, `email`, `name`, `preferred_username`, `given_name`, `family_name`, `email_verified`.
- Example token documented in README and tests.

## API

```go
type Claims struct {
    Subject string
    Email   string
    Name    string
    Username string
    GivenName string
    FamilyName string
    EmailVerified bool
    Roles []string
    Raw map[string]any
}

type ClaimsMapper interface {
    Map(raw map[string]any) (Claims, error)
}
```

- Default `KeycloakClaimsMapper` plus examples for Okta/Auth0.
- GraphQL `@auth` directive to require roles or authenticated user.
