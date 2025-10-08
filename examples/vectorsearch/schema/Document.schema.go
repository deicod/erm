package schema

import "github.com/deicod/erm/internal/orm/dsl"

type Document struct{ dsl.Schema }

func (Document) Fields() []dsl.Field {
	return []dsl.Field{
		dsl.UUIDv7("id").Primary(),
		dsl.String("title").NotEmpty(),
		dsl.String("content").Optional(),
		dsl.Vector("embedding", 1536),
		dsl.JSON("metadata").Optional(),
		dsl.Time("created_at").DefaultNow(),
		dsl.Time("updated_at").UpdateNow(),
	}
}

func (Document) Edges() []dsl.Edge { return nil }

func (Document) Indexes() []dsl.Index {
	return []dsl.Index{
		dsl.Idx("idx_documents_embedding").On("embedding").MethodUsing("ivfflat"),
	}
}
