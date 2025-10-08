# Performance and Observability

The erm framework includes built-in performance optimizations and observability features to help you build, monitor, and maintain high-performance GraphQL APIs with PostgreSQL backends.

## Performance Features

### Dataloader Integration

Dataloader is automatically integrated to prevent N+1 query problems:

```go
// Generated dataloaders for each entity
type Loaders struct {
    UserLoader    *UserLoader
    PostLoader    *PostLoader
    CommentLoader *CommentLoader
    // ... other loaders
}

// In resolvers, data is batched automatically
func (r *Resolver) User(ctx context.Context, obj *model.Post) (*model.User, error) {
    // This is automatically batched with other user loads
    return r.Loaders.UserLoader.Load(ctx, obj.UserID)
}
```

#### How It Works
- Batches multiple requests to the same data type
- Caches results within a single request context
- Eliminates N+1 queries from nested resolvers

### Connection-Based Pagination

Relay-compliant pagination avoids loading large datasets:

```go
// Fetch data in chunks with cursors
func (r *Resolver) Users(ctx context.Context, first *int, after *string) (*model.UserConnection, error) {
    // Efficiently fetch only the requested page
    return r.UserService.Connection(ctx, first, after)
}
```

### Query Optimization

The generated queries include optimizations:

1. **Index Generation**: Schema definitions automatically generate appropriate indexes
2. **Query Planning**: Efficient query structure based on requested fields
3. **Connection Pooling**: Configurable pgx connection pooling
4. **Statement Caching**: Prepared statement caching in pgx

### Schema-Based Optimizations

Based on your schema, the generator creates:

- **Appropriate indexes** for fields used in filters/lookups
- **Optimized joins** for relationship queries
- **Batch operations** for bulk data access
- **Eager loading** for common access patterns

## Connection Pooling and Database Config

### pgx Configuration

The framework uses `jackc/pgx/v5` with optimized defaults:

```yaml
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "password"
  name: "myapp"
  # Connection pooling
  max_connections: 20
  min_connections: 5
  max_conn_lifetime: "1h"
  max_conn_idle_time: "30m"
  health_check_period: "1m"
  # Timeouts
  timeout: "10s"
  connect_timeout: "5s"
  # Statement caching
  statement_cache_size: 128
  description_cache_size: 64
```

### Performance Tuning

#### Connection Pool Settings
- `max_connections`: Adjust based on your application's concurrent load
- `min_connections`: Keep frequently used connections warm
- `max_conn_lifetime`: Prevent long-running connections from accumulating state
- `max_conn_idle_time`: Free up resources when not needed

#### Statement Cache
- Cache prepared statements for frequently executed queries
- Reduce parsing overhead on PostgreSQL
- Tune based on query diversity in your application

## N+1 Detection

The framework includes N+1 query detection:

```go
// When N+1 is detected, it logs a warning:
// "N+1 query detected: loading 10 posts resulted in 10 individual user queries"
```

### Best Practices for Avoiding N+1
1. Use dataloaders for relationship resolution
2. Implement field-level batching in resolvers
3. Use eager loading for common query patterns
4. Optimize with connection-based pagination

## OpenTelemetry Integration

The framework includes OpenTelemetry integration for distributed tracing:

### Tracing Configuration

```yaml
telemetry:
  enabled: true
  endpoint: "http://localhost:4317"  # OTLP gRPC endpoint
  service_name: "myapp-graphql"
  sample_rate: 1.0  # 1.0 = all requests, 0.1 = 10% of requests
  attributes:
    deployment.environment: "production"
    service.version: "v1.0.0"
```

### Traced Operations

The framework automatically traces:

1. **GraphQL Operations**
   - Query execution time
   - Field resolver execution
   - Total request time

2. **Database Operations**
   - Query execution time
   - Connection pool utilization
   - Transaction boundaries

3. **Authentication**
   - OIDC token validation
   - Claims mapping time
   - Context injection

### Metrics

Built-in metrics collection:

```go
// Available metrics
"graphql_request_count" // Total requests
"graphql_request_duration" // Request duration histogram
"graphql_field_resolver_duration" // Per-field timing
"database_query_count" // Database queries per request
"database_query_duration" // Query timing
"pool_connections_used" // Connection pool metrics
"cache_hit_ratio" // Cache efficiency
```

## Query Complexity Analysis

### Complexity Limits

Configure query complexity to prevent expensive queries:

```yaml
graphql:
  complexity:
    max_depth: 10          # Maximum query depth
    max_complexity: 1000   # Maximum calculated complexity
    field_cost: 1          # Base cost per field
    connection_cost: 5     # Cost for connection fields
    edge_cost: 2           # Cost per edge
```

### Complexity Calculation

The system calculates complexity based on:
- Query depth
- Number of fields requested
- Connection/page size
- Nested fragment spreads

## Caching Strategies

### Response Caching

For read-heavy applications, implement response caching:

```go
// In your resolvers
func (r *Resolver) PublicData(ctx context.Context) (*model.Data, error) {
    cacheKey := fmt.Sprintf("public_data_%s", r.getVersion())
    
    if cached, ok := r.Cache.Get(cacheKey); ok {
        return cached.(*model.Data), nil
    }
    
    data, err := r.DataService.GetAll(ctx)
    if err != nil {
        return nil, err
    }
    
    r.Cache.Set(cacheKey, data, 5*time.Minute)
    return data, nil
}
```

### Data Caching

The framework provides caching hooks in generated code:

```go
// Generated methods support caching
func (c *Client) UserCaching(enabled bool) *UserQuery {
    if !enabled {
        return c.User.Query()
    }
    // Use configured cache for subsequent queries
    return c.User.CachedQuery()
}
```

## Vector and Geospatial Performance

### pgvector Optimizations

For vector similarity searches:

```yaml
extensions:
  pgvector:
    index_type: "hnsw"      # Options: hnsw, ivfflat, ivfpq
    vectors_per_index: 1000 # For ivfflat - number of vectors per list
    m: 16                   # For HNSW - number of connections per layer
    ef_construction: 64     # For HNSW - construction parameter
```

### PostGIS Spatial Indexes

For geographic queries:

```go
func (Location) Indexes() []dsl.Index {
    return []dsl.Index{
        dsl.Index().Spatial().Using("GIST").Fields("coordinates"),
        dsl.Index().Spatial().Using("SPGIST").Fields("boundary"),
    }
}
```

## Benchmarking and Testing

### Built-in Benchmarks

The framework includes benchmark utilities:

```bash
# Run performance benchmarks
go test -bench=.

# Benchmark GraphQL query performance
erm bench query --file ./benchmark/queries.graphql

# Benchmark connection performance
erm bench connection --concurrency 10 --requests 1000
```

### Performance Testing

Example performance test:

```go
func BenchmarkUserQuery(b *testing.B) {
    client := setupTestClient()
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := client.User.Query().First(ctx)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Monitoring and Alerting

### Health Checks

Built-in health check endpoint:

```go
// Available at /health
{
    "status": "healthy",
    "checks": {
        "database": "healthy",
        "oidc": "healthy", 
        "memory": "healthy",
        "disk": "healthy"
    },
    "timestamp": "2024-01-01T10:00:00Z"
}
```

### Performance Indicators

Monitor these key metrics:

1. **GraphQL Request Duration**
   - P95 latency should be < 100ms for simple queries
   - P99 latency should be < 500ms for complex queries

2. **Database Query Performance**
   - Simple queries < 10ms
   - Complex paginated queries < 50ms
   - Vector similarity searches < 100ms

3. **Connection Pool Utilization**
   - Average connections used < 80% of max
   - No connection timeouts
   - Healthy connection churn rate

### Alerting Rules

Example alerting rules:

```yaml
alerts:
  - name: "high_graphql_latency"
    condition: "graphql_request_duration_p95 > 500ms"
    description: "95th percentile GraphQL requests exceed 500ms"
    
  - name: "nplus1_detected"
    condition: "nplus1_events > 0"
    description: "N+1 query patterns detected"
    
  - name: "cache_miss_ratio"
    condition: "cache_hit_ratio < 0.9"
    description: "Cache hit ratio below 90%"
    
  - name: "connection_pool_exhausted"
    condition: "pool_connections_used > 0.9 * max_connections"
    description: "Connection pool utilization above 90%"
```

## Performance Best Practices

### Schema Design
- Use appropriate indexes for frequently queried fields
- Design efficient relationship patterns
- Consider denormalization for read-heavy use cases

### Query Design
- Use connection-based pagination for lists
- Fetch only required fields (GraphQL benefit)
- Use bulk operations for multiple entities

### Database Design
- Optimize for your access patterns
- Use appropriate PostgreSQL extensions
- Monitor and tune vacuum/autovacuum settings

### Caching Strategy
- Implement at multiple levels (response, data, query)
- Use cache invalidation strategies
- Monitor cache efficiency metrics