package generator

func normalizeEntityColumns(entities []Entity) {
	for i := range entities {
		normalizeEntity(&entities[i])
	}
}

func normalizeEntity(ent *Entity) {
	for i := range ent.Fields {
		field := ent.Fields[i]
		field.Column = fieldColumn(field)
		ent.Fields[i] = field
	}
	for i := range ent.Edges {
		edge := ent.Edges[i]
		if edge.Column != "" {
			edge.Column = normalizeIdentifier(edge.Column)
		}
		if edge.RefName != "" {
			edge.RefName = normalizeIdentifier(edge.RefName)
		}
		ent.Edges[i] = edge
	}
	for i := range ent.Indexes {
		idx := ent.Indexes[i]
		if len(idx.Columns) == 0 {
			ent.Indexes[i] = idx
			continue
		}
		cols := make([]string, len(idx.Columns))
		for j, col := range idx.Columns {
			cols[j] = normalizeIdentifier(col)
		}
		idx.Columns = cols
		ent.Indexes[i] = idx
	}
}
