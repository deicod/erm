package resolvers

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
)

const (
	defaultPageSize = 20
	cursorPrefix    = "cursor:"
)

func (r *queryResolver) Health(ctx context.Context) (string, error) {
	return "ok", nil
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
