package schema

import "github.com/deicod/erm/orm/dsl"

type Node struct{ dsl.Schema }

func (Node) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
	}
}

func (Node) Edges() []dsl.Edge {
	return []dsl.Edge{
		dsl.ToOne("parent", "Node"),
		dsl.ToMany("children", "Node"),
	}
}
