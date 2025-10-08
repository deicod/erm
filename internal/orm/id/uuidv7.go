package id

import (
	"time"

	"github.com/google/uuid"
)

// NewV7 returns a time-ordered UUID v7 string.
func NewV7() (string, error) {
	u, err := uuid.NewV7()
	if err != nil { return "", err }
	return u.String(), nil
}

// NowUnixMilli exists to aid testability.
func NowUnixMilli() int64 { return time.Now().UnixMilli() }
