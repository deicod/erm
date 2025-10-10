package generator

import (
	"fmt"
	"sort"
	"strings"
)

type OperationKind string

const (
	OpCreateExtension  OperationKind = "create_extension"
	OpDropExtension    OperationKind = "drop_extension"
	OpCreateTable      OperationKind = "create_table"
	OpDropTable        OperationKind = "drop_table"
	OpAddColumn        OperationKind = "add_column"
	OpDropColumn       OperationKind = "drop_column"
	OpAlterColumn      OperationKind = "alter_column"
	OpAddForeignKey    OperationKind = "add_foreign_key"
	OpDropForeignKey   OperationKind = "drop_foreign_key"
	OpAddIndex         OperationKind = "add_index"
	OpDropIndex        OperationKind = "drop_index"
	OpCreateHypertable OperationKind = "create_hypertable"
	OpDropHypertable   OperationKind = "drop_hypertable"
)

type Operation struct {
	Kind   OperationKind
	Target string
	SQL    string
}

func diffSchema(prev, next SchemaSnapshot) []Operation {
	ops := make([]Operation, 0)
	ops = append(ops, diffExtensions(prev.Extensions, next.Extensions)...)
	ops = append(ops, diffTables(prev.Tables, next.Tables)...)
	return ops
}

func diffExtensions(prev, next []string) []Operation {
	ops := make([]Operation, 0)
	prevSet := make(map[string]struct{}, len(prev))
	nextSet := make(map[string]struct{}, len(next))
	for _, ext := range prev {
		prevSet[ext] = struct{}{}
	}
	for _, ext := range next {
		nextSet[ext] = struct{}{}
	}
	var drops []string
	for ext := range prevSet {
		if _, ok := nextSet[ext]; !ok {
			drops = append(drops, ext)
		}
	}
	sort.Strings(drops)
	for _, ext := range drops {
		ops = append(ops, Operation{Kind: OpDropExtension, Target: ext, SQL: fmt.Sprintf("DROP EXTENSION IF EXISTS %s;", ext)})
	}
	var adds []string
	for ext := range nextSet {
		if _, ok := prevSet[ext]; !ok {
			adds = append(adds, ext)
		}
	}
	sort.Strings(adds)
	for _, ext := range adds {
		ops = append(ops, Operation{Kind: OpCreateExtension, Target: ext, SQL: fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s;", ext)})
	}
	return ops
}

func diffTables(prev, next []TableSnapshot) []Operation {
	ops := make([]Operation, 0)
	prevMap := make(map[string]TableSnapshot, len(prev))
	nextMap := make(map[string]TableSnapshot, len(next))
	for _, tbl := range prev {
		prevMap[tbl.Name] = tbl
	}
	for _, tbl := range next {
		nextMap[tbl.Name] = tbl
	}

	dropNames := make([]string, 0)
	for name := range prevMap {
		if _, ok := nextMap[name]; !ok {
			dropNames = append(dropNames, name)
		}
	}
	sort.Slice(dropNames, func(i, j int) bool {
		a := prevMap[dropNames[i]]
		b := prevMap[dropNames[j]]
		if a.IsJoinTable == b.IsJoinTable {
			return dropNames[i] < dropNames[j]
		}
		if a.IsJoinTable {
			return false
		}
		return true
	})
	for _, name := range dropNames {
		ops = append(ops, Operation{Kind: OpDropTable, Target: name, SQL: fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", name)})
	}

	modifyNames := make([]string, 0)
	for name := range prevMap {
		if _, ok := nextMap[name]; ok {
			modifyNames = append(modifyNames, name)
		}
	}
	sort.Strings(modifyNames)
	for _, name := range modifyNames {
		ops = append(ops, diffTable(prevMap[name], nextMap[name])...)
	}

	addNames := make([]string, 0)
	for name := range nextMap {
		if _, ok := prevMap[name]; !ok {
			addNames = append(addNames, name)
		}
	}
	sort.Slice(addNames, func(i, j int) bool {
		a := nextMap[addNames[i]]
		b := nextMap[addNames[j]]
		if a.IsJoinTable == b.IsJoinTable {
			return addNames[i] < addNames[j]
		}
		if a.IsJoinTable {
			return false
		}
		return true
	})
	for _, name := range addNames {
		ops = append(ops, createTableOps(nextMap[name])...)
	}

	return ops
}

func diffTable(prev, next TableSnapshot) []Operation {
	ops := make([]Operation, 0)
	if prev.Name != next.Name {
		ops = append(ops, Operation{Kind: OpDropTable, Target: prev.Name, SQL: fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", prev.Name)})
		ops = append(ops, createTableOps(next)...)
		return ops
	}

	dropIndexes, addIndexes := diffIndexes(next.Name, prev.Indexes, next.Indexes)
	ops = append(ops, dropIndexes...)

	dropFKs, addFKs := diffForeignKeys(prev.ForeignKeys, next.ForeignKeys, next.Name)
	ops = append(ops, dropFKs...)

	columnOps := diffColumns(prev, next)
	ops = append(ops, columnOps...)

	ops = append(ops, addIndexes...)
	ops = append(ops, addFKs...)

	ops = append(ops, diffHypertable(prev, next)...)

	return ops
}

func diffColumns(prev, next TableSnapshot) []Operation {
	ops := make([]Operation, 0)
	prevMap := make(map[string]ColumnSnapshot, len(prev.Columns))
	nextMap := make(map[string]ColumnSnapshot, len(next.Columns))
	for _, col := range prev.Columns {
		prevMap[col.Name] = col
	}
	for _, col := range next.Columns {
		nextMap[col.Name] = col
	}

	// Drop columns after dropping indexes/FKs
	var dropCols []string
	for name := range prevMap {
		if _, ok := nextMap[name]; !ok {
			dropCols = append(dropCols, name)
		}
	}
	sort.Strings(dropCols)

	// Modify shared columns
	var shared []string
	for name := range prevMap {
		if _, ok := nextMap[name]; ok {
			shared = append(shared, name)
		}
	}
	sort.Strings(shared)
	for _, name := range shared {
		prevCol := prevMap[name]
		nextCol := nextMap[name]
		ops = append(ops, diffColumn(prev.Name, prevCol, nextCol)...)
	}

	// Add columns
	var addCols []string
	for name := range nextMap {
		if _, ok := prevMap[name]; !ok {
			addCols = append(addCols, name)
		}
	}
	sort.Strings(addCols)
	for _, name := range addCols {
		col := nextMap[name]
		ops = append(ops, Operation{
			Kind:   OpAddColumn,
			Target: fmt.Sprintf("%s.%s", next.Name, name),
			SQL:    fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", next.Name, renderColumnDefinition(col)),
		})
	}

	for _, name := range dropCols {
		ops = append(ops, Operation{
			Kind:   OpDropColumn,
			Target: fmt.Sprintf("%s.%s", prev.Name, name),
			SQL:    fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s CASCADE;", prev.Name, name),
		})
	}

	return ops
}

func diffColumn(table string, prev, next ColumnSnapshot) []Operation {
	ops := make([]Operation, 0)
	if prev.GeneratedExpr != "" || next.GeneratedExpr != "" || prev.ReadOnly || next.ReadOnly {
		if prev.GeneratedExpr != next.GeneratedExpr || prev.Type != next.Type || prev.Nullable != next.Nullable || prev.Unique != next.Unique || prev.ReadOnly != next.ReadOnly {
			drop := Operation{
				Kind:   OpDropColumn,
				Target: fmt.Sprintf("%s.%s", table, prev.Name),
				SQL:    fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", table, prev.Name),
			}
			add := Operation{
				Kind:   OpAddColumn,
				Target: fmt.Sprintf("%s.%s", table, next.Name),
				SQL:    fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", table, renderColumnDefinition(next)),
			}
			return []Operation{drop, add}
		}
		return ops
	}
	if prev.Type != next.Type {
		ops = append(ops, Operation{
			Kind:   OpAlterColumn,
			Target: fmt.Sprintf("%s.%s", table, prev.Name),
			SQL:    fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;", table, prev.Name, next.Type),
		})
	}
	if prev.Nullable != next.Nullable {
		clause := "SET NOT NULL"
		if next.Nullable {
			clause = "DROP NOT NULL"
		}
		ops = append(ops, Operation{
			Kind:   OpAlterColumn,
			Target: fmt.Sprintf("%s.%s", table, prev.Name),
			SQL:    fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s %s;", table, prev.Name, clause),
		})
	}
	prevDefault, prevHas := columnDefaultExpr(prev)
	nextDefault, nextHas := columnDefaultExpr(next)
	if prevHas != nextHas || prevDefault != nextDefault {
		if nextHas {
			ops = append(ops, Operation{
				Kind:   OpAlterColumn,
				Target: fmt.Sprintf("%s.%s", table, prev.Name),
				SQL:    fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;", table, prev.Name, nextDefault),
			})
		} else {
			ops = append(ops, Operation{
				Kind:   OpAlterColumn,
				Target: fmt.Sprintf("%s.%s", table, prev.Name),
				SQL:    fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;", table, prev.Name),
			})
		}
	}
	return ops
}

func renderColumnDefinition(col ColumnSnapshot) string {
	parts := []string{fmt.Sprintf("%s %s", col.Name, col.Type)}
	if !col.Nullable {
		parts = append(parts, "NOT NULL")
	}
	if expr, ok := columnDefaultExpr(col); ok {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", expr))
	}
	if col.Unique {
		parts = append(parts, "UNIQUE")
	}
	return strings.Join(parts, " ")
}

func diffIndexes(table string, prev, next []IndexSnapshot) (drops, adds []Operation) {
	prevMap := make(map[string]IndexSnapshot, len(prev))
	nextMap := make(map[string]IndexSnapshot, len(next))
	for _, idx := range prev {
		prevMap[idx.Name] = idx
	}
	for _, idx := range next {
		nextMap[idx.Name] = idx
	}

	var dropNames []string
	for name := range prevMap {
		if _, ok := nextMap[name]; !ok {
			dropNames = append(dropNames, name)
		} else if !indexEqual(prevMap[name], nextMap[name]) {
			dropNames = append(dropNames, name)
		}
	}
	sort.Strings(dropNames)
	for _, name := range dropNames {
		drops = append(drops, Operation{Kind: OpDropIndex, Target: name, SQL: fmt.Sprintf("DROP INDEX IF EXISTS %s;", name)})
	}

	var addNames []string
	for name := range nextMap {
		if _, ok := prevMap[name]; !ok {
			addNames = append(addNames, name)
		} else if !indexEqual(prevMap[name], nextMap[name]) {
			addNames = append(addNames, name)
		}
	}
	sort.Strings(addNames)
	for _, name := range addNames {
		idx := nextMap[name]
		adds = append(adds, Operation{Kind: OpAddIndex, Target: idx.Name, SQL: renderCreateIndex(table, idx)})
	}
	return drops, adds
}

func indexEqual(a, b IndexSnapshot) bool {
	if a.Name != b.Name || a.Unique != b.Unique || a.Method != b.Method || a.Where != b.Where || a.NullsNotDistinct != b.NullsNotDistinct {
		return false
	}
	if len(a.Columns) != len(b.Columns) {
		return false
	}
	for i := range a.Columns {
		if a.Columns[i] != b.Columns[i] {
			return false
		}
	}
	return true
}

func diffForeignKeys(prev, next []ForeignKeySnapshot, table string) (drops, adds []Operation) {
	prevMap := make(map[string]ForeignKeySnapshot, len(prev))
	nextMap := make(map[string]ForeignKeySnapshot, len(next))
	for _, fk := range prev {
		prevMap[fk.Constraint] = fk
	}
	for _, fk := range next {
		nextMap[fk.Constraint] = fk
	}

	var dropNames []string
	for name := range prevMap {
		if _, ok := nextMap[name]; !ok {
			dropNames = append(dropNames, name)
		} else if !foreignKeyEqual(prevMap[name], nextMap[name]) {
			dropNames = append(dropNames, name)
		}
	}
	sort.Strings(dropNames)
	for _, name := range dropNames {
		drops = append(drops, Operation{
			Kind:   OpDropForeignKey,
			Target: fmt.Sprintf("%s.%s", table, name),
			SQL:    fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;", table, name),
		})
	}

	var addNames []string
	for name := range nextMap {
		if _, ok := prevMap[name]; !ok {
			addNames = append(addNames, name)
		} else if !foreignKeyEqual(prevMap[name], nextMap[name]) {
			addNames = append(addNames, name)
		}
	}
	sort.Strings(addNames)
	for _, name := range addNames {
		fk := nextMap[name]
		clause := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)", table, fk.Constraint, fk.Column, fk.TargetTable, fk.TargetColumn)
		if fk.OnDelete != "" {
			clause += fmt.Sprintf(" ON DELETE %s", fk.OnDelete)
		}
		if fk.OnUpdate != "" {
			clause += fmt.Sprintf(" ON UPDATE %s", fk.OnUpdate)
		}
		adds = append(adds, Operation{
			Kind:   OpAddForeignKey,
			Target: fmt.Sprintf("%s.%s", table, name),
			SQL:    clause + ";",
		})
	}

	return drops, adds
}

func foreignKeyEqual(a, b ForeignKeySnapshot) bool {
	return a.Column == b.Column && a.TargetTable == b.TargetTable && a.TargetColumn == b.TargetColumn && a.Constraint == b.Constraint && a.OnDelete == b.OnDelete && a.OnUpdate == b.OnUpdate
}

func diffHypertable(prev, next TableSnapshot) []Operation {
	ops := make([]Operation, 0)
	if prev.HypertableColumn == next.HypertableColumn {
		return ops
	}
	if prev.HypertableColumn != "" {
		ops = append(ops, Operation{
			Kind:   OpDropHypertable,
			Target: next.Name,
			SQL:    fmt.Sprintf("SELECT remove_hypertable('%s');", next.Name),
		})
	}
	if next.HypertableColumn != "" {
		ops = append(ops, Operation{
			Kind:   OpCreateHypertable,
			Target: next.Name,
			SQL:    fmt.Sprintf("SELECT create_hypertable('%s', '%s', if_not_exists => TRUE);", next.Name, next.HypertableColumn),
		})
	}
	return ops
}

func createTableOps(table TableSnapshot) []Operation {
	ops := make([]Operation, 0)
	defs := make([]string, 0, len(table.Columns)+1)
	for _, col := range table.Columns {
		defs = append(defs, fmt.Sprintf("    %s", renderColumnDefinition(col)))
	}
	if len(table.PrimaryKey) > 0 {
		defs = append(defs, fmt.Sprintf("    PRIMARY KEY (%s)", strings.Join(table.PrimaryKey, ", ")))
	}
	stmt := fmt.Sprintf("CREATE TABLE %s (\n%s\n);", table.Name, strings.Join(defs, ",\n"))
	ops = append(ops, Operation{Kind: OpCreateTable, Target: table.Name, SQL: stmt})

	for _, idx := range table.Indexes {
		ops = append(ops, Operation{Kind: OpAddIndex, Target: idx.Name, SQL: renderCreateIndex(table.Name, idx)})
	}
	for _, fk := range table.ForeignKeys {
		ops = append(ops, Operation{Kind: OpAddForeignKey, Target: fmt.Sprintf("%s.%s", table.Name, fk.Constraint), SQL: fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s);", table.Name, fk.Constraint, fk.Column, fk.TargetTable, fk.TargetColumn)})
	}
	if table.HypertableColumn != "" {
		ops = append(ops, Operation{Kind: OpCreateHypertable, Target: table.Name, SQL: fmt.Sprintf("SELECT create_hypertable('%s', '%s', if_not_exists => TRUE);", table.Name, table.HypertableColumn)})
	}
	return ops
}

func renderCreateIndex(table string, idx IndexSnapshot) string {
	parts := []string{"CREATE"}
	if idx.Unique {
		parts = append(parts, "UNIQUE")
	}
	parts = append(parts, "INDEX IF NOT EXISTS", idx.Name, "ON", table)
	if idx.Method != "" {
		parts = append(parts, "USING", idx.Method)
	}
	cols := fmt.Sprintf("(%s)", strings.Join(idx.Columns, ", "))
	parts = append(parts, cols)
	if idx.Where != "" {
		parts = append(parts, "WHERE", idx.Where)
	}
	if idx.NullsNotDistinct {
		parts = append(parts, "NULLS NOT DISTINCT")
	}
	return strings.Join(parts, " ") + ";"
}
