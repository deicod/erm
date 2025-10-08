# erm Documentation - Overview

## What is erm?

**erm** is a Go code generation tool that creates an opinionated, production-grade GraphQL backend with integrated ORM capabilities. It fully supports the Relay Server Specification and is built for PostgreSQL with security through OIDC (OpenID Connect).

### Key Features

- **Relay-Complete GraphQL**: Implements the full Relay Server Specification including global `Node` IDs, cursor-based connections/edges, `PageInfo`, and opaque cursors
- **Schema-as-Code ORM**: Ent-like schema definition with code generation for fields, edges, indexes, views, mixins, annotations, hooks, interceptors, privacy policies, and more
- **PostgreSQL First-Class**: Built on top of `jackc/pgx` v5 with batteries-included support for PostGIS, pgvector, and TimescaleDB extensions
- **OIDC Security**: Pluggable authentication middleware with default mapping for Keycloak and support for other OIDC providers
- **UUID v7 IDs**: Default UUID v7 generation at the ORM layer for consistent, app-side ID generation

## Project Goals

The project aims to provide a development experience that is both excellent for senior developers and AI-friendly for rapid iteration and parallel agent work. 

### Primary Objectives

1. Generate a complete, production-ready GraphQL backend with minimal setup
2. Provide robust database abstractions with advanced features like eager loading and privacy policies
3. Implement secure authentication with OIDC middleware
4. Deliver excellent developer experience and AI-assisted development capabilities
5. Maintain high performance with features like dataloaders and N+1 query detection

## Architecture

The architecture is designed around several core components:

- **CLI**: Command-line tool (`erm`) with commands like `init`, `new`, `gen`, and `graphql init`
- **ORM Generator**: Creates database models, queries, and migrations from schema definitions
- **GraphQL Generator**: Creates Relay-compliant GraphQL schemas and resolvers 
- **OIDC Middleware**: Handles JWT verification and claim mapping with pluggable providers
- **Extensions Support**: Built-in support for PostGIS, pgvector, and TimescaleDB

## Getting Started

See the [Quickstart Guide](./quickstart.md) for a step-by-step introduction to setting up and using erm.