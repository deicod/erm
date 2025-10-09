# Best Practices

Follow these conventions to keep generated and handwritten code aligned, reduce merge conflicts, and make life easier for humans
and AI collaborators alike.

---

## Workflow Hygiene

1. **Edit Schema First** – Always modify the DSL before touching generated files. Regenerate immediately after editing.
2. **Small Commits** – Keep schema, migrations, and resolver changes in the same commit so reviewers can reason about the impact.
3. **Run `erm gen` Before Commit** – Prevent CI failures by regenerating code locally.
4. **Update Documentation** – When you introduce new patterns, update the docs under `docs/` and reference the change in PRs.
5. **Include Tests** – Add or update tests (`internal/testutil`, GraphQL tests) whenever business logic changes.

---

## Schema Design

- Use mixins for timestamps, soft deletes, and multi-tenancy to avoid duplication.
- Prefer UUIDv7 primary keys for all entities; the generator handles ordering and pagination correctly.
- Add comments to fields and edges—these surface in database comments, GraphQL descriptions, and generated code.
- Keep privacy rules explicit. Start with restrictive defaults (`Deny`) and open up as needed.
- Use annotations to shape GraphQL output instead of editing resolvers manually.

---

## GraphQL Layer

- Use connection fields (`edges`, `pageInfo`) even for small lists to stay Relay-compliant.
- Expose computed fields via annotations rather than editing generated files.
- Keep resolver customizations in `_extension.go` files next to generated resolvers.
- Validate new schema additions with the GraphQL client and update API docs when queries change.

---

## CLI & Automation

- Add `//go:generate erm gen` to schema packages so `go generate ./...` keeps code fresh.
- Integrate `erm gen --dry-run` into CI pipelines to detect drift early.
- Run `erm doctor` once it exits preview to enforce migration ordering and configuration sanity.

---

## Database & Migrations

- Review generated SQL before applying it. Adjust indexes and constraints if the default naming does not suit your domain.
- Tag migrations with descriptive names using `erm gen --name` for easier auditing.
- Apply migrations in a separate deployment step to detect failures before rolling out code.
- Keep a clean migration history—avoid editing past migrations once applied.

---

## Security

- Verify that `oidc` configuration in `erm.yaml` matches each environment (issuer, audience, scopes).
- Store secrets (DSN, client IDs) in environment variables and reference them in `erm.yaml` using `${ENV_VAR}` syntax.
- Audit custom claims mappers periodically to ensure roles and permissions align with organizational policy.
- Use privacy rules to protect sensitive edges even when resolvers bypass GraphQL directives.

---

## Observability

- Turn on tracing and metrics in staging environments before production to capture baselines.
- Use log redaction features (`.Sensitive()`) to avoid leaking secrets in structured logs.
- Monitor dataloader metrics (`erm_dataloader_batch_size`) to detect regressions when adding new edges.

---

## Collaboration with AI Tools

- Provide schema snippets when prompting AI assistants so they honor field names and annotations.
- Ask AI to generate `_extension.go` files rather than editing generated files.
- Encourage AI to update docs/tests alongside code changes—include references to sections in this portal within prompts.

---

## Pull Request Checklist

Before requesting review:

- [ ] `erm gen` run with no pending changes.
- [ ] SQL migrations reviewed and tested locally.
- [ ] Tests added or updated (`go test ./...`).
- [ ] Documentation updated (`docs/` or README notes).
- [ ] Screenshots or GraphQL query examples attached when API changes.

Following these practices keeps erm projects predictable and maintainable.
