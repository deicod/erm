# Error Handling Walkthrough

Follow this sequence when a production incident hits the editorial workspace stack.

## 1. Capture the Failing Request

```bash
# Replace with your replay tooling; `erm gql replay` is shown as an example.
erm gql replay --config examples/blog/erm.yaml --request-id "$REQUEST_ID"
```

Replaying the request against a sandbox ensures dataloaders and privacy policies execute exactly as they did in production.

## 2. Validate Relationship Assumptions

```bash
go test ./examples/blog -run CommentThreadingEdges
```

This assertion guarantees that comment threading edges remain optional for orphaned comments and that reply edges still target
the correct parent reference.

## 3. Reproduce the SQL Failure

Use pgxmock or a staging database to simulate the offending mutation:

```bash
ERM_OBSERVABILITY_DEBUG=1 go test ./examples/blog -run TestUserORMCRUDFlow -v
```

Adjust expectations in `orm_test.go` to mimic the failing error (duplicate key, foreign key violation, etc.) and confirm the
application surfaces actionable error messages.

## 4. Document the Fix

Update [`examples/blog/README.md`](../README.md) with a short postmortem: root cause, schema changes, and observability signals
that confirmed the remediation. Future responders can start with the documented checklist instead of rediscovering the steps.
