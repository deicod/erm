package schema

import "github.com/deicod/erm/internal/orm/dsl"

type User struct{ dsl.Schema }

func (User) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
		dsl.Text("slug").Computed(dsl.Computed(dsl.Expression("id::text"))),
		dsl.TimestampTZ("created_at").DefaultNow(),
		dsl.TimestampTZ("updated_at").UpdateNow(),
	}
}

func (User) Edges() []dsl.Edge    { return nil }
func (User) Indexes() []dsl.Index { return nil }

func (User) Query() dsl.QuerySpec {
	return dsl.Query().
		WithPredicates(
			dsl.NewPredicate("id", dsl.OpEqual).Named("IDEq"),
		).
		WithOrders(
			dsl.OrderBy("created_at", dsl.SortAsc).Named("CreatedAtAsc"),
		).
		WithAggregates(
			dsl.CountAggregate("Count"),
		)
}
