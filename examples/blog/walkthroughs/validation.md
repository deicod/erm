# Validation Walkthrough

This runbook validates the editorial workspace schema before it reaches production. Each step is executable from the repository
root and maps back to tests under `examples/blog`.

## 1. Regenerate Artifacts

```bash
erm gen --config examples/blog/erm.yaml
```

The command above regenerates ORM packages and migrations using the example schema configuration. Swap the config path for your
project when adapting the walkthrough.

## 2. Inspect Migrations

```bash
ls -1 migrations | tail -n 3
psql "$ERM_DATABASE_URL" -c '\d memberships'
```

Confirm the composite unique index on `(workspace_id, user_id)` exists and that `slug` on `workspaces` is unique.

## 3. Run Schema Assertions

```bash
go test ./examples/blog -run "WorkspaceSlugConstraints|MembershipCompositeIndex"
```

These tests inspect the DSL directly to verify that validators, uniqueness flags, and composite indexes remain intact after
refactors. Keep them in CI so schema drift is caught before migrations ship.

## 4. Gate GraphQL Changes

Before merging GraphQL changes, run the existing CRUD smoke test to ensure base entities still round-trip with mocks:

```bash
go test ./examples/blog -run TestUserORMCRUDFlow
```

Combine these checks with your application-specific tests to maintain confidence as the relationship graph grows.
