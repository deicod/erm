# Command Line Interface (CLI)

The `erm` command-line interface provides a complete toolkit for scaffolding, generating, and managing erm-based projects. It implements an opinionated workflow that enables fast development and consistent project structure.

## Installation

### Installing the CLI

```bash
# Install the latest version
go install github.com/deicod/erm/cmd/erm@latest

# Or install a specific version
go install github.com/deicod/erm/cmd/erm@v0.1.0
```

### Verifying Installation

```bash
erm version
erm help
```

## Commands

### `erm init` - Initialize Project

Creates the basic project structure and configuration files:

```bash
erm init
```

This command:
- Creates project directory structure
- Generates `erm.yaml` configuration
- Sets up initial schema directory
- Creates basic database and GraphQL configuration
- Initializes the project's `go.mod` dependencies

#### Options
- `--name string` - Project name (defaults to current directory name)
- `--db-type postgres` - Database type (currently only Postgres supported)
- `--with-graphql` - Initialize with GraphQL support
- `--with-oidc` - Include OIDC configuration

#### Example
```bash
erm init --name myapp --with-graphql --with-oidc
```

### `erm new <Entity>` - Create New Schema

Creates a new schema definition file:

```bash
erm new User
```

This creates a `User.schema.go` file in the `schema/` directory with a basic structure.

#### Options
- `--with-mixin name` - Include a specific mixin in the schema
- `--with-view name` - Add a default view to the schema
- `--with-hook type` - Add a specific type of hook

#### Example
```bash
erm new Post --with-mixin AuditMixin
```

### `erm gen` - Generate Code

Generates all code based on schema definitions:

```bash
erm gen
```

This command:
- Scans all `.schema.go` files in the schema directory
- Generates ORM models and queries
- Creates GraphQL types, resolvers, and schema
- Generates database migration files
- Creates privacy policies and hooks
- Updates dependency files if needed

#### Options
- `--dry-run` - Show what would be generated without making changes
- `--force` - Overwrite existing generated files
- `--verbose` - Show detailed generation process
- `--watch` - Watch for schema changes and regenerate automatically

#### Example
```bash
erm gen --verbose
```

### `erm graphql init` - Initialize GraphQL

Sets up GraphQL server and configuration:

```bash
erm graphql init
```

This command:
- Creates GraphQL server files
- Configures gqlgen
- Sets up relay-specific types and connections
- Creates GraphQL playground configuration
- Sets up dataloader integration

#### Options
- `--server-file string` - Specify server file location
- `--schema-file string` - Specify schema file location
- `--resolvers-dir string` - Specify resolvers directory

### `erm migrate` - Handle Migrations

Manages database migrations:

```bash
# Create a new migration
erm migrate create add_users_table

# Apply pending migrations
erm migrate up

# Rollback last migration
erm migrate down

# Show migration status
erm migrate status
```

#### Options
- `--env string` - Environment to use (dev, staging, prod)
- `--dry-run` - Show what would be executed without making changes
- `--version int` - Specific version to migrate to

### `erm serve` - Start Development Server

Starts the application server with hot reloading:

```bash
erm serve
```

#### Options
- `--port int` - Port to run the server on (default: 8080)
- `--host string` - Host to bind to (default: localhost)
- `--watch` - Watch for file changes and restart server
- `--env string` - Environment to run in

#### Example
```bash
erm serve --port 3000 --env development
```

### `erm doctor` - Diagnostic Tool

Provides diagnostic information (when available):

```bash
erm doctor
```

Checks:
- Go version compatibility
- Database connectivity
- Configuration validity
- Generated code consistency

### `erm version` - Show Version

Displays the current version:

```bash
erm version
```

### `erm completion` - Shell Completion

Generates shell completion scripts:

```bash
# For bash
erm completion bash > /etc/bash_completion.d/erm

# For zsh
erm completion zsh > /usr/local/share/zsh/site-functions/_erm
```

## Configuration File (erm.yaml)

The CLI uses an `erm.yaml` configuration file in the project root. This file defines project settings and can be modified to customize behavior.

### Example Configuration

```yaml
# Project configuration
project:
  name: "myapp"
  version: "0.1.0"
  description: "A sample erm application"

# Database configuration
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "password"
  name: "myapp"
  ssl_mode: "disable"
  max_connections: 20

# GraphQL configuration
graphql:
  endpoint: "/graphql"
  playground: true
  introspection: true
  complexity:
    max_depth: 10
    max_complexity: 1000

# OIDC configuration
oidc:
  issuer: "https://your-keycloak/realms/myrealm"
  client_id: "myapp"
  jwks_url: "https://your-keycloak/realms/myrealm/protocol/openid-connect/certs"
  claims_mapper: "keycloak"

# Code generation options
generation:
  output_dir: "./internal/generated"
  templates_dir: "./internal/templates"
  package_name: "generated"
  features:
    - "relay"
    - "dataloader"
    - "privacy"
    - "hooks"
    - "interceptors"

# Extensions
extensions:
  enabled:
    - "postgis"
    - "pgvector"
    - "timescaledb"
```

## Project Structure

The CLI creates and maintains the following project structure:

```
myproject/
├── go.mod
├── go.sum
├── erm.yaml              # Configuration
├── main.go               # Application entry point
├── schema/               # Schema definition files
│   ├── User.schema.go
│   └── Post.schema.go
├── internal/
│   ├── generated/        # Generated code
│   │   ├── models/
│   │   ├── graphql/
│   │   └── db/
│   ├── handlers/         # Custom handlers
│   └── middleware/       # Custom middleware
├── migrations/           # Database migration files
├── cmd/
│   └── server/           # Server command
└── docs/                 # Documentation
```

## Workflow Examples

### Full Development Workflow

1. **Initialize the project**
   ```bash
   erm init --name myproject --with-graphql --with-oidc
   cd myproject
   ```

2. **Define your schema**
   ```bash
   erm new User
   erm new Post
   # Edit the generated schema files to add fields and relationships
   ```

3. **Generate code**
   ```bash
   erm gen
   ```

4. **Set up GraphQL**
   ```bash
   erm graphql init
   ```

5. **Create and run migrations**
   ```bash
   erm migrate create initial_schema
   erm migrate up
   ```

6. **Start development server**
   ```bash
   erm serve
   ```

### Adding New Features

1. **Create new schema**
   ```bash
   erm new Comment
   # Edit Comment.schema.go
   ```

2. **Regenerate code**
   ```bash
   erm gen
   ```

3. **Create migration and apply**
   ```bash
   erm migrate create add_comments_table
   erm migrate up
   ```

## Advanced Usage

### Environment-Specific Configuration

Use different configurations for different environments:

```bash
erm serve --env production
```

The CLI looks for `erm.prod.yaml`, `erm.dev.yaml`, etc., for environment-specific settings.

### Custom Templates

The CLI supports custom code generation templates:

```yaml
generation:
  templates_dir: "./custom-templates"
  template_overrides:
    - "graphql_resolver"
    - "model_methods"
```

### Batch Operations

Perform multiple operations in sequence:

```bash
erm new User && erm new Post && erm gen && erm migrate create initial_schema
```

## Troubleshooting

### Common Issues

1. **Permission Denied when Installing**:
   ```bash
   # Ensure Go bin directory is in PATH
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

2. **Schema Parsing Errors**:
   - Check that schema files follow the DSL format
   - Ensure proper imports: `"github.com/deicod/erm/internal/orm/dsl"`
   - Verify syntax with `go fmt` and `go vet`

3. **Database Connection Issues**:
   - Verify `erm.yaml` database configuration
   - Ensure PostgreSQL is running
   - Check network connectivity to database

4. **Code Generation Failures**:
   - Run with `--verbose` to see detailed error messages
   - Verify schema syntax and completeness
   - Check file permissions in output directories

### Diagnostic Commands

```bash
# Verbose generation to see details
erm gen --verbose

# Dry run to preview changes
erm gen --dry-run

# Check configuration
erm doctor

# Get help for specific commands
erm <command> --help
```