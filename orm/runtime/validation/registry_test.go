package validation

import (
	"context"
	"regexp"
	"testing"
	"time"
)

func TestRegistry_ValidateAggregatesErrors(t *testing.T) {
	reg := NewRegistry()
	reg.Entity("User").OnCreate(
		RuleFunc(func(_ context.Context, subject Subject) error {
			if v, ok := subject.Record.String("ID"); !ok || v == "" {
				return FieldError{Field: "ID", Message: "missing"}
			}
			return nil
		}),
		String("Email").Required().Matches(regexp.MustCompile(`^[^@]+@example.com$`)).Rule(),
	)

	err := reg.Validate(context.Background(), "User", OpCreate, Record{
		"Email": "invalid",
	}, nil)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	verrs, ok := err.(Errors)
	if !ok {
		t.Fatalf("expected Errors, got %T", err)
	}
	if len(verrs) != 2 {
		t.Fatalf("expected two errors, got %d", len(verrs))
	}
}

func TestRegistry_NoRules(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Validate(context.Background(), "User", OpCreate, nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegistry_CrossFieldRule(t *testing.T) {
	reg := NewRegistry()
	reg.Entity("User").OnUpdate(RuleFunc(func(_ context.Context, subject Subject) error {
		created, _ := subject.Record.Time("CreatedAt")
		updated, _ := subject.Record.Time("UpdatedAt")
		if updated.Before(created) {
			return FieldError{Field: "UpdatedAt", Message: "must be after CreatedAt"}
		}
		return nil
	}))

	created := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(-time.Hour)
	err := reg.Validate(context.Background(), "User", OpUpdate, Record{
		"CreatedAt": created,
		"UpdatedAt": updated,
	}, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	if fe, ok := err.(Errors); !ok || len(fe) != 1 || fe[0].Field != "UpdatedAt" {
		t.Fatalf("unexpected error: %#v", err)
	}
}
