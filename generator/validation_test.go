package generator

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateEntitiesDetectsForeignKeyTypeMismatch(t *testing.T) {
	root := filepath.Join("testdata", "invalid_fk")

	_, err := loadEntities(root)
	if err == nil {
		t.Fatalf("expected loadEntities to fail for invalid foreign key type")
	}

	var validationErr *SchemaValidationErrorList
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected SchemaValidationErrorList, got %v", err)
	}
	if validationErr == nil {
		t.Fatalf("expected validation error details")
	}
	if len(validationErr.Problems) != 1 {
		t.Fatalf("expected 1 validation problem, got %d", len(validationErr.Problems))
	}

	problem := validationErr.Problems[0]
	if problem.Field != "parent_id" {
		t.Fatalf("expected problem on field parent_id, got %q", problem.Field)
	}
	if !strings.Contains(problem.Detail, "parent_id") {
		t.Fatalf("expected problem detail to mention parent_id, got %q", problem.Detail)
	}
	if !strings.Contains(err.Error(), "Use dsl.UUIDv7(\"parent_id\")") {
		t.Fatalf("expected error suggestion to recommend UUIDv7, got:\n%s", err)
	}
}

func TestValidateEntitiesDetectsMissingTarget(t *testing.T) {
	root := filepath.Join("testdata", "missing_target")

	_, err := loadEntities(root)
	if err == nil {
		t.Fatalf("expected loadEntities to fail for missing target")
	}

	var validationErr *SchemaValidationErrorList
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected SchemaValidationErrorList, got %v", err)
	}
	if len(validationErr.Problems) != 1 {
		t.Fatalf("expected 1 validation problem, got %d", len(validationErr.Problems))
	}
	problem := validationErr.Problems[0]
	if !strings.Contains(problem.Detail, "target entity \"Writer\" not found") {
		t.Fatalf("expected missing target detail, got %q", problem.Detail)
	}
}

func TestValidateEntitiesRequiresExplicitForeignKeyField(t *testing.T) {
	root := filepath.Join("testdata", "missing_fk_field")

	_, err := loadEntities(root)
	if err == nil {
		t.Fatalf("expected loadEntities to fail for missing foreign key field")
	}

	var validationErr *SchemaValidationErrorList
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected SchemaValidationErrorList, got %v", err)
	}
	if len(validationErr.Problems) != 1 {
		t.Fatalf("expected 1 validation problem, got %d", len(validationErr.Problems))
	}
	problem := validationErr.Problems[0]
	if !strings.Contains(problem.Detail, "edge overrides column \"author_id\"") {
		t.Fatalf("expected detail about missing author_id column, got %q", problem.Detail)
	}
}

func TestValidateEntitiesDetectsConflictingEdges(t *testing.T) {
	root := filepath.Join("testdata", "conflicting_edges")

	_, err := loadEntities(root)
	if err == nil {
		t.Fatalf("expected loadEntities to fail for conflicting edges")
	}

	var validationErr *SchemaValidationErrorList
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected SchemaValidationErrorList, got %v", err)
	}
	if len(validationErr.Problems) != 1 {
		t.Fatalf("expected 1 validation problem, got %d", len(validationErr.Problems))
	}
	problem := validationErr.Problems[0]
	if !strings.Contains(problem.Detail, "both reference column \"owner_id\"") {
		t.Fatalf("expected detail about shared owner_id column, got %q", problem.Detail)
	}
}

func TestValidateEntitiesDetectsInvalidSelfReference(t *testing.T) {
	root := filepath.Join("testdata", "self_ref_invalid")

	_, err := loadEntities(root)
	if err == nil {
		t.Fatalf("expected loadEntities to fail for invalid self reference")
	}

	var validationErr *SchemaValidationErrorList
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected SchemaValidationErrorList, got %v", err)
	}
	if len(validationErr.Problems) != 2 {
		t.Fatalf("expected 2 validation problems, got %d", len(validationErr.Problems))
	}
	var missingField, missingRef bool
	for _, problem := range validationErr.Problems {
		if strings.Contains(problem.Detail, "self-referential to-one") {
			missingField = true
		}
		if strings.Contains(problem.Detail, "self-referential to-many") {
			missingRef = true
		}
	}
	if !missingField || !missingRef {
		t.Fatalf("expected both self-reference validations, got %#v", validationErr.Problems)
	}
}
