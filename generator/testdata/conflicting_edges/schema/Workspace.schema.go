package schema

import "github.com/deicod/erm/orm/dsl"

type User struct{ dsl.Schema }

type Workspace struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
	}
}

func (Workspace) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
		dsl.UUIDv7("owner_id"),
	}
}

func (Workspace) Edges() []dsl.Edge {
	return []dsl.Edge{
		dsl.ToOne("primary_owner", "User").Field("owner_id"),
		dsl.ToOne("secondary_owner", "User").Field("owner_id"),
	}
}
