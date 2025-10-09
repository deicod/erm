package schema

import "github.com/deicod/erm/internal/orm/dsl"

type User struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
		dsl.String("email").Unique().NotEmpty(),
		dsl.String("name").Optional(),
		dsl.Time("created_at").DefaultNow(),
		dsl.Time("updated_at").UpdateNow(),
	}
}

func (User) Edges() []dsl.Edge {
	return []dsl.Edge{
		dsl.ToMany("posts", "Post").Ref("author_id"),
	}
}
func (User) Indexes() []dsl.Index { return []dsl.Index{dsl.Idx("idx_user_email").On("email").Unique()} }
