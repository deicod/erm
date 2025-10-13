# Schema Development Workflow

When touching files in this directory:

1. Practice TDD — write or update tests alongside schema changes (see `generator` and example apps).
2. Run `gofmt -w` on edited schema files before committing.
3. Validate the project with:
   - `go test ./...`
   - `go test -race ./...`
   - `go vet ./...`
   (See [`docs/testing.md`](../docs/testing.md#race-detector-workflow) for batching tips, expected runtimes, and the `erm test --race` helper.)
4. Regenerate code with `erm gen` when the schema shape changes and review the diff before committing.
