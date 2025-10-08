# Authentication and Authorization

The erm framework provides a robust OIDC (OpenID Connect) authentication system with pluggable claim mapping, primarily optimized for Keycloak but supporting other OIDC providers like Auth0, Okta, and custom implementations.

## Architecture

The authentication system is built around:

1. **OIDC Middleware** - Verifies JWT tokens using JWKS (JSON Web Key Set)
2. **Claim Mappers** - Pluggable interface for mapping provider-specific claims
3. **GraphQL Directives** - `@auth` directive for securing resolvers
4. **Privacy Policies** - Fine-grained access control in schema definitions

## Configuration

Authentication is configured primarily through the `erm.yaml` file:

```yaml
oidc:
  issuer: "https://your-keycloak-domain/realms/your-realm"
  client_id: "your-client-id"
  jwks_url: "https://your-keycloak-domain/realms/your-realm/protocol/openid-connect/certs"
  audience: ["your-audience"]  # Optional: validate token audience
  claims_mapper: "keycloak"    # Available: keycloak, auth0, okta, custom
```

## OIDC Middleware

The OIDC middleware handles:

- JWT signature verification using JWKS
- Token expiration validation
- Issuer validation against configured issuer URL
- Audience validation (if configured)
- Claims extraction and context injection

### Key Features

- **RS256/ES256 Support**: Secure signature verification algorithms
- **Automatic JWKS Refresh**: Periodic fetching of updated signing keys
- **Issuer Validation**: Ensures tokens come from trusted issuers
- **Context Injection**: Makes user information available in GraphQL resolvers

## Claim Mappers

The system uses a pluggable `ClaimsMapper` interface to handle provider-specific claim structures:

### Keycloak Claims Mapper (Default)

Keycloak-specific mapping extracts:
- User ID from `sub` field
- Roles from `realm_access.roles`
- Email from `email` field
- Name from `name` field
- Preferred username from `preferred_username`

### Other Providers

The framework includes mappers for:
- **Auth0**: Adapts to Auth0's claim structure
- **Okta**: Adapts to Okta's claim structure  
- **Custom**: Pluggable mapper interface for any OIDC provider

### Implementing Custom Mappers

```go
type ClaimsMapper interface {
    MapClaims(token *oidc.IDToken, rawClaims json.RawMessage) (*UserContext, error)
}

type CustomClaimsMapper struct{}

func (c *CustomClaimsMapper) MapClaims(token *oidc.IDToken, rawClaims json.RawMessage) (*UserContext, error) {
    // Extract custom claims from rawClaims
    // Map to UserContext struct
    return &UserContext{
        ID:    extractCustomID(rawClaims),
        Roles: extractCustomRoles(rawClaims),
        Email: extractCustomEmail(rawClaims),
    }, nil
}
```

## GraphQL @auth Directive

The `@auth` directive can be applied to GraphQL fields, queries, or mutations to enforce authentication:

```graphql
type Mutation {
    createUser(input: UserInput!): User! @auth
    updateUser(id: ID!, input: UserInput!): User! @auth
}

type Query {
    me: User! @auth
    users: [User!]! @auth
}
```

## Context Injection

Authenticated user information is automatically injected into GraphQL context:

```go
// In resolver functions
func (r *Resolver) Me(ctx context.Context) (*model.User, error) {
    // Extract user context from GraphQL context
    userCtx := context.UserFromContext(ctx)
    
    if userCtx == nil {
        return nil, errors.New("unauthenticated")
    }
    
    // Access user information
    userID := userCtx.ID()
    roles := userCtx.Roles()
    
    // Proceed with operation
    return r.UserService.GetByID(ctx, userID)
}
```

## Privacy Policies

Fine-grained authorization can be implemented using privacy policies in schema definitions:

```go
func (User) Privacy() dsl.Privacy {
    return dsl.Privacy{
        Read:   "user.id == context.UserID || 'admin' in context.Roles",
        Write:  "user.id == context.UserID",
        Create: "'create_user' in context.Roles",
        Delete: "'admin' in context.Roles",
    }
}
```

The privacy policies are evaluated in the GraphQL resolvers and database queries to ensure that users can only access data they're authorized to see.

## Keycloak Integration

The default Keycloak integration assumes the following setup:

### Realm Setup
- Realm with users and roles
- Client configured with valid redirect URIs
- Appropriate roles assigned to users

### Role Mapping
- Realm roles are mapped to permissions in privacy policies
- `realm_access.roles` contains user roles
- Supports hierarchical role structures

### Token Configuration
- RS256 signed tokens (recommended)
- Appropriate audience configuration
- Token expiration (default 5 minutes for access tokens)

## Security Best Practices

### Token Handling
- Use HTTPS in production
- Validate token audience when possible
- Implement proper token refresh strategies
- Monitor for token replay attacks

### Role and Permission Design
- Follow principle of least privilege
- Use roles for coarse-grained access control
- Use privacy policies for fine-grained access control
- Regularly audit permissions and roles

### JWKS Caching
- Configure appropriate cache TTL (typically 5-15 minutes)
- Monitor JWKS refresh failures
- Implement fallback strategies for JWKS unavailability

## Troubleshooting

### Common Issues

1. **Token Verification Failures**:
   - Verify JWKS endpoint is accessible
   - Check issuer URL matches token issuer
   - Confirm supported signature algorithm

2. **Claim Mapping Issues**:
   - Verify claims mapper configuration
   - Check provider-specific claim structure
   - Enable debug logging for claim extraction

3. **Authorization Problems**:
   - Review privacy policy logic
   - Verify role assignments in OIDC provider
   - Check context injection in resolvers

### Debugging Tips

Enable detailed logging for OIDC operations:
```yaml
logging:
  level: debug
  include_oidc: true
```

Check the GraphQL context in resolvers to verify user information is properly injected.

## Testing

The framework provides utilities for testing authenticated operations:

```go
// Create test context with authenticated user
ctx := context.WithUser(context.Background(), &UserContext{
    ID:    "test-user-id",
    Roles: []string{"admin"},
    Email: "test@example.com",
})
```

Mock OIDC providers can be used for integration testing without external dependencies.