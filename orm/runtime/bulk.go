package runtime

import (
	"fmt"
	"strings"
)

type BulkInsertSpec struct {
	Table     string
	Columns   []string
	Returning []string
	Rows      [][]any
}

func BuildBulkInsertSQL(spec BulkInsertSpec) (string, []any, error) {
	if spec.Table == "" {
		return "", nil, fmt.Errorf("table is required")
	}
	if len(spec.Columns) == 0 {
		return "", nil, fmt.Errorf("columns are required")
	}
	if len(spec.Rows) == 0 {
		return "", nil, fmt.Errorf("at least one row is required")
	}
	args := make([]any, 0, len(spec.Rows)*len(spec.Columns))
	values := make([]string, len(spec.Rows))
	param := 1
	for i, row := range spec.Rows {
		if len(row) != len(spec.Columns) {
			return "", nil, fmt.Errorf("row %d has %d values, expected %d", i, len(row), len(spec.Columns))
		}
		placeholders := make([]string, len(spec.Columns))
		for j, value := range row {
			placeholders[j] = fmt.Sprintf("$%d", param)
			args = append(args, value)
			param++
		}
		values[i] = fmt.Sprintf("(%s)", strings.Join(placeholders, ", "))
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", spec.Table, strings.Join(spec.Columns, ", "), strings.Join(values, ", "))
	if len(spec.Returning) > 0 {
		sql += " RETURNING " + strings.Join(spec.Returning, ", ")
	}
	return sql, args, nil
}

type BulkUpdateRow struct {
	Primary any
	Values  []any
}

type BulkUpdateSpec struct {
	Table         string
	PrimaryColumn string
	Columns       []string
	Returning     []string
	Rows          []BulkUpdateRow
}

func BuildBulkUpdateSQL(spec BulkUpdateSpec) (string, []any, error) {
	if spec.Table == "" {
		return "", nil, fmt.Errorf("table is required")
	}
	if spec.PrimaryColumn == "" {
		return "", nil, fmt.Errorf("primary column is required")
	}
	if len(spec.Columns) == 0 {
		return "", nil, fmt.Errorf("columns are required")
	}
	if len(spec.Rows) == 0 {
		return "", nil, fmt.Errorf("at least one row is required")
	}
	cols := append([]string{spec.PrimaryColumn}, spec.Columns...)
	assignments := make([]string, len(spec.Columns))
	for i, col := range spec.Columns {
		assignments[i] = fmt.Sprintf("%s = data.%s", col, col)
	}
	args := make([]any, 0, len(spec.Rows)*(len(spec.Columns)+1))
	values := make([]string, len(spec.Rows))
	param := 1
	for i, row := range spec.Rows {
		if len(row.Values) != len(spec.Columns) {
			return "", nil, fmt.Errorf("row %d has %d values, expected %d", i, len(row.Values), len(spec.Columns))
		}
		placeholders := make([]string, len(cols))
		placeholders[0] = fmt.Sprintf("$%d", param)
		args = append(args, row.Primary)
		param++
		for j, value := range row.Values {
			placeholders[j+1] = fmt.Sprintf("$%d", param)
			args = append(args, value)
			param++
		}
		values[i] = fmt.Sprintf("(%s)", strings.Join(placeholders, ", "))
	}
	sql := fmt.Sprintf("WITH data(%s) AS (VALUES %s) UPDATE %s AS t SET %s FROM data WHERE t.%s = data.%s",
		strings.Join(cols, ", "),
		strings.Join(values, ", "),
		spec.Table,
		strings.Join(assignments, ", "),
		spec.PrimaryColumn,
		spec.PrimaryColumn,
	)
	if len(spec.Returning) > 0 {
		sql += " RETURNING " + strings.Join(spec.Returning, ", ")
	}
	return sql, args, nil
}

type BulkDeleteSpec struct {
	Table         string
	PrimaryColumn string
	IDs           []any
}

func BuildBulkDeleteSQL(spec BulkDeleteSpec) (string, []any, error) {
	if spec.Table == "" {
		return "", nil, fmt.Errorf("table is required")
	}
	if spec.PrimaryColumn == "" {
		return "", nil, fmt.Errorf("primary column is required")
	}
	if len(spec.IDs) == 0 {
		return "", nil, fmt.Errorf("at least one id is required")
	}
	placeholders := make([]string, len(spec.IDs))
	args := make([]any, len(spec.IDs))
	for i, id := range spec.IDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	sql := fmt.Sprintf("DELETE FROM %s WHERE %s IN (%s)", spec.Table, spec.PrimaryColumn, strings.Join(placeholders, ", "))
	return sql, args, nil
}
