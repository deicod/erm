# erm Documentation Summary

## Overview

This comprehensive documentation suite covers the **erm** framework - a Go code generation tool that creates production-grade GraphQL backends with integrated ORM capabilities, following the Relay Server Specification and secured by OIDC.

## Documentation Structure

The documentation includes the following comprehensive guides:

### Core Concepts
1. **[Overview](./overview.md)** - Introduction to erm and its core concepts
2. **[Quickstart Guide](./quickstart.md)** - Step-by-step tutorial for your first project
3. **[Schema Definition Guide](./schema-definition.md)** - Define your data models using the schema DSL
4. **[GraphQL API](./graphql-api.md)** - Relay-compliant GraphQL API features and usage

### Security and Integration
5. **[Authentication and Authorization](./authentication.md)** - OIDC integration and access control
6. **[Extensions Support](./extensions.md)** - PostGIS, pgvector, and TimescaleDB integration

### Tools and Operations
7. **[Command Line Interface (CLI)](./cli.md)** - Complete reference for the erm command-line tool
8. **[Performance and Observability](./performance-observability.md)** - Optimizing and monitoring your application
9. **[Testing](./testing.md)** - Comprehensive testing strategies and utilities

### Best Practices and Support
10. **[Best Practices](./best-practices.md)** - Recommended approaches for using erm effectively
11. **[Troubleshooting](./troubleshooting.md)** - Common issues and their solutions

### Navigation
12. **[README](./README.md)** - Main documentation index and entry point

## Key Features Documented

### Schema Definition
- Schema-as-code DSL similar to Facebook's ent
- Support for various field types and constraints
- Relationships and edge definitions
- Indexes, views, mixins and annotations

### GraphQL API
- Full Relay specification compliance
- Global object IDs
- Connection-based pagination with cursors
- Dataloader integration to prevent N+1 queries
- Query optimization and performance features

### Security
- OIDC authentication with pluggable claim mappers
- Keycloak integration (default)
- Support for Auth0, Okta, and custom OIDC providers
- Privacy policies for fine-grained access control

### Extensions
- PostGIS support for geographic data
- pgvector for AI/ML embedding storage and similarity search
- TimescaleDB for time-series data

### Developer Experience
- Intuitive CLI with init, new, gen, and graphql commands
- Automatic code generation from schema definitions
- Proper database migration handling
- Comprehensive testing utilities

## Project Status

Based on the PRD and ROADMAP documentation found in the codebase, erm is currently in alpha stage with the following milestones planned:

- **Milestone 0**: Repo bootstrap (completed)
- **Milestone 1**: ORM core (in progress)
- **Milestone 2**: GraphQL Relay (in progress)
- **Milestone 3**: OIDC (in progress)
- **Milestone 4**: Extensions & Performance (in progress)
- **Milestone 5**: DX polish and v0.1.0 release (planned)

## Contributing

For contribution guidelines, refer to the [CONTRIBUTING.md](../CONTRIBUTING.md) file in the root directory.

## License

The erm framework is released under the MIT License as described in the [LICENSE](../LICENSE) file.