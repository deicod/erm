# Extensions Support

The erm framework provides first-class support for PostgreSQL extensions including PostGIS, pgvector, and TimescaleDB. These extensions are integrated into the schema DSL and generator to provide type-safe operations and proper database setup.

## PostGIS Support

PostGIS extends PostgreSQL with geographic objects, allowing location queries to be run in SQL. The erm framework provides type-safe integration for geometric data types and operations.

### Geometric Types

The framework includes special field types for geographic data:

```go
type Location struct{ dsl.Schema }

func (Location) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.Point("coordinates"),           // 2D point: POINT
        dsl.PointZ("coordinates_3d"),       // 3D point: POINTZ
        dsl.LineString("path"),             // Line string: LINESTRING
        dsl.Polygon("boundary"),            // Polygon: POLYGON
        dsl.Geometry("shape"),              // Generic geometry
        dsl.GeometryCollection("shapes"),   // Collection of geometries
        dsl.MultiPoint("interest_points"),  // Multiple points
        dsl.MultiLineString("roads"),       // Multiple line strings
        dsl.MultiPolygon("districts"),      // Multiple polygons
        dsl.Geography("area"),              // Geography type (WGS84)
    }
}
```

### PostGIS Functions

The generated models include helpers for common PostGIS operations:

```go
// Example operations made available through generated code
location, err := client.Location.Query().Where(
    dsl.Location.DistanceWithin("coordinates", point, 1000), // Within 1000m
).First(ctx)

locations, err := client.Location.Query().Where(
    dsl.Location.Intersects("boundary", otherPolygon),
).All(ctx)
```

### Spatial Indexes

Spatial indexes can be defined for optimized geographic queries:

```go
func (Location) Indexes() []dsl.Index {
    return []dsl.Index{
        dsl.Index().Spatial().Fields("coordinates"),  // 2D spatial index
        dsl.Index().Spatial().Fields("boundary"),     // Spatial index on polygon
        dsl.Index().Gist().Fields("area"),            // GiST index for geography
    }
}
```

### Migration Integration

The framework ensures PostGIS extension is enabled in migrations:

```sql
-- Generated migration ensures PostGIS is available
CREATE EXTENSION IF NOT EXISTS postgis;
```

## pgvector Support

pgvector enables efficient storage and similarity search of embedding vectors for AI/ML applications. The framework provides typed support for vector operations.

### Vector Types

```go
type Embedding struct{ dsl.Schema }

func (Embedding) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.Vector("embedding", 1536),      // Float vector with dimension 1536
        dsl.Vector64("compressed", 64),     // 64-bit vector
        dsl.VectorFloat16("quantized", 256), // Float16 vector (compressed)
        dsl.SparseVector("sparse", 1000),   // Sparse vector
    }
}
```

### Vector Operations

The framework generates helpers for vector similarity searches:

```go
// Cosine similarity search
similarEmbeddings, err := client.Embedding.Query().Where(
    dsl.Embedding.CosineDistance("embedding", queryVector, 0.8),
).OrderBy(dsl.Embedding.CosineDistanceTo("embedding", queryVector)).All(ctx)

// Euclidean distance search
nearbyEmbeddings, err := client.Embedding.Query().Where(
    dsl.Embedding.EuclideanDistance("embedding", queryVector, 1.5),
).All(ctx)

// Inner product search
relatedEmbeddings, err := client.Embedding.Query().Where(
    dsl.Embedding.InnerProduct("embedding", queryVector, 0.5),
).OrderBy(dsl.Embedding.InnerProductTo("embedding", queryVector)).All(ctx)
```

### Vector Indexes

Specialized indexes for efficient vector similarity search:

```go
func (Embedding) Indexes() []dsl.Index {
    return []dsl.Index{
        dsl.Index().Hnsw().Fields("embedding"),    // HNSW index for fast ANN search
        dsl.Index().Ivfflat().Fields("embedding"), // IVFFlat index for balanced performance
        dsl.Index().Ivfpq().Fields("embedding"),   // IVFPQ for memory-efficient search
    }
}
```

### Migration Integration

pgvector extension is automatically enabled:

```sql
-- Generated migration ensures pgvector is available
CREATE EXTENSION IF NOT EXISTS vector;
```

## TimescaleDB Support

TimescaleDB is a time-series database built on PostgreSQL. The framework provides integration for hypertable creation and time-series operations.

### Time-Series Schemas

```go
type Metric struct{ dsl.Schema }

func (Metric) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.Time("timestamp").Required(),  // Time column for partitioning
        dsl.String("name").Required(),
        dsl.Float("value"),
        dsl.JSON("tags"),                  // JSON for flexible metadata
        dsl.String("device_id").Required(), // Series identifier
    }
}

// Define as hypertable
func (Metric) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.Annotation{"timescaledb": map[string]interface{}{
            "hypertable": true,
            "time_column": "timestamp",
            "partition_column": "device_id",
            "chunk_time_interval": "1 day",
        }},
    }
}
```

### Hypertable Configuration

Time-series specific schema annotations:

```go
func (Metric) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.Annotation{"timescaledb": map[string]interface{}{
            "hypertable": true,                    // Create as hypertable
            "time_column": "timestamp",            // Column to partition by time
            "partition_column": "device_id",       // Optional: partition by series
            "chunk_time_interval": "1 day",        // Time interval for chunks
            "compress_segmentby": ["device_id"],   // Compression options
            "compress_orderby": ["timestamp"],     // Compression order
        }},
    }
}
```

### Time-Series Functions

Generated helpers for time-series operations:

```go
// Time-series aggregation
metrics, err := client.Metric.Query().Where(
    dsl.Metric.TimestampBetween(time.Now().Add(-24*time.Hour), time.Now()),
).Aggregate(
    dsl.Metric.Avg("value"),
    dsl.Metric.TimeBucket("1 hour", "timestamp"),
).All(ctx)

// Gap-filling operations
metrics, err := client.Metric.Query().Where(
    dsl.Metric.TimestampBetween(startDate, endDate),
).TimeSeries(
    dsl.Metric.FillGaps("timestamp", "1 hour"),
).All(ctx)
```

### Continuous Aggregations

Support for materialized views of time-series data:

```go
func (Metric) Views() []dsl.View {
    return []dsl.View{
        dsl.View("hourly_metrics").TimeSeries(map[string]interface{}{
            "query": `
                SELECT 
                    time_bucket('1 hour', timestamp) as bucket,
                    avg(value) as avg_value,
                    max(value) as max_value,
                    device_id
                FROM metrics 
                GROUP BY bucket, device_id
            `,
            "refresh_interval": "5 minutes",
        }),
    }
}
```

## Extension Installation

### Automatic Installation

The framework can automatically install extensions:

```go
// In schema definition
func (System) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.Annotation{"extensions": []string{
            "postgis",
            "vector", 
            "timescaledb",
        }},
    }
}
```

### Migration Scripts

Extensions are enabled in the generated migration files:

```go
// Example migration enabling extensions
func (m *Migrations) Up_20250101_000001() error {
    return m.Execute([]string{
        "CREATE EXTENSION IF NOT EXISTS postgis;",
        "CREATE EXTENSION IF NOT EXISTS vector;",
        "CREATE EXTENSION IF NOT EXISTS timescaledb;",
    })
}
```

## Performance Considerations

### PostGIS
- Use spatial indexes for geometric queries
- Consider geography vs geometry types based on use case
- Be mindful of coordinate system projections

### pgvector
- Choose appropriate index type based on query patterns
- Consider vector dimensionality vs. performance tradeoffs
- Use quantization for high-dimensional vectors

### TimescaleDB
- Properly size chunk time intervals for your data patterns
- Use compression for historical data
- Configure appropriate retention policies

## Best Practices

1. **Choose the right extension** for your use case:
   - PostGIS for geographic applications
   - pgvector for AI/ML embedding similarity
   - TimescaleDB for time-series data

2. **Proper indexing** for each extension type:
   - Spatial indexes for PostGIS
   - Vector indexes for pgvector
   - Time-partitioned indexes for TimescaleDB

3. **Dimensional planning** for vectors:
   - Balance between accuracy and performance
   - Consider quantization for storage efficiency

4. **Migration safety**:
   - Test extension installation in development
   - Plan for extension updates
   - Consider backup/restore procedures for extension-dependent data

## Example: Combined Usage

A schema that uses multiple extensions:

```go
type SensorReading struct{ dsl.Schema }

func (SensorReading) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.Time("timestamp").Required(),           // TimescaleDB time column
        dsl.Point("location"),                     // PostGIS location
        dsl.Vector("sensor_signature", 128),       // pgvector for sensor pattern
        dsl.Float("temperature"),
        dsl.Float("humidity"),
        dsl.String("sensor_id").Required(),
    }
}

func (SensorReading) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.Annotation{"timescaledb": map[string]interface{}{
            "hypertable": true,
            "time_column": "timestamp",
            "partition_column": "sensor_id",
        }},
    }
}

func (SensorReading) Indexes() []dsl.Index {
    return []dsl.Index{
        dsl.Index().Spatial().Fields("location"),     // PostGIS spatial index
        dsl.Index().Hnsw().Fields("sensor_signature"), // pgvector index
        dsl.Index().Fields("sensor_id", "timestamp"),  // Hypertable partition index
    }
}
```