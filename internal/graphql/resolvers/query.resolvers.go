package resolvers

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/deicod/erm/internal/graphql"
	"github.com/deicod/erm/internal/graphql/dataloaders"
	"github.com/deicod/erm/internal/graphql/relay"
	"github.com/deicod/erm/internal/orm/gen"
)

const (
	defaultPageSize = 20
	cursorPrefix    = "cursor:"
)

func (r *queryResolver) Health(ctx context.Context) (string, error) {
	return "ok", nil
}

func (r *queryResolver) Node(ctx context.Context, id string) (graphql.Node, error) {
	typ, nativeID, err := relay.FromGlobalID(id)
	if err != nil {
		return nil, err
	}
	switch typ {
	case "User":
		var record *gen.User
		if loaders := dataloaders.FromContext(ctx); loaders != nil && loaders.Users != nil {
			record, err = loaders.Users.Load(ctx, nativeID)
			if err != nil {
				return nil, err
			}
		} else {
			record, err = r.ORM.Users().ByID(ctx, nativeID)
			if err != nil {
				return nil, err
			}
		}
		if record == nil {
			return nil, nil
		}
		return toGraphQLUser(record), nil
	default:
		return nil, fmt.Errorf("unknown node type %s", typ)
	}
}

func (r *queryResolver) Users(ctx context.Context, first *int, after *string, last *int, before *string) (*graphql.UserConnection, error) {
	if last != nil || before != nil {
		return nil, fmt.Errorf("backward pagination is not supported")
	}
	limit := defaultPageSize
	if first != nil && *first > 0 {
		limit = *first
	}
	offset := 0
	if after != nil && *after != "" {
		if decoded, err := decodeCursor(*after); err == nil {
			offset = decoded + 1
		}
	}

	total, err := r.ORM.Users().Count(ctx)
	if err != nil {
		return nil, err
	}
	records, err := r.ORM.Users().List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	loaders := dataloaders.FromContext(ctx)
	edges := make([]*graphql.UserEdge, len(records))
	for idx, record := range records {
		cursor := encodeCursor(offset + idx)
		if loaders != nil && loaders.Users != nil && record != nil {
			loaders.Users.Prime(record.ID, record)
		}
		edges[idx] = &graphql.UserEdge{
			Cursor: cursor,
			Node:   toGraphQLUser(record),
		}
	}

	var startCursor, endCursor *string
	if len(edges) > 0 {
		sc := edges[0].Cursor
		ec := edges[len(edges)-1].Cursor
		startCursor = &sc
		endCursor = &ec
	}

	pageInfo := &graphql.PageInfo{
		HasNextPage:     offset+len(edges) < total,
		HasPreviousPage: offset > 0,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}

	return &graphql.UserConnection{
		Edges:      edges,
		PageInfo:   pageInfo,
		TotalCount: total,
	}, nil
}

func toGraphQLUser(record *gen.User) *graphql.User {
	if record == nil {
		return nil
	}
	return &graphql.User{
		ID:        relay.ToGlobalID("User", record.ID),
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func encodeCursor(offset int) string {
	payload := []byte(fmt.Sprintf("%s%d", cursorPrefix, offset))
	return base64.StdEncoding.EncodeToString(payload)
}

func decodeCursor(cursor string) (int, error) {
	raw, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	if len(raw) <= len(cursorPrefix) || string(raw[:len(cursorPrefix)]) != cursorPrefix {
		return 0, fmt.Errorf("invalid cursor")
	}
	return strconv.Atoi(string(raw[len(cursorPrefix):]))
}
