# Best Practices

This document outlines best practices for using the erm framework effectively, covering schema design, performance optimization, security considerations, and maintainability.

## Schema Design Best Practices

### Naming Conventions
- Use PascalCase for entity names: `User`, `Post`, `Comment`
- Use camelCase for field names: `firstName`, `createdAt`, `emailAddress`
- Use descriptive names that clearly indicate purpose
- Avoid abbreviations unless they are widely understood

### ID Management
- Always use UUID v7 for primary keys (default in erm)
- Leverage the global object ID system for Relay compatibility
- Don't expose internal database IDs directly to clients

### Field Design
- Specify appropriate sizes for string fields to optimize storage and queries
- Use proper data types (Time for timestamps, Bool for boolean values)
- Apply constraints where appropriate (Unique, Required)
- Be intentional about nullable vs. required fields

```go
// Good
dsl.String("email").Size(255).Unique().Required()
dsl.Time("created_at").DefaultNow()

// Avoid unless necessary
dsl.String("data").Required() // No size limit
```

### Relationship Design
- Define both sides of relationships for clarity
- Use proper foreign key constraints
- Consider the impact of relationship depth on query performance
- Use eager loading for frequently accessed relationships

```go
// Good relationship definition
type User struct{ dsl.Schema }

func (User) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.To("posts", Post.Type).ForeignKey("user_id"),
    }
}

type Post struct{ dsl.Schema }

func (Post) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.From("user", User.Type).ForeignKey("user_id"),
    }
}
```

## Performance Best Practices

### Indexing Strategy
- Create indexes for fields frequently used in WHERE clauses
- Use composite indexes for multi-field queries
- Index foreign keys used in joins
- Add indexes for fields used in ORDER BY clauses
- Monitor index usage and remove unused indexes

```go
func (User) Indexes() []dsl.Index {
    return []dsl.Index{
        dsl.Index().Fields("email"),                    // For user lookup by email
        dsl.Index().Fields("created_at").Desc(),        // For chronological queries
        dsl.Index().Fields("status", "created_at"),     // For status-based queries with ordering
    }
}
```

### Pagination
- Always use Relay-style pagination for list queries
- Implement proper cursor-based pagination for large datasets
- Avoid offset-based pagination for large result sets
- Set reasonable default limits to prevent overly large responses

### Query Optimization
- Use eager loading for predictable relationship access patterns
- Leverage dataloaders to prevent N+1 queries
- Fetch only the fields you need (GraphQL's selectivity is a feature)
- Use connection-based queries instead of fetching entire collections

### Database Configuration
- Configure appropriate connection pool sizes based on your application's needs
- Set reasonable timeout values
- Enable prepared statement caching
- Monitor and tune PostgreSQL configuration for your workload

## Security Best Practices

### Authentication
- Always validate OIDC tokens properly
- Use HTTPS in production environments
- Implement proper token refresh mechanisms
- Log authentication failures for security monitoring

### Authorization
- Use privacy policies for fine-grained access control
- Validate user permissions at both API and database levels
- Implement role-based access control where appropriate
- Regularly audit access patterns and permissions

```go
func (User) Privacy() dsl.Privacy {
    return dsl.Privacy{
        Read:   "user.id == context.UserID || 'admin' in context.Roles",
        Write:  "user.id == context.UserID",
        Create: "'user_management' in context.Roles",
        Delete: "'admin' in context.Roles",
    }
}
```

### Input Validation
- Validate all inputs at the API boundary
- Use GraphQL input types for structured validation
- Implement proper sanitization for user-provided content
- Set appropriate limits on input sizes

### Data Protection
- Encrypt sensitive data at rest when necessary
- Use proper data retention policies
- Implement proper data export and deletion procedures
- Audit data access patterns

## Architecture Best Practices

### Separation of Concerns
- Keep business logic in hooks and interceptors
- Use privacy policies for access control
- Maintain clear boundaries between schema, business logic, and presentation layers
- Organize code by domain instead of technical layer

### Configuration Management
- Store configuration in environment variables for production
- Use the erm.yaml file for application configuration
- Separate configuration for different environments
- Don't hardcode configuration values in the schema

### Error Handling
- Implement proper error handling in hooks and interceptors
- Return appropriate error codes and messages
- Log errors with sufficient context for debugging
- Don't expose internal errors to clients

## Testing Best Practices

### Test Organization
- Write tests for all business logic
- Use integration tests to verify database operations
- Test authentication and authorization flows
- Include performance benchmarks for critical operations

### Test Data Management
- Use factories for creating consistent test data
- Isolate test data between tests
- Clean up test data after tests
- Use transactions or savepoints for test data rollback

## Development Workflow Best Practices

### Version Control
- Use feature branches for schema changes
- Include migration files in schema change commits
- Review generated code changes in pull requests
- Use semantic versioning for releases

### Code Generation
- Regenerate code after schema changes
- Commit generated code to version control
- Test generated code changes thoroughly
- Use `erm gen --dry-run` to preview changes

### Schema Evolution
- Make backward-compatible changes when possible
- Plan migration strategies for breaking changes
- Test schema changes thoroughly before applying to production
- Document breaking changes clearly

## Monitoring and Observability Best Practices

### Metrics Collection
- Monitor GraphQL query performance
- Track error rates and types
- Monitor database query performance
- Track authentication and authorization metrics

### Logging
- Log significant business events
- Include proper correlation IDs for request tracing
- Log authentication and authorization decisions
- Use structured logging for easier analysis

### Health Checks
- Implement comprehensive health checks for all services
- Monitor database connectivity
- Check external service dependencies
- Include business logic health indicators

## Extension-Specific Best Practices

### PostGIS
- Use appropriate geometric types for your use case
- Create spatial indexes for geometric queries
- Consider coordinate system implications
- Validate geometric data before storage

### pgvector
- Choose the right index type for your query patterns
- Consider the trade-off between dimensionality and performance
- Use appropriate similarity metrics for your use case
- Monitor vector index performance

### TimescaleDB
- Choose appropriate chunk sizes for your time-series patterns
- Use compression for older historical data
- Implement proper data retention policies
- Monitor hypertable performance metrics

## Maintenance Best Practices

### Regular Tasks
- Regularly update dependencies
- Monitor and optimize database performance
- Review and clean up old migrations
- Update privacy policies as requirements change

### Documentation
- Keep schema documentation up to date
- Document business logic and constraints
- Maintain API documentation
- Record important architectural decisions

This document will evolve as best practices for using erm continue to emerge and mature.