package blog_test

import (
	"context"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	pgxmock "github.com/pashagolub/pgxmock/v4"

	"github.com/deicod/erm/internal/graphql/relay"
	ermtesting "github.com/deicod/erm/internal/testing"
)

func TestGraphQLUserResolvers(t *testing.T) {
	sandbox := ermtesting.NewPostgresSandbox(t)
	ctx := context.Background()
	orm := sandbox.ORM(t)
	harness := ermtesting.NewGraphQLHarness(t, ermtesting.GraphQLHarnessOptions{ORM: orm})
	mock := sandbox.Mock()

	createdAt := time.Date(2024, time.January, 2, 15, 30, 0, 0, time.UTC)
	updatedAt := createdAt.Add(30 * time.Minute)
	globalID := relay.ToGlobalID("User", "user-graphql")

	mock.ExpectQuery("INSERT INTO users (id, created_at, updated_at) VALUES ($1, $2, $3) RETURNING id, slug, created_at, updated_at").
		WithArgs("user-graphql", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"id", "slug", "created_at", "updated_at"}).AddRow("user-graphql", "user-graphql", createdAt, updatedAt))

	var createResp struct {
		CreateUser struct {
			User struct {
				ID string
			}
		}
	}
	harness.MustExec(t, ctx, `mutation($input: CreateUserInput!) {
                createUser(input: $input) {
                        user { id }
                }
        }`, &createResp, client.Var("input", map[string]any{"id": "user-graphql"}))
	if createResp.CreateUser.User.ID != globalID {
		t.Fatalf("unexpected global id: %s", createResp.CreateUser.User.ID)
	}

	mock.ExpectQuery("SELECT id, slug, created_at, updated_at FROM users WHERE id = $1").
		WithArgs("user-graphql").
		WillReturnRows(mock.NewRows([]string{"id", "slug", "created_at", "updated_at"}).AddRow("user-graphql", "user-graphql", createdAt, updatedAt))

	var userResp struct {
		User *struct {
			ID string
		}
	}
	harness.MustExec(t, ctx, `query($id: ID!) {
                user(id: $id) { id }
        }`, &userResp, client.Var("id", globalID))
	if userResp.User == nil || userResp.User.ID != globalID {
		t.Fatalf("unexpected user payload: %+v", userResp.User)
	}

	mock.ExpectQuery("SELECT COUNT(*) FROM users").
		WillReturnRows(mock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT id, slug, created_at, updated_at FROM users ORDER BY id LIMIT $1 OFFSET $2").
		WithArgs(2, 0).
		WillReturnRows(mock.NewRows([]string{"id", "slug", "created_at", "updated_at"}).
			AddRow("user-graphql", "user-graphql", createdAt, updatedAt).
			AddRow("user-other", "user-other", createdAt.Add(time.Minute), updatedAt.Add(time.Minute)))

	var listResp struct {
		Users struct {
			TotalCount int
			Edges      []struct {
				Node struct {
					ID string
				}
			}
		}
	}
	harness.MustExec(t, ctx, `query($first: Int) {
                users(first: $first) {
                        totalCount
                        edges { node { id } }
                }
        }`, &listResp, client.Var("first", 2))
	if listResp.Users.TotalCount != 1 {
		t.Fatalf("unexpected total count: %d", listResp.Users.TotalCount)
	}
	if len(listResp.Users.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(listResp.Users.Edges))
	}

	mock.ExpectQuery("UPDATE users SET updated_at = $1 WHERE id = $2 RETURNING id, slug, created_at, updated_at").
		WithArgs(pgxmock.AnyArg(), "user-graphql").
		WillReturnRows(mock.NewRows([]string{"id", "slug", "created_at", "updated_at"}).AddRow("user-graphql", "user-graphql", createdAt, updatedAt.Add(time.Hour)))

	var updateResp struct {
		UpdateUser struct {
			User struct {
				ID string
			}
		}
	}
	harness.MustExec(t, ctx, `mutation($input: UpdateUserInput!) {
                updateUser(input: $input) {
                        user { id }
                }
        }`, &updateResp, client.Var("input", map[string]any{"id": globalID}))
	if updateResp.UpdateUser.User.ID != globalID {
		t.Fatalf("unexpected id after update: %s", updateResp.UpdateUser.User.ID)
	}

	mock.ExpectExec("DELETE FROM users WHERE id = $1").
		WithArgs("user-graphql").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	var deleteResp struct {
		DeleteUser struct {
			DeletedUserID string
		}
	}
	harness.MustExec(t, ctx, `mutation($input: DeleteUserInput!) {
                deleteUser(input: $input) {
                        deletedUserID
                }
        }`, &deleteResp, client.Var("input", map[string]any{"id": globalID}))
	if deleteResp.DeleteUser.DeletedUserID != globalID {
		t.Fatalf("unexpected deleted id: %s", deleteResp.DeleteUser.DeletedUserID)
	}

	sandbox.ExpectationsWereMet(t)
}
