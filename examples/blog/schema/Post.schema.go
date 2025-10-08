package schema

import "github.com/deicod/erm/internal/orm/dsl"

type Post struct{ dsl.Schema }

func (Post) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
		dsl.String("title").NotEmpty(),
		dsl.String("body").Optional(),
		dsl.Time("created_at").DefaultNow(),
		dsl.Time("updated_at").UpdateNow(),
	}
}

func (Post) Edges() []dsl.Edge    { return nil }
func (Post) Indexes() []dsl.Index { return nil }
