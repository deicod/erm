package runtime

import (
	"sync"

	"github.com/jackc/pgx/v5"
)

type Stream[T any] struct {
	rows   pgx.Rows
	scan   func(pgx.Rows) (T, error)
	once   sync.Once
	err    error
	closed bool
	item   T
}

func NewStream[T any](rows pgx.Rows, scan func(pgx.Rows) (T, error)) *Stream[T] {
	return &Stream[T]{rows: rows, scan: scan}
}

func (s *Stream[T]) Next() bool {
	if s == nil || s.closed {
		return false
	}
	if s.err != nil {
		return false
	}
	if !s.rows.Next() {
		s.err = s.rows.Err()
		s.Close()
		var zero T
		s.item = zero
		return false
	}
	item, err := s.scan(s.rows)
	if err != nil {
		s.err = err
		s.Close()
		var zero T
		s.item = zero
		return false
	}
	s.item = item
	return true
}

func (s *Stream[T]) Item() T {
	if s == nil {
		var zero T
		return zero
	}
	return s.item
}

func (s *Stream[T]) Err() error {
	if s == nil {
		return nil
	}
	return s.err
}

func (s *Stream[T]) Close() error {
	if s == nil {
		return nil
	}
	s.once.Do(func() {
		s.closed = true
		if s.rows != nil {
			s.rows.Close()
		}
	})
	return s.err
}
