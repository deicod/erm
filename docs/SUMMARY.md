# erm Documentation Summary

## Overview

This documentation suite provides complete coverage for the **erm** framework—a Go-based GraphQL + ORM generator that delivers
Relay-compliant APIs, Postgres migrations, OIDC security, and DX-focused tooling.

## Documentation Structure

### Core Concepts
1. **[Overview](./overview.md)** – Architecture, request lifecycle, and configuration surfaces.
2. **[Quickstart Guide](./quickstart.md)** – Bootstrap a project from schema to running server.
3. **[Getting Started with the App Skeleton](./getting-started-app.md)** – Explore the files created by `erm init`, development workflows, and how to run the generated server.
4. **[Getting Started with the Schema Skeleton](./getting-started-schema.md)** – Walkthrough of the generated schema template, defaults, and iteration workflow.
5. **[Schema Definition Guide](./schema-definition.md)** – DSL reference with advanced patterns, including the new complex relationship playbook.
6. **[GraphQL API](./graphql-api.md)** – Relay implementation details, resolvers, and extensions.

### Security and Integration
7. **[Authentication & Authorization](./authentication.md)** – OIDC middleware, claims mapping, and directives.
8. **[Extensions Support](./extensions.md)** – PostGIS, pgvector, TimescaleDB, and custom extensions.

### Tools and Operations
9. **[CLI Reference](./cli.md)** – Command usage, workflows, and automation tips.
10. **[Performance & Observability](./performance-observability.md)** – Metrics, tracing, tuning guidance, plus a workspace timeline optimization scenario.
11. **[Testing](./testing.md)** – Unit, integration, GraphQL, and benchmark strategies.
12. **[CI/CD Guidance](./ci-cd.md)** – Pipeline patterns, schema validation, and rollback automation.
13. **[Deployment Playbooks](./deployment.md)** – Environment targeting, secrets management, and rollback procedures.

### Practices and Troubleshooting
14. **[Best Practices](./best-practices.md)** – Conventions for schema, GraphQL, security, and collaboration.
15. **[Troubleshooting](./troubleshooting.md)** – Symptom-based remediation across the stack with a production incident playbook.
16. **[End-to-End Examples](./examples.md)** – Comprehensive feature walkthroughs you can adapt, including the editorial workspace blog series.

### Navigation
17. **[README](./README.md)** – Portal entry point and map of the guides.

## Key Highlights

- **Schema DSL** – Fields, edges, indexes, mixins, hooks, interceptors, privacy policies, and migrations.
- **GraphQL Layer** – Global Node IDs, connections, dataloaders, custom queries/mutations, and testing utilities.
- **Security** – OIDC discovery, JWKS caching, claims mapping, `@auth` directives, and privacy integration.
- **Extensions** – Batteries-included support for PostGIS, pgvector, TimescaleDB, plus guidance for custom extensions.
- **DX & Automation** – CLI workflows, `erm doctor`, documentation updates, AI collaboration tips.

## Project Status

Refer to [ROADMAP.md](../ROADMAP.md) for milestone tracking. Documentation updates accompany feature releases and highlight
migration steps when behavior changes.

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines. When updating docs, include runnable snippets and note CLI versions if
behavior differs between releases.

## License

erm is released under the MIT License. See [LICENSE](../LICENSE) for details.
