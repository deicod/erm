# Troubleshooting and Common Issues

This guide covers common issues you may encounter when using the erm framework and provides solutions for each.

## Installation Issues

### CLI Installation Problems

**Problem**: Can't install the `erm` CLI tool
```bash
go install github.com/deicod/erm/cmd/erm@latest
# Error: "command not found" after installation
```

**Solution**: Ensure Go bin directory is in your PATH
```bash
# Add to your shell profile (.bashrc, .zshrc, etc.)
export PATH=$PATH:$(go env GOPATH)/bin

# Verify installation
which erm
erm version
```

### Module Initialization Issues

**Problem**: Errors during `go mod tidy` after running `erm init`
```bash
go mod tidy
# Error: module requires a version that doesn't exist
```

**Solution**: Ensure you're using the correct Go version (1.22+) and update dependencies:
```bash
go mod init github.com/yourname/yourproject
go get github.com/deicod/erm@latest
go mod tidy
```

## Schema Definition Issues

### Invalid Schema Syntax

**Problem**: Schema doesn't generate properly due to syntax errors
```go
// Incorrect - missing required imports
type User struct {
    dsl.Schema  // Error: dsl is undefined
}
```

**Solution**: Ensure proper imports and structure
```go
package schema

import "github.com/deicod/erm/internal/orm/dsl"

type User struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
    }
}

func (User) Edges() []dsl.Edge { return nil }
func (User) Indexes() []dsl.Index { return nil }
```

### Circular Dependencies

**Problem**: Schemas reference each other causing circular import issues
```go
// This creates a circular dependency
type User struct{ dsl.Schema }
func (User) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.To("profile", Profile.Type), // Profile references User
    }
}

type Profile struct{ dsl.Schema }
func (Profile) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.From("user", User.Type), // User references Profile
    }
}
```

**Solution**: Ensure proper relationship definitions
```go
// This is valid - only one side needs the foreign key
type User struct{ dsl.Schema }

func (User) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.To("profile", Profile.Type).ForeignKey("user_id"),
    }
}

type Profile struct{ dsl.Schema }

func (Profile) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.From("user", User.Type).ForeignKey("user_id"),
    }
}
```

## Database Connection Issues

### PostgreSQL Connection Problems

**Problem**: Can't connect to PostgreSQL database
```bash
# Error: "dial tcp 127.0.0.1:5432: connect: connection refused"
```

**Solution**: Verify PostgreSQL is running and accessible
```bash
# Check if PostgreSQL is running
sudo systemctl status postgresql  # Linux
brew services info postgresql     # macOS

# Test connection
psql -h localhost -p 5432 -U postgres -d postgres

# Verify erm.yaml configuration
database:
  host: "localhost"    # Ensure correct hostname
  port: 5432           # Ensure correct port
  user: "postgres"     # Ensure correct username
  password: "password" # Ensure correct password
  name: "myproject"    # Ensure database exists
```

### Database Permission Issues

**Problem**: Database operations fail with permission errors
```bash
# Error: "permission denied for database"
```

**Solution**: Ensure the database user has proper permissions
```sql
-- In PostgreSQL
CREATE DATABASE myproject;
GRANT ALL PRIVILEGES ON DATABASE myproject TO postgres;
GRANT ALL PRIVILEGES ON SCHEMA public TO postgres;
```

## Code Generation Issues

### Generation Fails

**Problem**: `erm gen` command fails with errors
```bash
erm gen
# Error: "failed to parse schema files"
```

**Solution**: 
1. Verify all schema files are syntactically correct
2. Run with verbose flag to see detailed errors
3. Check that all necessary imports are present

```bash
erm gen --verbose
```

### Generated Code Conflicts

**Problem**: Generated code conflicts with custom code
```bash
# Error about duplicate functions or types
```

**Solution**: 
- Place custom code in non-generated files
- Use hooks and interceptors for custom business logic
- Don't modify generated files directly

### Schema Changes Not Reflected

**Problem**: After updating schema, changes don't appear in generated code

**Solution**: 
1. Run `erm gen` after every schema change
2. Verify the schema file syntax
3. Check for any errors during generation

## GraphQL API Issues

### Relay Specification Compliance

**Problem**: GraphQL client reports non-compliance with Relay specification
- Missing `node(id:)` resolver
- Incorrect cursor format
- Missing `PageInfo` fields

**Solution**: Ensure you're using the generated Relay-compliant API
```bash
# Verify GraphQL schema generation
erm graphql init  # Reinitialize if needed
erm gen           # Regenerate after schema changes
```

### N+1 Query Problems

**Problem**: Inefficient queries causing performance issues
```bash
# Logs show multiple individual queries for related data
```

**Solution**: Use dataloaders (built-in) and optimize resolver patterns
```go
// Instead of direct queries, use dataloaders
func (r *Resolver) User(ctx context.Context, obj *Post) (*User, error) {
    return r.Loaders.UserLoader.Load(ctx, obj.UserID)  // Uses dataloader
}
```

### Query Complexity Issues

**Problem**: Complex queries timing out or causing performance problems

**Solution**: 
1. Configure query complexity limits in `erm.yaml`
2. Use connection-based pagination
3. Implement proper indexing
4. Consider query optimization

## Authentication and Authorization Issues

### OIDC Configuration Problems

**Problem**: Authentication fails or JWT validation errors
```bash
# Error: "oidc: JWT verification failed"
```

**Solution**: Verify OIDC configuration
```yaml
oidc:
  issuer: "https://your-keycloak/realms/myrealm"  # Ensure correct issuer
  client_id: "myapp"                              # Ensure correct client ID
  jwks_url: "https://your-keycloak/realms/myrealm/protocol/openid-connect/certs"  # Ensure accessible
```

### Claim Mapping Issues

**Problem**: User roles or information not properly mapped

**Solution**: 
1. Verify the claims mapper configuration
2. Check JWT token structure matches expected format
3. Enable debug logging to see claim extraction

```yaml
oidc:
  claims_mapper: "keycloak"  # Or "auth0", "okta", etc.
```

### Authorization Bypass

**Problem**: Privacy policies not being enforced

**Solution**: 
1. Verify privacy policy syntax in schema
2. Ensure OIDC context is properly injected
3. Check that `@auth` directives are applied correctly

## Migration Issues

### Migration Failures

**Problem**: Migrations fail to apply
```bash
erm migrate up
# Error: "relation does not exist" or "duplicate key value violates unique constraint"
```

**Solution**: 
1. Check migration order and dependencies
2. Ensure database is in consistent state
3. Manually fix migration state if needed

```bash
# Check migration status
erm migrate status

# Rollback if needed
erm migrate down
```

### Schema Mismatch

**Problem**: Generated code expects database schema that doesn't exist

**Solution**: Ensure migrations are applied before starting application
```bash
erm migrate up
# Then start your application
```

## Extension-Specific Issues

### PostGIS Not Available

**Problem**: Geometric field types fail during generation or runtime
```bash
# Error: "extension postgis does not exist"
```

**Solution**: Ensure PostGIS is installed in your PostgreSQL instance
```sql
-- Connect to your database and run
CREATE EXTENSION IF NOT EXISTS postgis;
```

### pgvector Errors

**Problem**: Vector operations fail
```bash
# Error: "type vector does not exist"
```

**Solution**: Install pgvector extension
```sql
-- In PostgreSQL
CREATE EXTENSION IF NOT EXISTS vector;
```

### TimescaleDB Setup Issues

**Problem**: TimescaleDB features not working
```bash
# Error: "extension timescaledb does not exist"
```

**Solution**: Install TimescaleDB extension
```sql
-- In PostgreSQL
CREATE EXTENSION IF NOT EXISTS timescaledb;
```

## Performance Issues

### Slow Queries

**Problem**: API requests are slow
1. Check for N+1 queries
2. Verify proper indexing
3. Monitor database performance

**Solutions**:
- Enable query logging to identify slow queries
- Add appropriate indexes
- Use connection-based pagination
- Implement caching where appropriate

### High Memory Usage

**Problem**: Application uses excessive memory
- Large dataset queries loading too much data
- Memory leaks in long-running operations

**Solutions**:
- Use pagination for large datasets
- Implement streaming for large result sets
- Monitor and optimize your code for memory usage

## Debugging Strategies

### Enable Verbose Logging

For detailed debugging information:
```bash
# Set environment variable for detailed logging
export ERM_DEBUG=true
# Or in your erm.yaml
logging:
  level: debug
  include_sql: true
  include_oidc: true
```

### Check Generated Code

Review generated code for any issues:
- Look for template errors
- Verify proper imports
- Check for syntax errors

### Use CLI Diagnostic Commands

```bash
# Check configuration
erm doctor  # When available

# Verbose generation
erm gen --verbose

# Dry run to preview changes
erm gen --dry-run
```

### Database Debugging

Enable PostgreSQL query logging:
```sql
-- In PostgreSQL, enable logging
SET log_statement = 'all';
SET log_min_duration_statement = 0;
```

## Getting Help

### When to File Issues

File GitHub issues for:
- Bugs in the framework
- Unexpected behavior
- Feature requests
- Documentation needs

### Useful Information for Issues

When reporting issues, include:
- erm version (`erm version`)
- Go version (`go version`)
- PostgreSQL version
- Steps to reproduce
- Expected vs. actual behavior
- Error messages and logs
- Relevant configuration files

### Community Resources

- Check existing issues before filing new ones
- Provide pull requests for fixes when possible
- Contribute to documentation improvements

## Common Solutions Checklist

Before opening an issue, verify:

- [ ] Go version is 1.22+
- [ ] PostgreSQL is running and accessible
- [ ] Schema files are syntactically correct
- [ ] You've run `erm gen` after schema changes
- [ ] Migrations have been applied
- [ ] Configuration files are properly formatted
- [ ] Extensions are installed (PostGIS, pgvector, etc.)
- [ ] OIDC configuration is correct
- [ ] You're using the latest version of erm
- [ ] Generated code hasn't been manually modified

If issues persist after checking these items, provide detailed information about the problem when seeking help.