package runtime

import (
	"fmt"
	"strconv"
	"strings"
)

type Operator string

const (
	OpEqual       Operator = "="
	OpNotEqual    Operator = "<>"
	OpGreaterThan Operator = ">"
	OpLessThan    Operator = "<"
	OpGTE         Operator = ">="
	OpLTE         Operator = "<="
	OpILike       Operator = "ILIKE"
)

type SortDirection string

const (
	SortAsc  SortDirection = "ASC"
	SortDesc SortDirection = "DESC"
)

type Predicate struct {
	Column   string
	Operator Operator
	Value    any
}

type Order struct {
	Column    string
	Direction SortDirection
}

type SelectSpec struct {
	Table      string
	Columns    []string
	Predicates []Predicate
	Orders     []Order
	Limit      int
	Offset     int
}

type AggregateFunc string

const (
	AggCount AggregateFunc = "COUNT"
	AggSum   AggregateFunc = "SUM"
	AggAvg   AggregateFunc = "AVG"
	AggMin   AggregateFunc = "MIN"
	AggMax   AggregateFunc = "MAX"
)

type Aggregate struct {
	Func   AggregateFunc
	Column string
}

type AggregateSpec struct {
	Table      string
	Predicates []Predicate
	Aggregate  Aggregate
}

func BuildSelectSQL(spec SelectSpec) (string, []any) {
	columns := spec.Columns
	if len(columns) == 0 {
		columns = []string{"*"}
	}

	var sb strings.Builder
	sb.WriteString("SELECT ")
	sb.WriteString(strings.Join(columns, ", "))
	sb.WriteString(" FROM ")
	sb.WriteString(spec.Table)

	args := make([]any, 0, len(spec.Predicates)+2)
	param := 1

	if len(spec.Predicates) > 0 {
		sb.WriteString(" WHERE ")
		for i, pred := range spec.Predicates {
			if i > 0 {
				sb.WriteString(" AND ")
			}
			sb.WriteString(pred.Column)
			sb.WriteByte(' ')
			sb.WriteString(string(pred.Operator))
			sb.WriteByte(' ')
			sb.WriteString("$")
			sb.WriteString(strconv.Itoa(param))
			args = append(args, pred.Value)
			param++
		}
	}

	if len(spec.Orders) > 0 {
		sb.WriteString(" ORDER BY ")
		for i, order := range spec.Orders {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(order.Column)
			sb.WriteByte(' ')
			sb.WriteString(string(order.Direction))
		}
	}

	if spec.Limit > 0 {
		sb.WriteString(" LIMIT $")
		sb.WriteString(strconv.Itoa(param))
		args = append(args, spec.Limit)
		param++
	}

	if spec.Offset > 0 {
		sb.WriteString(" OFFSET $")
		sb.WriteString(strconv.Itoa(param))
		args = append(args, spec.Offset)
	}

	return sb.String(), args
}

func BuildAggregateSQL(spec AggregateSpec) (string, []any) {
	column := spec.Aggregate.Column
	if column == "" {
		column = "*"
	}

	var sb strings.Builder
	sb.WriteString("SELECT ")
	sb.WriteString(string(spec.Aggregate.Func))
	sb.WriteByte('(')
	sb.WriteString(column)
	sb.WriteString(") FROM ")
	sb.WriteString(spec.Table)

	args := make([]any, 0, len(spec.Predicates))
	param := 1

	if len(spec.Predicates) > 0 {
		sb.WriteString(" WHERE ")
		for i, pred := range spec.Predicates {
			if i > 0 {
				sb.WriteString(" AND ")
			}
			sb.WriteString(pred.Column)
			sb.WriteByte(' ')
			sb.WriteString(string(pred.Operator))
			sb.WriteByte(' ')
			sb.WriteString("$")
			sb.WriteString(strconv.Itoa(param))
			args = append(args, pred.Value)
			param++
		}
	}

	return sb.String(), args
}

func (spec AggregateSpec) Validate() error {
	if spec.Table == "" {
		return fmt.Errorf("table name is required")
	}
	if spec.Aggregate.Func == "" {
		return fmt.Errorf("aggregate function is required")
	}
	return nil
}
