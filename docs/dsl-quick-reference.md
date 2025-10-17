# Schema DSL Quick Reference

Keep this cheat sheet open while editing files under `schema/*.schema.go`. It lists the constructors and modifiers the `erm` generator understands so you can reach for the right helper without scanning package docs.

## Field Constructors

### Core primitives

| Helper | Postgres type | Notes |
| ------ | ------------- | ----- |
| `dsl.String("name")`, `dsl.Text("body")` | `TEXT` | Aliases that default the Go type to `string`.
| `dsl.Int("count")`, `dsl.Integer("count")` | `INTEGER` | Use `dsl.SmallInt`/`dsl.BigInt` for narrower or wider storage.
| `dsl.Float("price")`, `dsl.Real`, `dsl.DoublePrecision` | `DOUBLE PRECISION` by default | Pair with `.Precision()` / `.Scale()` when you need decimals.
| `dsl.Boolean("active")` / `dsl.Bool` | `BOOLEAN` | Convenience alias for booleans.
| `dsl.UUID("id")`, `dsl.UUIDv7("id")` | `UUID` | `UUIDv7` seeds the generator with monotonic IDs.
| `dsl.Bytes("blob")`, `dsl.Bytea("payload")` | `BYTEA` | Use for opaque binary content.
| `dsl.Date`, `dsl.Time`, `dsl.Timestamp`, `dsl.TimestampTZ` | Native date/time types | All default to `time.Time` in Go.
| `dsl.JSON`, `dsl.JSONB` | `JSON` / `JSONB` | Stored as JSON with Go type `map[string]any` unless overridden.
| `dsl.Enum("status", "draft", "published")` | `TEXT` + check constraint | The generator emits enum metadata for GraphQL.

### Advanced types

| Helper | Postgres type | Notes |
| ------ | ------------- | ----- |
| `dsl.Decimal("amount", precision, scale)` / `dsl.Numeric` | `DECIMAL` / `NUMERIC` | Remember to specify both precision and scale.
| `dsl.Money`, `dsl.Interval` | `MONEY`, `INTERVAL` | Useful for finance and scheduling use cases.
| `dsl.Inet`, `dsl.CIDR`, `dsl.MACAddr`, `dsl.MACAddr8` | Network primitives | Stored as strings in Go.
| `dsl.Bit`, `dsl.VarBit` | Bit string types | Provide the bit length as the second argument.
| `dsl.Array("tags", dsl.TypeText)` | Array wrapper | Supply a field type constant via `dsl.Type*` helpers.
| `dsl.Geometry`, `dsl.Geography`, `dsl.Vector` | PostGIS / pgvector | Ensure the corresponding extension flag is enabled in `erm.yaml`.
| Range helpers (`dsl.Int4Range`, `dsl.TSRange`, …) | Range types | Use for temporal windows or numeric intervals.

### Field type constants

Use `dsl.Type*` constants when a modifier expects a field type (for example `ArrayElement`). Common values include `dsl.TypeText`, `dsl.TypeUUID`, `dsl.TypeInteger`, `dsl.TypeTimestampTZ`, and `dsl.TypeJSONB`. This cheat sheet ships with `erm init`; peek at the generator tests for the exhaustive list.

## Field Modifiers

| Modifier | Description |
| -------- | ----------- |
| `.Primary()` | Marks a field as the primary key. Automatically applied to the `id` field when omitted.
| `.Optional()` | Makes the column nullable.
| `.Unique()` / `.UniqueConstraint()` | Adds a uniqueness constraint.
| `.ColumnName("override")` | Overrides the generated column name.
| `.DefaultNow()` / `.UpdateNow()` | Populate timestamps automatically using database defaults.
| `.Default(value)` | Use a literal default. Accepts native Go values.
| `.WithDefault("sql expression")` | Stores a raw SQL default expression.
| `.WithGoType("package.Type")` | Overrides the Go type used in generated code.
| `.Identity(dsl.IdentityAlways)` | Switches to Postgres identity columns for serial fields.
| `.Length(n)`, `.Precision(p)`, `.Scale(s)` | Configure string length or numeric precision.
| `.ArrayElement(dsl.TypeUUID)` | Declares the element type for arrays.
| `.Computed(columnSpec)` | Attach a `dsl.Computed` expression to materialised views or generated columns.
| `.SRID(4326)` / `.TimeSeries()` | Spatial and TimescaleDB annotations.

## Edge Builders

| Helper | Relationship | Notes |
| ------ | ------------- | ----- |
| `dsl.ToOne("user", "User")` | Many-to-one | Generates a nullable foreign key column by default.
| `dsl.ToMany("posts", "Post")` | One-to-many | Synthesises inverse edges automatically.
| `dsl.ManyToMany("tags", "Tag")` | Many-to-many | Requires a join table; use `.ThroughTable()` to name it.
| `dsl.PolymorphicTarget("target", "TargetType")` | Polymorphic references | Supply alongside `.Polymorphic()` on the owning edge.

### Edge Modifiers

| Modifier | Description |
| -------- | ----------- |
| `.Field("column")` / `.Ref("column")` | Override forward or reverse column names.
| `.ThroughTable("table")` | Explicit join table for many-to-many edges.
| `.Optional()` | Allows null foreign keys on to-one edges.
| `.UniqueEdge()` | Enforces a unique relationship (one-to-one).
| `.Inverse("backref")` | Declares the reverse edge name.
| `.OnDeleteSetNull()` / `.OnDeleteCascade()` / `.OnDeleteRestrict()` / `.OnDeleteNoAction()` | Configure delete behaviour.
| `.OnUpdateCascade()` / `.OnUpdateSetNull()` / `.OnUpdateRestrict()` / `.OnUpdateNoAction()` | Configure update behaviour.
| `.Polymorphic(targets...)` / `.PolymorphicTargets(...)` | Register allowed target entities.

## Index Helpers

| Helper | Purpose |
| ------ | ------- |
| `dsl.Idx("by_email")` | Creates a named index.
| `.On("email", "tenant_id")` | Declares indexed columns.
| `.Unique()` / `.UniqueConstraint()` | Builds a unique index.
| `.WhereClause("status = 'active'")` | Adds partial index predicates.
| `.MethodUsing("gist")` | Specifies the index method (B-Tree, GIN, GiST, etc.).
| `.NullsNotDistinctConstraint()` | Enables Postgres 15 `NULLS NOT DISTINCT` behaviour.

## Query Builders and Aggregates

| Helper | Description |
| ------ | ----------- |
| `dsl.Query()` | Starts a custom query specification.
| `.WithPredicates(preds...)` | Attach reusable filters (`dsl.NewPredicate`).
| `.WithOrders(orders...)` | Register ordering defaults.
| `.WithAggregates(aggs...)` | Provide aggregate selections for analytics endpoints.
| `.WithDefaultLimit(n)` / `.WithMaxLimit(n)` | Configure pagination defaults.
| `dsl.NewPredicate("field", dsl.OpEqual)` | Build predicate descriptors.
| `dsl.OrderBy("field", dsl.SortAsc)` | Declare ordering metadata.
| `dsl.NewAggregate("field", dsl.AggSum)` / `dsl.CountAggregate("field")` | Aggregation helpers.

## GraphQL Integration

| Helper | Description |
| ------ | ----------- |
| `dsl.GraphQL("FieldName", options...)` | Registers an entity in the GraphQL schema. Options include `dsl.GraphQLRelayConnection()` and `dsl.GraphQLDisableMutations()`.
| `dsl.GraphQLSubscriptions(dsl.SubscriptionEventCreate, ...)` | Enables subscription events per entity.
| `dsl.Expression("SELECT ...", deps...)` | Describes SQL snippets referenced by computed columns.
| `dsl.Computed(expr)` | Wraps a computed expression descriptor for reuse in `.Computed()` field modifiers.

## Troubleshooting Tips

- Typos in helper names now produce suggestions. If you see an error such as `Did you mean "String"?`, update the schema accordingly.
- When an array element type is rejected, double-check the `dsl.Type*` constant and refer back to the tables above.
- Run `erm gen --dry-run --diff` to preview schema changes, and add `--force` after dependency upgrades to rewrite generated packages.

For deeper explanations and walkthroughs, see [`docs/schema-definition.md`](./schema-definition.md) and the surrounding guides in this documentation set.
