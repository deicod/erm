package schema

import "github.com/deicod/erm/internal/orm/dsl"

type Post struct{ dsl.Schema }

func (Post) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
		dsl.UUIDv7("author_id"),
		dsl.UUIDv7("workspace_id"),
		dsl.String("title").NotEmpty(),
		dsl.String("body").Optional(),
		dsl.TimestampTZ("published_at").Optional(),
		dsl.TimestampTZ("created_at").DefaultNow(),
		dsl.TimestampTZ("updated_at").UpdateNow(),
	}
}

func (Post) Edges() []dsl.Edge {
	return []dsl.Edge{
		dsl.ToOne("author", "User").Field("author_id").OnDeleteCascade().Inverse("posts"),
		dsl.ToOne("workspace", "Workspace").
			Field("workspace_id").
			OnDeleteCascade().
			Polymorphic(
				dsl.PolymorphicTarget("Workspace", "kind = 'team'"),
				dsl.PolymorphicTarget("Workspace", "kind = 'personal'"),
			),
		dsl.ToMany("comments", "Comment").Ref("post"),
	}
}

func (Post) Indexes() []dsl.Index {
	return []dsl.Index{
		dsl.Idx("posts_workspace_created_at").On("workspace_id", "created_at"),
	}
}

func (Post) Query() dsl.QuerySpec {
	return dsl.Query().
		WithPredicates(
			dsl.NewPredicate("id", dsl.OpEqual).Named("IDEq"),
			dsl.NewPredicate("author_id", dsl.OpEqual).Named("AuthorIDEq"),
			dsl.NewPredicate("workspace_id", dsl.OpEqual).Named("WorkspaceIDEq"),
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
