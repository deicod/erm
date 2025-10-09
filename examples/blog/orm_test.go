package blog_test

import (
	"context"
	"testing"
	"time"

	pgxmock "github.com/pashagolub/pgxmock/v4"

	"github.com/deicod/erm/internal/orm/gen"
	ermtesting "github.com/deicod/erm/internal/testing"
)

func TestUserORMCRUDFlow(t *testing.T) {
	sandbox := ermtesting.NewPostgresSandbox(t)
	ctx := context.Background()
	client := sandbox.ORM(t)
	mock := sandbox.Mock()

	createdAt := time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)

	mock.ExpectQuery("INSERT INTO users (id, created_at, updated_at) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at").
		WithArgs("user-1", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("user-1", createdAt, updatedAt))

	user, err := client.Users().Create(ctx, &gen.User{ID: "user-1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if user.ID != "user-1" {
		t.Fatalf("unexpected id: %s", user.ID)
	}

	mock.ExpectQuery("SELECT id, created_at, updated_at FROM users WHERE id = $1").
		WithArgs("user-1").
		WillReturnRows(mock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("user-1", createdAt, updatedAt))

	fetched, err := client.Users().ByID(ctx, "user-1")
	if err != nil {
		t.Fatalf("by id: %v", err)
	}
	if fetched == nil || fetched.ID != "user-1" {
		t.Fatalf("unexpected record: %+v", fetched)
	}

	mock.ExpectQuery("SELECT id, created_at, updated_at FROM users ORDER BY id LIMIT $1 OFFSET $2").
		WithArgs(5, 0).
		WillReturnRows(mock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow("user-1", createdAt, updatedAt).
			AddRow("user-2", createdAt.Add(time.Minute), updatedAt.Add(time.Minute)))

	list, err := client.Users().List(ctx, 5, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 records, got %d", len(list))
	}

	mock.ExpectQuery("UPDATE users SET updated_at = $1 WHERE id = $2 RETURNING id, created_at, updated_at").
		WithArgs(pgxmock.AnyArg(), "user-1").
		WillReturnRows(mock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("user-1", createdAt, updatedAt.Add(2*time.Hour)))

	updated, err := client.Users().Update(ctx, &gen.User{ID: "user-1"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.UpdatedAt.Before(updatedAt) {
		t.Fatalf("expected updated timestamp to advance")
	}

	mock.ExpectQuery("SELECT id, created_at, updated_at FROM users WHERE id = $1 LIMIT $2").
		WithArgs("user-1", 1).
		WillReturnRows(mock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("user-1", createdAt, updatedAt))

	all, err := client.Users().Query().WhereIDEq("user-1").Limit(1).All(ctx)
	if err != nil {
		t.Fatalf("query all: %v", err)
	}
	if len(all) != 1 || all[0].ID != "user-1" {
		t.Fatalf("unexpected query result: %+v", all)
	}

	mock.ExpectQuery("SELECT COUNT(*) FROM users WHERE id = $1").
		WithArgs("user-1").
		WillReturnRows(mock.NewRows([]string{"count"}).AddRow(1))

	count, err := client.Users().Query().WhereIDEq("user-1").Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("unexpected count: %d", count)
	}

	mock.ExpectExec("DELETE FROM users WHERE id = $1").
		WithArgs("user-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	if err := client.Users().Delete(ctx, "user-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	sandbox.ExpectationsWereMet(t)
}
