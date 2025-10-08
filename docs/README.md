# erm Documentation

Welcome to the comprehensive documentation for **erm** - a Go code generation tool that creates production-grade GraphQL backends with integrated ORM capabilities, following the Relay Server Specification and secured by OIDC.

## Table of Contents

### Getting Started
- [Overview](./overview.md) - Introduction to erm and its core concepts
- [Quickstart Guide](./quickstart.md) - Step-by-step tutorial for your first project

### Core Concepts
- [Schema Definition Guide](./schema-definition.md) - Define your data models using the schema DSL
- [GraphQL API](./graphql-api.md) - Relay-compliant GraphQL API features and usage
- [Authentication and Authorization](./authentication.md) - OIDC integration and access control
- [Extensions Support](./extensions.md) - PostGIS, pgvector, and TimescaleDB integration

### Tools and Utilities
- [Command Line Interface (CLI)](./cli.md) - Complete reference for the erm command-line tool
- [Performance and Observability](./performance-observability.md) - Optimizing and monitoring your application

### Development Practices
- [Testing](./testing.md) - Comprehensive testing strategies and utilities

## About erm

The erm framework is designed to accelerate backend development by generating:
- Type-safe ORM models with full CRUD operations
- Relay-compliant GraphQL APIs
- OIDC-secured endpoints
- Database migrations
- Performance optimizations (dataloader, pagination)

The project emphasizes:
- **Developer Experience**: Intuitive CLI and clear generated code
- **AI-Friendly**: Structured documentation and prompts for LLM-assisted development
- **Production Ready**: Built-in observability, security, and performance features
- **Extensible**: Support for PostgreSQL extensions and custom business logic

## Contributing

See the [CONTRIBUTING.md](../CONTRIBUTING.md) file for guidelines on contributing to the erm project.

## Support

For support, bug reports, or feature requests, please open an issue in the GitHub repository.