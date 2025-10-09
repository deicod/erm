# Extensions Support

erm embraces PostgreSQL extensions out of the box. The DSL exposes field constructors, mixins, and migration helpers that let
you adopt specialized storage models without manual SQL. This guide covers the three built-in extension families and how to add
your own.

---

## Enabling Extensions

Declare extensions in schema annotations or via mixins. The generator ensures migrations include `CREATE EXTENSION IF NOT EXISTS`.

```go
func (Location) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.Extension("postgis"),
    }
}
```

You can also enable extensions globally in `erm.yaml`:

```yaml
extensions:
  postgis: true
  pgvector: false
  timescaledb: false
```

Run `erm gen` after toggling flags so migrations stay in sync. Extensions are idempotent; running migrations multiple times is safe.

---

## PostGIS

Use the geometry/geography field helpers to store spatial data.

```go
type Location struct{ dsl.Schema }

func (Location) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.Geometry("geom", 4326).
            SRID(4326).
            Type(dsl.GeometryPoint).
            NotEmpty().
            Comment("WGS84 point"),
        dsl.String("label").Optional(),
    }
}
```

### Generated Features

- SQL migration includes `CREATE EXTENSION IF NOT EXISTS postgis;` and sets SRID metadata.
- ORM builder exposes `SetGeom(point geom.Geometry)` with helper constructors in `internal/orm/pgxext/postgis`.
- GraphQL exposes `GeoJSON` scalars for geometry fields.
- Predicate helpers support spatial operators (`WhereGeomWithin`, `WhereGeomIntersects`).

### Example Query

```go
client.Location.Query().
    Where(location.GeomWithinRadius(lat, lng, 500)).
    Order(location.ByDistance(lat, lng, orm.OrderAsc)).
    All(ctx)
```

### Tips

- Use `.Geography()` for distance calculations across long distances (uses spheroidal math).
- Add indexes with `dsl.Index().Fields("geom").Using("gist")` to accelerate spatial queries.
- Combine with `TimescaleDB` hypertables if storing time-series location updates.

---

## pgvector

Store machine learning embeddings using `dsl.Vector` fields.

```go
type Document struct{ dsl.Schema }

func (Document) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.String("title").NotEmpty(),
        dsl.Vector("embedding", 1536).
            NotEmpty().
            Comment("OpenAI text-embedding-3-large vector"),
    }
}
```

### Similarity Search

```go
query := client.Document.Query().
    Where(document.EmbeddingNearestNeighbor(vec, orm.WithDistanceColumn("similarity"))).
    Limit(20)

results, err := query.All(ctx)
```

GraphQL surfaces a `Vector` scalar that accepts/returns lists of floats. Use annotations to expose specialized queries:

```go
dsl.GraphQL("Document").
    CustomQuery("searchDocuments", dsl.Query{
        Args: []dsl.QueryArg{
            dsl.VectorArg("embedding", 1536),
            dsl.IntArg("limit").Default(10),
        },
        Resolver: `return client.Document.Query().Where(document.EmbeddingNearestNeighbor(args.Embedding, orm.WithDistanceColumn("similarity"))).Limit(args.Limit).All(ctx)`,
    })
```

### Indexing

- `.Index().Fields("embedding").Using("ivfflat").With(`lists = 100`)` – recommended for large datasets.
- Tune `ANALYZE` and `SET enable_seqscan = off` for benchmark runs.

---

## TimescaleDB

TimescaleDB powers time-series workloads. Enable it via annotation:

```go
func (Metric) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.Extension("timescaledb"),
        dsl.Timescale(dsl.TimescaleOptions{
            TimeColumn: "collected_at",
            ChunkInterval: 24 * time.Hour,
            Compressed: true,
        }),
    }
}
```

Define fields:

```go
type Metric struct{ dsl.Schema }

func (Metric) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.TimestampTZ("collected_at").NotEmpty(),
        dsl.String("name").NotEmpty(),
        dsl.Float("value").Required(),
        dsl.JSON("labels").Optional(),
    }
}
```

### Generated Support

- Migration converts the table to a hypertable via `SELECT create_hypertable(...)`.
- Compression policies and retention policies map from annotation options.
- ORM helpers expose `WhereCollectedAtBetween` and `AggregateByInterval` functions.
- GraphQL adds bucketed aggregation queries when `dsl.Timescale().ExposeAggregation()` is set.

### Sample Aggregation Resolver

```go
func (r *metricResolver) Bucketed(ctx context.Context, obj *orm.Metric, args struct {
    Interval time.Duration
}) ([]*model.MetricBucket, error) {
    return r.Client.Metric.Query().
        Where(metric.NameEQ(obj.Name)).
        Aggregate(metric.CollectedAtBucket(args.Interval), metric.ValueAvg()).
        All(ctx)
}
```

---

## Creating Custom Extensions

If you rely on another PostgreSQL extension, create a mixin:

```go
type LTREE struct{ dsl.Mixin }

func (LTREE) Fields() []dsl.Field {
    return []dsl.Field{dsl.String("path").Comment("ltree path")}
}

func (LTREE) Annotations() []dsl.Annotation {
    return []dsl.Annotation{dsl.Extension("ltree")}
}
```

Add the mixin to your schema and run `erm gen`. The migration will include `CREATE EXTENSION IF NOT EXISTS ltree;` automatically.

---

## Operational Considerations

- Ensure your target database has the extension installed or that your user has permission to run `CREATE EXTENSION`.
- In multi-tenant deployments, pre-install extensions during provisioning to avoid race conditions.
- Include extension prerequisites in your infrastructure-as-code (Terraform, Helm) for deterministic environments.

---

## Troubleshooting

| Symptom | Resolution |
|---------|------------|
| Migration fails with “permission denied” | Run migrations using a role with `CREATE` privileges or pre-install the extension. |
| GraphQL rejects vector input | Ensure the list length matches the dimension specified in `dsl.Vector`. |
| PostGIS distance queries are slow | Create a `GIST`/`SP-GiST` index and confirm your SRID matches the input coordinates. |
| Timescale hypertable not created | Verify the annotation includes the correct time column and that TimescaleDB is installed. |

---

Continue to [performance-observability.md](./performance-observability.md) to understand how extensions interact with monitoring.
