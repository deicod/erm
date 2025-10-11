package schema

import "github.com/deicod/erm/orm/dsl"

type Workspace struct{ dsl.Schema }

func (Workspace) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
		dsl.String("slug").NotEmpty().Unique(),
		dsl.String("name").NotEmpty(),
		dsl.String("description").Optional().Length(512),
		dsl.TimestampTZ("created_at").DefaultNow(),
		dsl.TimestampTZ("updated_at").UpdateNow(),
	}
}

func (Workspace) Edges() []dsl.Edge {
	return []dsl.Edge{
		dsl.ToMany("memberships", "Membership").Ref("workspace"),
		dsl.ToMany("posts", "Post").Ref("workspace"),
	}
}

func (Workspace) Indexes() []dsl.Index {
	return []dsl.Index{
		dsl.Idx("workspace_slug_unique").On("slug").Unique(),
	}
}

func (Workspace) Query() dsl.QuerySpec {
	return dsl.Query().
		WithPredicates(
			dsl.NewPredicate("id", dsl.OpEqual).Named("IDEq"),
			dsl.NewPredicate("slug", dsl.OpEqual).Named("SlugEq"),
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
