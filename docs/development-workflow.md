# Development workflow

This project relies on deterministic code generation. The generator now fingerprints the schema inputs and caches the last known signature for each component (ORM, GraphQL, migrations). When you re-run `erm gen` only the components whose input hash changed are updated; everything else is either skipped or, in watch mode, emitted to a staging directory for review.

## Everyday iteration

1. Edit your `*.schema.go` files.
2. Run `erm gen` from the project root.
   - The command updates only the impacted artifacts. Up-to-date components are reported as "up-to-date" and left untouched.
   - Migrations are generated only when the schema snapshot changes. Re-running without changes keeps the migration directory clean.
3. Commit the regenerated files as usual.

Because the generator tracks hashes, you can safely run `erm gen` as often as you like—the second run after a change should report every component as "up-to-date". The tests under `generator/run_test.go` exercise this flow against the `examples/blog` workspace to guarantee idempotency.

## Watching schema changes

Use the new watch mode while iterating on schemas:

```bash
erm gen --watch
```

Watch mode keeps the project artifacts current and writes preview copies of unchanged components to `.erm/staging/<component>/…` so you can diff the proposed output without touching the working tree. Delete the staging directory whenever you want a clean slate:

```bash
rm -rf .erm/staging
```

## Previewing migrations

When you need to inspect upcoming database changes, combine `--dry-run` with `--diff`:

```bash
erm gen --dry-run --diff
```

This prints the SQL that would be generated and a concise diff summary. Because dry runs never update the generator cache, a subsequent real run still produces the correct migrations.

## Recommended checklist

- Run `erm gen` after every schema edit.
- If a component reports "up-to-date" unexpectedly, clean `.erm/cache/generator_state.json` and regenerate.
- Use `erm gen --watch` during heavy schema work to get immediate feedback in `.erm/staging` without touching tracked files.
- Rely on the `examples/blog` workspace (see `generator/run_test.go`) to verify generator idempotency before major changes.
