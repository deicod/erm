package generator

import (
	"fmt"
	"strings"

	"github.com/agnivade/levenshtein"

	"github.com/deicod/erm/orm/dsl"
)

type SchemaValidationError struct {
	Entity     string
	Edge       string
	Field      string
	Column     string
	Target     string
	Detail     string
	Suggestion string
}

func (e SchemaValidationError) describe() string {
	location := e.location()
	if e.Detail == "" {
		if location == "" {
			return "invalid schema configuration"
		}
		return fmt.Sprintf("%s: invalid schema configuration", location)
	}
	if location == "" {
		return e.Detail
	}
	return fmt.Sprintf("%s: %s", location, e.Detail)
}

func (e SchemaValidationError) location() string {
	parts := []string{}
	if e.Entity != "" {
		parts = append(parts, e.Entity)
	}
	if e.Edge != "" {
		parts = append(parts, e.Edge)
	}
	if e.Field != "" {
		parts = append(parts, e.Field)
	}
	return strings.Join(parts, ".")
}

type SchemaValidationErrorList struct {
	Problems []SchemaValidationError
}

func (l *SchemaValidationErrorList) Error() string {
	if l == nil || len(l.Problems) == 0 {
		return "schema validation failed"
	}
	if len(l.Problems) == 1 {
		entry := l.Problems[0]
		if entry.Suggestion == "" {
			return entry.describe()
		}
		return fmt.Sprintf("%s\nHint: %s", entry.describe(), entry.Suggestion)
	}
	var b strings.Builder
	b.WriteString("schema validation failed:")
	for _, problem := range l.Problems {
		b.WriteString("\n  - ")
		b.WriteString(problem.describe())
		if problem.Suggestion != "" {
			b.WriteString("\n    Hint: ")
			b.WriteString(problem.Suggestion)
		}
	}
	return b.String()
}

type entityMeta struct {
	entity  Entity
	fields  map[string]dsl.Field
	primary dsl.Field
}

func validateEntities(entities []Entity) error {
	metas := make(map[string]entityMeta, len(entities))
	for _, ent := range entities {
		fields := make(map[string]dsl.Field, len(ent.Fields))
		for _, field := range ent.Fields {
			fields[fieldColumn(field)] = field
		}
		primary, _ := findPrimaryField(ent)
		metas[ent.Name] = entityMeta{entity: ent, fields: fields, primary: primary}
	}

	problems := []SchemaValidationError{}
	for _, ent := range entities {
		meta := metas[ent.Name]
		columnOwners := make(map[string]dsl.Edge)
		for _, edge := range ent.Edges {
			if edge.Target == "" {
				problems = append(problems, SchemaValidationError{
					Entity:     ent.Name,
					Edge:       edge.Name,
					Detail:     "edge missing target entity",
					Suggestion: "Set the target entity name or remove the edge.",
				})
				continue
			}
			targetMeta, ok := metas[edge.Target]
			if !ok {
				suggestion := suggestEntityName(edge.Target, metas)
				problems = append(problems, SchemaValidationError{
					Entity:     ent.Name,
					Edge:       edge.Name,
					Target:     edge.Target,
					Detail:     fmt.Sprintf("target entity %q not found", edge.Target),
					Suggestion: suggestion,
				})
				continue
			}
			targetPrimary, ok := findPrimaryField(target)
			if !ok {
				continue
			}
			column := edgeColumn(edge)
			field, exists := fieldsByColumn[column]
			if !exists {
				if edge.Column == "" || isGeneratedInverse(edge) {
					continue
				}
				detail := fmt.Sprintf("foreign key column %q referenced by edge %s.%s is missing from %s.Fields()", column, ent.Name, edge.Name, ent.Name)
				suggestion := fmt.Sprintf("Add %q to %s.Fields() using the same type as %s.%s.", column, ent.Name, target.Name, targetPrimary.Name)
				problems = append(problems, SchemaValidationError{
					Entity:     ent.Name,
					Edge:       edge.Name,
					Field:      column,
					Column:     column,
					Target:     target.Name,
					Detail:     detail,
					Suggestion: suggestion,
				})
				continue
			}
			if field.Type == targetPrimary.Type {
				continue
			}

			switch edge.Kind {
			case dsl.EdgeToOne:
				column := edgeColumn(edge)
				if existing, exists := columnOwners[column]; exists {
					problems = append(problems, SchemaValidationError{
						Entity:     ent.Name,
						Edge:       edge.Name,
						Target:     edge.Target,
						Column:     column,
						Detail:     fmt.Sprintf("edges %q and %q both reference column %q", existing.Name, edge.Name, column),
						Suggestion: "Use .Field(\"<column>\") to assign unique columns to each edge.",
					})
				} else {
					columnOwners[column] = edge
				}

				if ent.Name == edge.Target && edge.Column == "" {
					problems = append(problems, SchemaValidationError{
						Entity:     ent.Name,
						Edge:       edge.Name,
						Detail:     "self-referential to-one edges must declare .Field(...) to avoid ambiguous column names",
						Suggestion: fmt.Sprintf("Call dsl.ToOne(%q, %q).Field(\"%s_id\") to pin the column name.", edge.Name, edge.Target, strings.ToLower(edge.Name)),
					})
				}

				if edge.Column != "" && edge.RefName == "" {
					if _, exists := meta.fields[edge.Column]; !exists {
						problems = append(problems, SchemaValidationError{
							Entity:     ent.Name,
							Edge:       edge.Name,
							Column:     edge.Column,
							Target:     edge.Target,
							Detail:     fmt.Sprintf("edge overrides column %q but %s.%s is not defined", edge.Column, ent.Name, edge.Column),
							Suggestion: fmt.Sprintf("Add dsl.Field(%q) to %s.Fields() or remove the .Field override.", edge.Column, ent.Name),
						})
					}
				}

				field, exists := meta.fields[column]
				if !exists {
					continue
				}
				targetPrimary := targetMeta.primary
				if targetPrimary.Name == "" {
					continue
				}
				if field.Type == targetPrimary.Type {
					continue
				}
				detail := fmt.Sprintf("foreign key column %q uses type %s but %s.%s expects %s", column, field.Type, targetMeta.entity.Name, targetPrimary.Name, targetPrimary.Type)
				suggestion := buildTypeMismatchSuggestion(ent, field, targetMeta.entity, targetPrimary)
				problems = append(problems, SchemaValidationError{
					Entity:     ent.Name,
					Edge:       edge.Name,
					Field:      field.Name,
					Column:     column,
					Target:     targetMeta.entity.Name,
					Detail:     detail,
					Suggestion: suggestion,
				})
			case dsl.EdgeToMany:
				if ent.Name == edge.Target && edge.RefName == "" {
					problems = append(problems, SchemaValidationError{
						Entity:     ent.Name,
						Edge:       edge.Name,
						Detail:     "self-referential to-many edges must use .Ref(...) to point at the owning column",
						Suggestion: fmt.Sprintf("Call dsl.ToMany(%q, %q).Ref(\"<edge>\") to reference the owning column.", edge.Name, edge.Target),
					})
				}

				refColumn := edgeRefColumn(ent, edge, meta.primary)
				if refColumn == "" {
					continue
				}
				if field, exists := targetMeta.fields[refColumn]; exists {
					if meta.primary.Name != "" && field.Type != meta.primary.Type {
						detail := fmt.Sprintf("reverse edge column %q uses type %s but %s.%s expects %s", refColumn, field.Type, ent.Name, meta.primary.Name, meta.primary.Type)
						suggestion := buildTypeMismatchSuggestion(targetMeta.entity, field, ent, meta.primary)
						problems = append(problems, SchemaValidationError{
							Entity:     edge.Target,
							Edge:       edge.Name,
							Field:      field.Name,
							Column:     refColumn,
							Target:     ent.Name,
							Detail:     detail,
							Suggestion: suggestion,
						})
					}
				}
			}
		}
	}

	if len(problems) == 0 {
		return nil
	}
	return &SchemaValidationErrorList{Problems: problems}
}

func suggestEntityName(target string, metas map[string]entityMeta) string {
	if len(metas) == 0 {
		return ""
	}
	lower := strings.ToLower(target)
	best := ""
	bestDist := -1
	for name := range metas {
		dist := levenshtein.ComputeDistance(lower, strings.ToLower(name))
		if bestDist == -1 || dist < bestDist {
			bestDist = dist
			best = name
		}
	}
	if best == "" || bestDist > 3 {
		return ""
	}
	return fmt.Sprintf("Did you mean %q?", best)
}

func buildTypeMismatchSuggestion(ent Entity, field dsl.Field, target Entity, targetPrimary dsl.Field) string {
	switch targetPrimary.Type {
	case dsl.TypeUUID:
		return fmt.Sprintf("Use dsl.UUIDv7(%q) or dsl.UUID(%q) for %s.%s to match %s.%s.", field.Name, field.Name, ent.Name, field.Name, target.Name, targetPrimary.Name)
	default:
		return fmt.Sprintf("Ensure %s.%s uses the same type as %s.%s (%s).", ent.Name, field.Name, target.Name, targetPrimary.Name, targetPrimary.Type)
	}
}

func isGeneratedInverse(edge dsl.Edge) bool {
	if edge.Annotations == nil {
		return false
	}
	if v, ok := edge.Annotations[generatedInverseAnnotation]; ok {
		if flag, ok := v.(bool); ok {
			return flag
		}
	}
	return false
}
