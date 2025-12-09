package schema

import "github.com/deicod/erm/orm/dsl"

type User struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
		dsl.Text("name"),
	}
}
