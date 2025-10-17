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

func TestValidateEntitiesRequiresForeignKeyField(t *testing.T) {
	root := filepath.Join("testdata", "missing_fk_field")

	_, err := loadEntities(root)
	if err == nil {
		t.Fatalf("expected loadEntities to fail when foreign key field is missing")
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
	if problem.Field != "author_id" {
		t.Fatalf("expected problem on field author_id, got %q", problem.Field)
	}
	if !strings.Contains(problem.Detail, "author_id") {
		t.Fatalf("expected problem detail to mention author_id, got %q", problem.Detail)
	}
	if !strings.Contains(problem.Suggestion, "Post.Fields()") {
		t.Fatalf("expected suggestion to mention Post.Fields(), got %q", problem.Suggestion)
	}
}
