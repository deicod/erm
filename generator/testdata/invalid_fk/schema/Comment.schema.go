package schema

import "github.com/deicod/erm/orm/dsl"

type Comment struct{ dsl.Schema }

func (Comment) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
		dsl.Integer("parent_id").Optional(),
	}
}

func (Comment) Edges() []dsl.Edge {
	return []dsl.Edge{
		dsl.ToOne("parent", "Comment").Field("parent_id").Optional(),
	}
}
