# Testing

The erm framework provides comprehensive testing capabilities to ensure your generated code is reliable, performant, and secure. This includes unit tests, integration tests, and end-to-end testing patterns.

## Testing Architecture

The framework organizes tests at multiple levels:

1. **Unit Tests** - Test individual generated functions and methods
2. **Integration Tests** - Test ORM operations with real database connections  
3. **GraphQL Tests** - Test GraphQL operations and resolvers
4. **End-to-End Tests** - Full API integration testing

## Unit Testing

### Generated Code Tests

The framework generates unit tests for core functionality:

```go
// Example generated unit test
func TestUserModel(t *testing.T) {
    user := &User{
        ID:        uuid.NewUUIDv7(),
        Name:      "Test User",
        Email:     "test@example.com",
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    
    // Test field validation
    assert.NotEmpty(t, user.ID)
    assert.Equal(t, "Test User", user.Name)
    assert.Contains(t, user.Email, "@")
    
    // Test generated methods
    assert.True(t, user.IsPersisted()) // If it has an ID
}
```

### Custom Business Logic Tests

For custom hooks, interceptors, and privacy policies:

```go
func TestUserCreationHook(t *testing.T) {
    hook := NewUserCreationHook()
    
    user := &User{
        Name:  "New User",
        Email: "new@example.com",
    }
    
    ctx := context.Background()
    result, err := hook(ctx, user)
    
    assert.NoError(t, err)
    assert.NotEmpty(t, result.CreatedAt)
    assert.Equal(t, "new@example.com", result.NormalizedEmail)
}
```

## Integration Testing

### Database Integration Tests

Integration tests use real PostgreSQL connections:

```go
func TestUserCRUD(t *testing.T) {
    // Use test database
    client := testhelper.NewTestClient(t)
    ctx := context.Background()
    
    // Create
    user, err := client.User.Create().
        SetName("Test User").
        SetEmail("test@example.com").
        Save(ctx)
    require.NoError(t, err)
    require.NotEmpty(t, user.ID)
    
    // Read
    found, err := client.User.Query().Where(user.ID).First(ctx)
    require.NoError(t, err)
    assert.Equal(t, "Test User", found.Name)
    
    // Update
    updated, err := client.User.UpdateOne(user).
        SetName("Updated User").
        Save(ctx)
    require.NoError(t, err)
    assert.Equal(t, "Updated User", updated.Name)
    
    // Delete
    count, err := client.User.DeleteOne(user).Exec(ctx)
    require.NoError(t, err)
    assert.Equal(t, 1, count)
}
```

### Test Database Setup

The framework provides utilities for test database management:

```go
// In testhelper package
func NewTestClient(t *testing.T) *ent.Client {
    // Create temporary database
    db := testdb.New(t)
    
    // Run migrations on test database
    runner := migrations.NewRunner(db.DB())
    runner.RunAll()
    
    // Create and return client
    client, err := ent.Open("pgx", db.URL())
    require.NoError(t, err)
    
    // Clean up after test
    t.Cleanup(func() {
        client.Close()
        db.Close()
    })
    
    return client
}
```

### Transaction Testing

Test database transactions and rollbacks:

```go
func TestUserTransaction(t *testing.T) {
    client := testhelper.NewTestClient(t)
    ctx := context.Background()
    
    // Test transaction rollback
    err := client.Tx(ctx, func(tx *ent.Tx) error {
        // Create user in transaction
        user, err := tx.User.Create().
            SetName("Transaction User").
            SetEmail("trans@example.com").
            Save(ctx)
        require.NoError(t, err)
        
        // Simulate error to trigger rollback
        return errors.New("simulate error")
    })
    
    assert.Error(t, err)
    
    // Verify user was not created
    _, err = client.User.Query().
        Where(user.Email("trans@example.com")).
        First(ctx)
    assert.True(t, ent.IsNotFound(err))
}
```

## GraphQL Testing

### Operation Testing

Test GraphQL operations with a test server:

```go
func TestUserQuery(t *testing.T) {
    ts := httptest.NewServer(testhelper.NewGraphQLHandler(t))
    defer ts.Close()
    
    query := `
        query GetUser($id: ID!) {
            user(id: $id) {
                id
                name
                email
            }
        }
    `
    
    variables := map[string]interface{}{
        "id": encodeGlobalID("User", "test-uuid"),
    }
    
    resp := graphqlTest(t, ts.URL, query, variables)
    
    user := resp.Data["user"].(map[string]interface{})
    assert.Equal(t, "Test User", user["name"])
    assert.Equal(t, "test@example.com", user["email"])
}

// Helper for GraphQL testing
func graphqlTest(t *testing.T, url, query string, variables map[string]interface{}) *graphql.Response {
    req := graphql.NewRequest(query)
    for k, v := range variables {
        req.Var(k, v)
    }
    
    var resp graphql.Response
    err := req.Do(context.Background(), url, &resp)
    require.NoError(t, err)
    
    return &resp
}
```

### Authentication Testing

Test OIDC authentication flows:

```go
func TestAuthenticatedQuery(t *testing.T) {
    ts := httptest.NewServer(testhelper.NewGraphQLHandler(t))
    defer ts.Close()
    
    // Create mock JWT token
    token := testhelper.NewMockJWT(map[string]interface{}{
        "sub":   "test-user-id",
        "email": "user@example.com",
        "roles": []string{"user"},
    })
    
    query := `query { me { id name email } }`
    
    req, _ := http.NewRequest("POST", ts.URL, strings.NewReader(`{
        "query": "` + query + `"
    }`))
    
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{}
    resp, err := client.Do(req)
    require.NoError(t, err)
    defer resp.Body.Close()
    
    // Verify response
    body, _ := io.ReadAll(resp.Body)
    assert.Contains(t, string(body), "test-user-id")
}
```

## Migration Testing

### Migration Validation Tests

Test that migrations apply cleanly:

```go
func TestMigrations(t *testing.T) {
    db := testdb.New(t)
    runner := migrations.NewRunner(db.DB())
    
    // Test that all migrations can be applied
    err := runner.RunAll()
    assert.NoError(t, err)
    
    // Test that database schema matches expectations
    inspector := pgconn.NewInspector(db.DB())
    assert.True(t, inspector.HasTable("users"))
    assert.True(t, inspector.HasColumn("users", "email"))
    assert.True(t, inspector.HasIndex("users", "users_email_key"))
}

func TestMigrationRollback(t *testing.T) {
    db := testdb.New(t)
    runner := migrations.NewRunner(db.DB())
    
    // Apply all migrations
    runner.RunAll()
    
    // Get initial state
    inspector := pgconn.NewInspector(db.DB())
    initialTables := inspector.TableNames()
    
    // Rollback all migrations
    err := runner.RollbackAll()
    assert.NoError(t, err)
    
    // Verify clean state
    finalTables := inspector.TableNames()
    assert.Equal(t, 0, len(finalTables)) // Should be empty
}
```

## Performance Testing

### Benchmark Tests

Include performance benchmarks for common operations:

```go
func BenchmarkUserQuery(b *testing.B) {
    client := testhelper.NewTestClient(b)
    ctx := context.Background()
    
    // Setup test data
    for i := 0; i < 1000; i++ {
        client.User.Create().
            SetName(fmt.Sprintf("User %d", i)).
            SetEmail(fmt.Sprintf("user%d@example.com", i)).
            ExecX(ctx)
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := client.User.Query().Limit(10).All(ctx)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkConnectionQuery(b *testing.B) {
    client := testhelper.NewTestClient(b)
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := client.User.Query().Paginate(ctx, &model.Pagination{
            First: ptr.Int(20),
        })
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Load Testing Configuration

Use the CLI for load testing:

```bash
# Run load test with 100 concurrent users
erm test load --concurrency 100 --requests 10000

# Test specific GraphQL operations
erm test load --file ./test-queries.graphql --duration 5m

# Generate test data
erm test generate --entity User --count 10000
```

## Extension Testing

### PostGIS Testing

Test geographic operations:

```go
func TestGeospatialQuery(t *testing.T) {
    client := testhelper.NewTestClient(t)
    ctx := context.Background()
    
    // Create location
    location, err := client.Location.Create().
        SetCoordinates(dsl.Point{X: -122.4194, Y: 37.7749}). // San Francisco
        Save(ctx)
    require.NoError(t, err)
    
    // Test distance query
    nearby, err := client.Location.Query().
        Where(dsl.Location.DistanceWithin("coordinates", 
            dsl.Point{X: -122.4194, Y: 37.7749}, 1000)). // Within 1km
        All(ctx)
    
    require.NoError(t, err)
    assert.Contains(t, nearby, location)
}
```

### pgvector Testing

Test vector similarity operations:

```go
func TestVectorSimilarity(t *testing.T) {
    client := testhelper.NewTestClient(t)
    ctx := context.Background()
    
    queryVector := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
    
    // Create similar embeddings
    embedding1, err := client.Embedding.Create().
        SetEmbedding([]float32{0.11, 0.21, 0.31, 0.41, 0.51}).
        Save(ctx)
    require.NoError(t, err)
    
    embedding2, err := client.Embedding.Create().
        SetEmbedding([]float32{0.8, 0.9, 0.7, 0.6, 0.5}).
        Save(ctx)
    require.NoError(t, err)
    
    // Test similarity search
    similar, err := client.Embedding.Query().
        Where(dsl.Embedding.CosineDistance("embedding", queryVector, 0.2)).
        OrderBy(dsl.Embedding.CosineDistanceTo("embedding", queryVector)).
        All(ctx)
    
    require.NoError(t, err)
    assert.Contains(t, similar, embedding1)
    assert.NotContains(t, similar, embedding2) // Less similar
}
```

## Testing Utilities

### Test Fixtures

The framework provides utilities for creating test data:

```go
// testdata package
type Fixtures struct {
    Users    []*User
    Posts    []*Post
    Comments []*Comment
}

func LoadFixtures(t *testing.T, client *ent.Client) *Fixtures {
    ctx := context.Background()
    
    // Create users
    user1, err := client.User.Create().
        SetName("Alice").
        SetEmail("alice@example.com").
        Save(ctx)
    require.NoError(t, err)
    
    user2, err := client.User.Create().
        SetName("Bob").
        SetEmail("bob@example.com").
        Save(ctx)
    require.NoError(t, err)
    
    // Create posts
    post1, err := client.Post.Create().
        SetTitle("First Post").
        SetContent("Content here").
        SetUserID(user1.ID).
        Save(ctx)
    require.NoError(t, err)
    
    return &Fixtures{
        Users: []*User{user1, user2},
        Posts: []*Post{post1},
    }
}
```

### Mock Services

For testing with external services:

```go
type MockOIDCProvider struct {
    *httptest.Server
    tokens map[string]*oidc.IDToken
}

func NewMockOIDCProvider(t *testing.T) *MockOIDCProvider {
    provider := &MockOIDCProvider{
        tokens: make(map[string]*oidc.IDToken),
    }
    
    provider.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch r.URL.Path {
        case "/.well-known/openid_configuration":
            json.NewEncoder(w).Encode(map[string]interface{}{
                "issuer":  provider.URL,
                "jwks_uri": provider.URL + "/jwks",
            })
        case "/jwks":
            // Return test JWKS
            jwks := jose.JSONWebKeySet{
                Keys: []jose.JSONWebKey{testJWK},
            }
            json.NewEncoder(w).Encode(jwks)
        }
    }))
    
    return provider
}
```

## Test Configuration

### Environment Variables

Use environment variables for test configuration:

```bash
# Test database configuration
TEST_DB_HOST=localhost
TEST_DB_PORT=5432
TEST_DB_USER=postgres
TEST_DB_PASSWORD=testpass
TEST_DB_NAME=erm_test

# OIDC test configuration
TEST_OIDC_ISSUER=http://localhost:8080/realms/test
TEST_OIDC_CLIENT_ID=test-client
```

### Test Configuration File

Example `test.yaml` configuration:

```yaml
database:
  host: ${TEST_DB_HOST:-localhost}
  port: ${TEST_DB_PORT:-5432}
  user: ${TEST_DB_USER:-postgres}
  password: ${TEST_DB_PASSWORD:-testpass}
  name: ${TEST_DB_NAME:-erm_test}
  ssl_mode: disable

oidc:
  issuer: ${TEST_OIDC_ISSUER:-http://localhost:8080/realms/test}
  client_id: ${TEST_OIDC_CLIENT_ID:-test-client}
  jwks_url: ${TEST_OIDC_ISSUER:-http://localhost:8080/realms/test}/protocol/openid-connect/certs

testing:
  parallel: true
  timeout: 30s
  fixtures: true
  cleanup: true
```

## Running Tests

### Standard Go Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test
go test -run TestUserCRUD ./internal/...

# Run benchmarks
go test -bench=. ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Using Makefile

```bash
# Run all tests
make test

# Run with race detection
make test-race

# Generate coverage
make coverage

# Run specific package tests
make test-package PKG=./internal/graphql
```

## Continuous Integration

### GitHub Actions Example

Example workflow for testing:

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v4
      with:
        go-version: '1.22'
        
    - name: Install dependencies
      run: go mod tidy
      
    - name: Run tests
      run: go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
      
    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
```

## Testing Best Practices

### Test Organization
- Place unit tests in the same directory as source files
- Integration tests in a separate `integration` subdirectory
- Use descriptive test names
- Follow the 3A pattern: Arrange, Act, Assert

### Test Data Management
- Use factories for creating test objects
- Clean up test data after tests
- Use transactions or savepoints for rollback
- Isolate test data between tests

### Performance Considerations
- Use in-memory databases for unit tests
- Reset database state between integration tests
- Mock external services when possible
- Parallelize tests when safe to do so

### Security Testing
- Test authentication bypass scenarios
- Verify privacy policy enforcement
- Test SQL injection prevention
- Validate input sanitization