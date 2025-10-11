package schema

import "github.com/deicod/erm/orm/dsl"

type User struct{ dsl.Schema }

type Pet struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
	}
}

func (User) Edges() []dsl.Edge {
	return []dsl.Edge{
		dsl.ToMany("pets", "Pet").Inverse("owner"),
	}
}

func (User) Indexes() []dsl.Index { return nil }

func (Pet) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
	}
}

func (Pet) Edges() []dsl.Edge { return nil }

func (Pet) Indexes() []dsl.Index { return nil }
