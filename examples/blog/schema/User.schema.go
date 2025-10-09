package schema

import "github.com/deicod/erm/internal/orm/dsl"

type User struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
		dsl.String("email").Unique().NotEmpty(),
		dsl.String("name").Optional(),
		dsl.TimestampTZ("created_at").DefaultNow(),
		dsl.TimestampTZ("updated_at").UpdateNow(),
	}
}

func (User) Edges() []dsl.Edge {
	return []dsl.Edge{
		dsl.ToMany("posts", "Post").Ref("author_id"),
		dsl.ToMany("memberships", "Membership").Ref("user"),
		dsl.ToMany("comments", "Comment").Ref("author"),
	}
}
func (User) Indexes() []dsl.Index { return []dsl.Index{dsl.Idx("idx_user_email").On("email").Unique()} }

func (User) Query() dsl.QuerySpec {
	return dsl.Query().
		WithPredicates(
			dsl.NewPredicate("id", dsl.OpEqual).Named("IDEq"),
			dsl.NewPredicate("email", dsl.OpILike).Named("EmailILike"),
		).
		WithOrders(
			dsl.OrderBy("created_at", dsl.SortDesc).Named("CreatedAtDesc"),
			dsl.OrderBy("email", dsl.SortAsc).Named("EmailAsc"),
		).
		WithAggregates(
			dsl.CountAggregate("Count"),
		).
		WithDefaultLimit(25).
		WithMaxLimit(100)
}
