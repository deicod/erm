package generator

import (
	"fmt"
	"strings"

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

func validateEntities(entities []Entity) error {
	index := make(map[string]Entity, len(entities))
	for _, ent := range entities {
		index[ent.Name] = ent
	}

	problems := []SchemaValidationError{}
	for _, ent := range entities {
		fieldsByColumn := make(map[string]dsl.Field, len(ent.Fields))
		for _, field := range ent.Fields {
			fieldsByColumn[fieldColumn(field)] = field
		}

		for _, edge := range ent.Edges {
			if edge.Kind != dsl.EdgeToOne {
				continue
			}
			if edge.Target == "" {
				continue
			}
			target, ok := index[edge.Target]
			if !ok {
				continue
			}
			targetPrimary, ok := findPrimaryField(target)
			if !ok {
				continue
			}
			column := edgeColumn(edge)
			field, exists := fieldsByColumn[column]
			if !exists {
				continue
			}
			if field.Type == targetPrimary.Type {
				continue
			}

			detail := fmt.Sprintf("foreign key column %q uses type %s but %s.%s expects %s", column, field.Type, target.Name, targetPrimary.Name, targetPrimary.Type)
			suggestion := buildTypeMismatchSuggestion(ent, field, target, targetPrimary)
			problems = append(problems, SchemaValidationError{
				Entity:     ent.Name,
				Edge:       edge.Name,
				Field:      field.Name,
				Column:     column,
				Target:     target.Name,
				Detail:     detail,
				Suggestion: suggestion,
			})
		}
	}

	if len(problems) == 0 {
		return nil
	}
	return &SchemaValidationErrorList{Problems: problems}
}

func buildTypeMismatchSuggestion(ent Entity, field dsl.Field, target Entity, targetPrimary dsl.Field) string {
	switch targetPrimary.Type {
	case dsl.TypeUUID:
		return fmt.Sprintf("Use dsl.UUIDv7(%q) or dsl.UUID(%q) for %s.%s to match %s.%s.", field.Name, field.Name, ent.Name, field.Name, target.Name, targetPrimary.Name)
	default:
		return fmt.Sprintf("Ensure %s.%s uses the same type as %s.%s (%s).", ent.Name, field.Name, target.Name, targetPrimary.Name, targetPrimary.Type)
	}
}
