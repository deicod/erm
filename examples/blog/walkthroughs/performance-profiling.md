# Performance Profiling Walkthrough

Use this guide to reason about query composition and instrumentation for the workspace timeline view.

## 1. Rebuild Generated Code

```bash
erm gen --config examples/blog/erm.yaml
```

This ensures ORM query builders include the latest predicates and eager-loading hints.

## 2. Assert Query Guards

```bash
go test ./examples/blog -run PostQuerySpecDefaults
```

The test inspects `Post.Query()` to verify that workspace predicates, author predicates, and limit ceilings are still in place.
This keeps resolver authors from accidentally issuing unbounded queries.

## 3. Capture Query Plans

Run the timeline query through psql to collect a baseline plan:

```bash
psql "$ERM_DATABASE_URL" <<'SQL'
EXPLAIN ANALYZE
SELECT id, author_id, workspace_id, created_at
FROM posts
WHERE workspace_id = '00000000-0000-0000-0000-000000000001'
ORDER BY created_at DESC
LIMIT 20;
SQL
```

Store the output alongside flamegraphs so you can diff regressions after future releases.

## 4. Profile Dataloader Batches

Enable verbose observability logging while executing the query via GraphQL (or the provided sandbox harness):

```bash
ERM_OBSERVABILITY_DEBUG=1 go test ./examples/blog -run TestUserORMCRUDFlow -v
```

The sandbox emits batched SQL calls, letting you confirm that comment replies are fetched via dataloaders instead of repeating
per-node queries. Adapt the harness to replay production payloads when necessary.
