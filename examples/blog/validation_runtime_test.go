package blog_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	pgxmock "github.com/pashagolub/pgxmock/v4"

	"github.com/deicod/erm/internal/orm/gen"
	"github.com/deicod/erm/internal/orm/runtime/validation"
	ermtesting "github.com/deicod/erm/internal/testing"
)

func TestUserValidationRegistry(t *testing.T) {
	sandbox := ermtesting.NewPostgresSandbox(t)
	ctx := context.Background()
	client := sandbox.ORM(t)
	mock := sandbox.Mock()

	gen.ValidationRegistry = validation.NewRegistry()
	t.Cleanup(func() { gen.ValidationRegistry = validation.NewRegistry() })

	pattern := regexp.MustCompile(`^usr_[a-z0-9]+$`)
	gen.ValidationRegistry.Entity("User").
		OnCreate(validation.String("ID").Required().Matches(pattern).Rule()).
		OnUpdate(validation.RuleFunc(func(_ context.Context, subject validation.Subject) error {
			created, _ := subject.Record.Time("CreatedAt")
			updated, _ := subject.Record.Time("UpdatedAt")
			if updated.Before(created) {
				return validation.FieldError{Field: "UpdatedAt", Message: "must be after CreatedAt"}
			}
			return nil
		}))

	if _, err := client.Users().Create(ctx, &gen.User{ID: "user-1"}); err == nil {
		t.Fatalf("expected create validation error")
	}

	createdAt := time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	mock.ExpectQuery("INSERT INTO users (id, created_at, updated_at) VALUES ($1, $2, $3) RETURNING id, slug, created_at, updated_at").
		WithArgs("usr_good", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"id", "slug", "created_at", "updated_at"}).AddRow("usr_good", "usr_good", createdAt, updatedAt))

	if _, err := client.Users().Create(ctx, &gen.User{ID: "usr_good"}); err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	future := time.Now().Add(time.Hour)
	if _, err := client.Users().Update(ctx, &gen.User{ID: "usr_good", CreatedAt: future}); err == nil {
		t.Fatalf("expected update validation error")
	}

	past := time.Now().Add(-time.Hour)
	newUpdated := past.Add(2 * time.Hour)
	mock.ExpectQuery("UPDATE users SET updated_at = $1 WHERE id = $2 RETURNING id, slug, created_at, updated_at").
		WithArgs(pgxmock.AnyArg(), "usr_good").
		WillReturnRows(mock.NewRows([]string{"id", "slug", "created_at", "updated_at"}).AddRow("usr_good", "usr_good", past, newUpdated))

	if _, err := client.Users().Update(ctx, &gen.User{ID: "usr_good", CreatedAt: past}); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}

	sandbox.ExpectationsWereMet(t)
}
