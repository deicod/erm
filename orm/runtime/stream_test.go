package runtime

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestStream(t *testing.T) {
	rows := newSliceRows([]any{1}, []any{2})
	stream := NewStream[int](rows, func(r pgx.Rows) (int, error) {
		var v int
		if err := r.Scan(&v); err != nil {
			return 0, err
		}
		return v, nil
	})
	defer stream.Close()
	var values []int
	for stream.Next() {
		values = append(values, stream.Item())
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream err: %v", err)
	}
	if len(values) != 2 || values[0] != 1 || values[1] != 2 {
		t.Fatalf("unexpected values: %#v", values)
	}
}

func TestStreamScanError(t *testing.T) {
	rows := newSliceRows([]any{1})
	stream := NewStream[int](rows, func(r pgx.Rows) (int, error) {
		return 0, errors.New("boom")
	})
	if stream.Next() {
		t.Fatalf("expected Next to return false on error")
	}
	if err := stream.Err(); err == nil {
		t.Fatalf("expected error")
	}
}

type sliceRows struct {
	data   [][]any
	idx    int
	closed bool
	err    error
}

func newSliceRows(rows ...[]any) *sliceRows {
	return &sliceRows{data: rows}
}

func (r *sliceRows) Close() { r.closed = true }

func (r *sliceRows) Err() error { return r.err }

func (r *sliceRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *sliceRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *sliceRows) Next() bool {
	if r.closed || r.idx >= len(r.data) {
		r.closed = true
		return false
	}
	r.idx++
	return true
}

func (r *sliceRows) Scan(dest ...any) error {
	if r.idx == 0 || r.idx > len(r.data) {
		return errors.New("no row selected")
	}
	row := r.data[r.idx-1]
	if len(dest) != len(row) {
		return errors.New("destination size mismatch")
	}
	for i, d := range dest {
		switch ptr := d.(type) {
		case *int:
			if val, ok := row[i].(int); ok {
				*ptr = val
			} else {
				return errors.New("unexpected type")
			}
		default:
			return errors.New("unsupported destination type")
		}
	}
	return nil
}

func (r *sliceRows) Values() ([]any, error) {
	if r.idx == 0 || r.idx > len(r.data) {
		return nil, errors.New("no row selected")
	}
	row := r.data[r.idx-1]
	out := make([]any, len(row))
	copy(out, row)
	return out, nil
}

func (r *sliceRows) RawValues() [][]byte { return nil }

func (r *sliceRows) Conn() *pgx.Conn { return nil }
