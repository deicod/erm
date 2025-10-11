package schema

import "github.com/deicod/erm/orm/dsl"

type Comment struct{ dsl.Schema }

func (Comment) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
		dsl.UUIDv7("post_id"),
		dsl.UUIDv7("author_id"),
		dsl.UUIDv7("parent_id").Optional(),
		dsl.String("body").NotEmpty().Length(2000),
		dsl.TimestampTZ("created_at").DefaultNow(),
		dsl.TimestampTZ("updated_at").UpdateNow(),
	}
}

func (Comment) Edges() []dsl.Edge {
	return []dsl.Edge{
		dsl.ToOne("post", "Post").Field("post_id").OnDeleteCascade(),
		dsl.ToOne("author", "User").Field("author_id").OnDeleteCascade(),
		dsl.ToOne("parent", "Comment").Field("parent_id").Optional().OnDeleteSetNull(),
		dsl.ToMany("replies", "Comment").Ref("parent").Polymorphic(
			dsl.PolymorphicTarget("Comment", "parent_id IS NOT NULL"),
		),
	}
}

func (Comment) Indexes() []dsl.Index {
	return []dsl.Index{
		dsl.Idx("comment_post_created_at").On("post_id", "created_at"),
	}
}

func (Comment) Query() dsl.QuerySpec {
	return dsl.Query().
		WithPredicates(
			dsl.NewPredicate("post_id", dsl.OpEqual).Named("PostIDEq"),
			dsl.NewPredicate("author_id", dsl.OpEqual).Named("AuthorIDEq"),
		).
		WithOrders(
			dsl.OrderBy("created_at", dsl.SortAsc).Named("CreatedAtAsc"),
		).
		WithDefaultLimit(25).
		WithMaxLimit(250)
}
