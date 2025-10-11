package schema

import "github.com/deicod/erm/orm/dsl"

type Membership struct{ dsl.Schema }

func (Membership) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
		dsl.UUIDv7("workspace_id"),
		dsl.UUIDv7("user_id"),
		dsl.String("role").NotEmpty(),
		dsl.TimestampTZ("joined_at").DefaultNow(),
	}
}

func (Membership) Edges() []dsl.Edge {
	return []dsl.Edge{
		dsl.ToOne("workspace", "Workspace").Field("workspace_id"),
		dsl.ToOne("user", "User").Field("user_id"),
	}
}

func (Membership) Indexes() []dsl.Index {
	return []dsl.Index{
		dsl.Idx("membership_workspace_user_unique").On("workspace_id", "user_id").Unique(),
	}
}

func (Membership) Query() dsl.QuerySpec {
	return dsl.Query().
		WithPredicates(
			dsl.NewPredicate("workspace_id", dsl.OpEqual).Named("WorkspaceIDEq"),
			dsl.NewPredicate("user_id", dsl.OpEqual).Named("UserIDEq"),
			dsl.NewPredicate("role", dsl.OpEqual).Named("RoleEq"),
		).
		WithDefaultLimit(50).
		WithMaxLimit(500)
}
