# End-to-End Examples

This chapter showcases complete workflows that combine schema design, generation, GraphQL operations, authentication, and
testing. Use these examples as templates when building real features.

---

## Example 1: Project Management Boards

### Goal

Create a simple project management API with `Workspace`, `Project`, and `Task` entities. Each workspace has many projects; each
project has many tasks. Users belong to workspaces with roles controlling access.

### Step 1 – Generate Schemas

```bash
erm new Workspace
erm new Project
erm new Task
```

### Step 2 – Author Schema Files

`internal/orm/schema/workspace.go`:

```go
package schema

import "github.com/erm-project/erm/orm/dsl"

type Workspace struct{ dsl.Schema }

func (Workspace) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.String("name").NotEmpty(),
        dsl.String("slug").NotEmpty().Unique(),
    }
}

func (Workspace) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToMany("projects", "Project").Ref("workspace"),
        dsl.ToMany("members", "User").Inverse("workspaces"),
    }
}

func (Workspace) Policy() dsl.Policy {
    return dsl.Policy{
        Query:  dsl.AllowIf("viewer.has_role('ADMIN') || viewer.has_role('MEMBER')"),
        Update: dsl.AllowIf("viewer.has_role('ADMIN')"),
    }
}

func (Workspace) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.Authz().Roles("ADMIN", "MEMBER"),
        dsl.GraphQL("Workspace").Description("Team workspace containing projects."),
    }
}
```

`internal/orm/schema/project.go`:

```go
package schema

import "github.com/erm-project/erm/orm/dsl"

type Project struct{ dsl.Schema }

func (Project) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.String("name").NotEmpty(),
        dsl.Enum("status", "BACKLOG", "ACTIVE", "DONE").Default("BACKLOG"),
    }
}

func (Project) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("workspace", "Workspace").Required().Field("workspace_id"),
        dsl.ToMany("tasks", "Task").Ref("project"),
    }
}

func (Project) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.GraphQL("Project").Description("Container for tasks.").FilterPreset("active", `status = 'ACTIVE'`),
    }
}
```

`internal/orm/schema/task.go`:

```go
package schema

import "github.com/erm-project/erm/orm/dsl"

type Task struct{ dsl.Schema }

func (Task) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.String("title").NotEmpty(),
        dsl.String("description").Optional(),
        dsl.Enum("priority", "LOW", "MEDIUM", "HIGH").Default("MEDIUM"),
        dsl.TimestampTZ("due_date").Optional().Nillable(),
        dsl.Bool("completed").Default(false),
    }
}

func (Task) Edges() []dsl.Edge {
    return []dsl.Edge{
        dsl.ToOne("project", "Project").Required().Field("project_id"),
        dsl.ToOne("assignee", "User").Optional().Field("assignee_id"),
    }
}

func (Task) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.GraphQL("Task").Description("Work item tracked within a project."),
    }
}
```

### Step 3 – Generate Code & Apply Migrations

```bash
erm gen
psql blog < migrations/*_workspace*.sql
psql blog < migrations/*_project*.sql
psql blog < migrations/*_task*.sql
```

### Step 4 – Seed Data

```go
workspace := client.Workspace.Create().SetName("Platform").SetSlug("platform").SaveX(ctx)
project := client.Project.Create().SetName("GraphQL Gateway").SetWorkspace(workspace).SetStatus(project.StatusACTIVE).SaveX(ctx)
client.Task.Create().SetTitle("Implement Node registry").SetProject(project).SaveX(ctx)
```

### Step 5 – Query via GraphQL

```graphql
query WorkspaceBoard($slug: String!) {
  workspaceBySlug(slug: $slug) {
    id
    name
    projects(first: 10, orderBy: { field: NAME, direction: ASC }) {
      edges {
        node {
          id
          name
          status
          tasks(first: 5) {
            edges {
              node {
                title
                priority
                completed
              }
            }
          }
        }
      }
    }
  }
}
```

### Step 6 – Custom Mutation

Extend the schema with a mutation to complete tasks:

```go
func (Task) Mutations() []dsl.Mutation {
    return []dsl.Mutation{
        dsl.Mutation{
            Name: "completeTask",
            InputFields: []dsl.MutationField{dsl.UUID("id")},
            OutputFields: []dsl.MutationField{dsl.Boolean("success")},
            Code: `task, err := client.Task.UpdateOneID(input.ID).SetCompleted(true).Save(ctx)
if err != nil {
    return nil, err
}
return &CompleteTaskPayload{Success: task.Completed}, nil`,
        },
    }
}
```

Run `erm gen` and invoke mutation:

```graphql
mutation CompleteTask($id: ID!) {
  completeTask(input: { id: $id }) {
    success
  }
}
```

### Step 6 – Stream Task Updates

Enable realtime dashboards by annotating entities with `dsl.GraphQLSubscriptions` and turning on the in-memory broker in `erm.yaml`. The generator adds subscription fields such as `taskCreated`, `taskUpdated`, and `taskDeleted` alongside typed resolver stubs. Wire your HTTP server with `server.NewServer` to enable the websocket transport, or provide a custom `subscriptions.Broker` implementation if you already emit change events to Redis/Kafka.

```graphql
subscription TaskCreated($project: ID!) {
  taskCreated(projectID: $project) {
    id
    title
    completed
  }
}
```

Publish events from background workers by calling `resolvers.Publish(ctx, broker, "Task", resolvers.SubscriptionTriggerCreated, gqlTask)` after persisting changes. Clients receive a hydrated `Task` object without polling, keeping kanban boards up to date.

---

## Example 2: Editorial Workspace Blog

The `examples/blog` project now models a multi-tenant editorial workflow with workspaces, membership roles, threaded comments, and performance-conscious
queries. Use it as a reference when designing complex relational graphs.

- **[Schema tour](../examples/blog/README.md)** – Highlights the new `Workspace`, `Membership`, and `Comment` schemas plus the cross-entity indexes they require.
- **[Validation walkthrough](../examples/blog/walkthroughs/validation.md)** – Shows how to exercise field validators, join-table uniqueness, and privacy policy
  assertions using `go test` and the sandbox harness.
- **[Performance profiling walkthrough](../examples/blog/walkthroughs/performance-profiling.md)** – Demonstrates query composition for workspace timelines,
  collecting `EXPLAIN ANALYZE` output, pprof traces, and dataloader metrics.
- **[Error handling walkthrough](../examples/blog/walkthroughs/error-handling.md)** – Provides a playbook for replaying production failures, capturing structured
  logs, and verifying remediation steps before redeploying.

Each walkthrough contains cut-and-paste commands so you can validate behavior end to end, from regeneration to observability dashboards.

---

## Example 3: Blog Query Builder

The `examples/blog` workspace demonstrates how query descriptors translate into fluent helpers for list and aggregate queries.

### Schema Highlights

`examples/blog/schema/Post.schema.go`:

```go
func (Post) Query() dsl.QuerySpec {
    return dsl.Query().
        WithPredicates(
            dsl.NewPredicate("author_id", dsl.OpEqual).Named("AuthorIDEq"),
            dsl.NewPredicate("title", dsl.OpILike).Named("TitleILike"),
        ).
        WithOrders(
            dsl.OrderBy("created_at", dsl.SortDesc).Named("CreatedAtDesc"),
        ).
        WithAggregates(
            dsl.CountAggregate("Count"),
        ).
        WithDefaultLimit(20).
        WithMaxLimit(200)
}
```

`examples/blog/schema/User.schema.go` adds complementary predicates (`IDEq`, `EmailILike`) and ordering helpers.

### Using the Generated Client

```go
// Fetch the five most recent posts by a specific author.
posts, err := client.Posts().
    Query().
    WhereAuthorIDEq(authorID).
    OrderByCreatedAtDesc().
    Limit(5).
    All(ctx)

// Count all posts matching a case-insensitive title fragment.
total, err := client.Posts().
    Query().
    WhereTitleILike("%roadmap%").
    Count(ctx)

if err != nil {
    log.Fatalf("query posts: %v", err)
}
```

Behind the scenes the runtime builds parametrised SQL using `runtime.BuildSelectSQL` and executes it via the pgx-backed
`pg.DB.Select` / `pg.DB.Aggregate` helpers.

### Step 7 – Test Privacy

```go
func TestWorkspacePolicy(t *testing.T) {
    ctx := authz.WithViewer(context.Background(), authz.Viewer{Roles: []string{"MEMBER"}})
    _, err := client.Workspace.Query().Only(ctx)
    require.NoError(t, err)

    ctxAdmin := authz.WithViewer(context.Background(), authz.Viewer{Roles: []string{"ADMIN"}})
    err = client.Workspace.Update().SetName("Updated").Exec(ctxAdmin)
    require.NoError(t, err)
}
```

---

## Example 4: Analytics Pipeline with TimescaleDB & pgvector

### Goal

Track ingestion events, store embeddings, and run similarity search over anomalies.

### Schema Highlights

```go
type Event struct{ dsl.Schema }

func (Event) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.TimestampTZ("occurred_at").NotEmpty(),
        dsl.String("category").NotEmpty(),
        dsl.JSON("payload").Optional(),
        dsl.Vector("embedding", 384).Optional(),
    }
}

func (Event) Annotations() []dsl.Annotation {
    return []dsl.Annotation{
        dsl.Extension("timescaledb"),
        dsl.Extension("vector"),
        dsl.Timescale(dsl.TimescaleOptions{TimeColumn: "occurred_at", ChunkInterval: time.Hour}),
    }
}
```

### Aggregation Query

```graphql
query EventsByHour($category: String!, $from: Time!, $to: Time!) {
  events(filter: { category: $category, occurredAtAfter: $from, occurredAtBefore: $to }) {
    histogram(interval: 1h) {
      bucketStart
      count
    }
  }
}
```

### Similarity Search

```graphql
query SimilarEvents($embedding: Vector!) {
  searchEvents(embedding: $embedding, limit: 5) {
    edges {
      node {
        id
        category
      }
      similarity
    }
  }
}
```

### Testing Embedding Indexes

```go
func TestEventSimilarity(t *testing.T) {
    client := testutil.NewClient(t)
    ctx := context.Background()

    vec := pgxext.VectorFromFloats(make([]float32, 384))
    _, err := client.Event.Create().SetCategory("anomaly").SetEmbedding(vec).Save(ctx)
    require.NoError(t, err)

    results, err := client.Event.Query().
        Where(event.EmbeddingNearestNeighbor(vec, orm.WithDistanceColumn("similarity"))).
        Limit(1).
        All(ctx)
    require.NoError(t, err)
    require.Len(t, results, 1)
}
```

---

## Example 5: Automation CLI Workflow

1. Add `//go:generate erm gen` to `internal/orm/schema/doc.go`.
2. Write a shell script:

```bash
#!/usr/bin/env bash
set -euo pipefail
erm gen
erm doctor
go test ./...
```

3. Configure CI pipeline to run the script. Capture generated docs with `tar -czf artifacts/docs.tar.gz docs/` for reviewer bundles.

---

These examples demonstrate how schema design, generation, GraphQL, authentication, and testing align across the stack. Adapt them
to your domain and keep documentation updated as patterns evolve.
