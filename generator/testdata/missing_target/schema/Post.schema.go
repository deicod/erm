package schema

import "github.com/deicod/erm/orm/dsl"

type Post struct{ dsl.Schema }

func (Post) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
	}
}

func (Post) Edges() []dsl.Edge {
	return []dsl.Edge{
		dsl.ToOne("author", "Writer").Field("author_id"),
	}
}
