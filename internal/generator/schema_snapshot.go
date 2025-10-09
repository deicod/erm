package generator

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type SchemaSnapshot struct {
	Tables     []TableSnapshot `json:"tables"`
	Extensions []string        `json:"extensions,omitempty"`
}

type TableSnapshot struct {
	Name             string               `json:"name"`
	Columns          []ColumnSnapshot     `json:"columns"`
	PrimaryKey       []string             `json:"primary_key,omitempty"`
	Indexes          []IndexSnapshot      `json:"indexes,omitempty"`
	ForeignKeys      []ForeignKeySnapshot `json:"foreign_keys,omitempty"`
	HypertableColumn string               `json:"hypertable_column,omitempty"`
	IsJoinTable      bool                 `json:"join_table,omitempty"`
}

type ColumnSnapshot struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Nullable    bool   `json:"nullable"`
	Unique      bool   `json:"unique,omitempty"`
	DefaultExpr string `json:"default_expr,omitempty"`
	DefaultNow  bool   `json:"default_now,omitempty"`
	Identity    bool   `json:"identity,omitempty"`
}

type IndexSnapshot struct {
	Name             string   `json:"name"`
	Columns          []string `json:"columns"`
	Unique           bool     `json:"unique,omitempty"`
	Method           string   `json:"method,omitempty"`
	Where            string   `json:"where,omitempty"`
	NullsNotDistinct bool     `json:"nulls_not_distinct,omitempty"`
}

type ForeignKeySnapshot struct {
	Column       string `json:"column"`
	TargetTable  string `json:"target_table"`
	TargetColumn string `json:"target_column"`
	Constraint   string `json:"constraint"`
}

func loadSchemaSnapshot(root string) (SchemaSnapshot, error) {
	path := filepath.Join(root, "migrations", "schema.snapshot.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SchemaSnapshot{}, nil
		}
		return SchemaSnapshot{}, err
	}
	if len(raw) == 0 {
		return SchemaSnapshot{}, nil
	}
	var snap SchemaSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return SchemaSnapshot{}, err
	}
	normalizeSnapshot(&snap)
	return snap, nil
}

func writeSchemaSnapshot(root string, snap SchemaSnapshot) error {
	normalizeSnapshot(&snap)
	path := filepath.Join(root, "migrations", "schema.snapshot.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func normalizeSnapshot(snap *SchemaSnapshot) {
	sort.Strings(snap.Extensions)
	tableLess := func(i, j int) bool {
		if snap.Tables[i].IsJoinTable == snap.Tables[j].IsJoinTable {
			return snap.Tables[i].Name < snap.Tables[j].Name
		}
		if snap.Tables[i].IsJoinTable {
			return false
		}
		return true
	}
	sort.Slice(snap.Tables, tableLess)
	for i := range snap.Tables {
		tbl := &snap.Tables[i]
		sort.Slice(tbl.Indexes, func(i, j int) bool { return tbl.Indexes[i].Name < tbl.Indexes[j].Name })
		sort.Slice(tbl.ForeignKeys, func(i, j int) bool { return tbl.ForeignKeys[i].Constraint < tbl.ForeignKeys[j].Constraint })
	}
}

func buildSchemaSnapshot(entities []Entity, flags extensionFlags) SchemaSnapshot {
	migrationEntities, joinTables := buildMigrationPlan(entities)
	tables := make([]TableSnapshot, 0, len(migrationEntities)+len(joinTables))
	for _, ent := range migrationEntities {
		table := TableSnapshot{
			Name:       pluralize(ent.Entity.Name),
			Columns:    make([]ColumnSnapshot, 0, len(ent.Fields)),
			PrimaryKey: []string{},
		}
		var hypertableColumn string
		for _, field := range ent.Fields {
			column := fieldColumn(field)
			col := ColumnSnapshot{
				Name:        column,
				Type:        fieldSQLType(field),
				Nullable:    field.Nullable,
				Unique:      field.IsUnique,
				DefaultNow:  field.HasDefaultNow,
				DefaultExpr: field.DefaultExpr,
				Identity:    isIdentityColumn(field),
			}
			if field.IsPrimary {
				table.PrimaryKey = append(table.PrimaryKey, column)
			}
			if ts, ok := field.Annotations["timeseries"].(bool); ok && ts {
				hypertableColumn = column
			}
			table.Columns = append(table.Columns, col)
		}
		table.Indexes = make([]IndexSnapshot, 0, len(ent.Entity.Indexes))
		for _, idx := range ent.Entity.Indexes {
			table.Indexes = append(table.Indexes, IndexSnapshot{
				Name:             idx.Name,
				Columns:          append([]string(nil), idx.Columns...),
				Unique:           idx.IsUnique,
				Method:           idx.Method,
				Where:            idx.Where,
				NullsNotDistinct: idx.NullsNotDistinct,
			})
		}
		table.ForeignKeys = make([]ForeignKeySnapshot, 0, len(ent.ForeignKeys))
		for _, fk := range ent.ForeignKeys {
			table.ForeignKeys = append(table.ForeignKeys, ForeignKeySnapshot{
				Column:       fk.Column,
				TargetTable:  fk.TargetTable,
				TargetColumn: fk.TargetColumn,
				Constraint:   fk.ConstraintKey,
			})
		}
		table.HypertableColumn = hypertableColumn
		tables = append(tables, table)
	}

	for _, jt := range joinTables {
		table := TableSnapshot{
			Name:        jt.Name,
			Columns:     []ColumnSnapshot{},
			PrimaryKey:  []string{jt.Left.Column, jt.Right.Column},
			ForeignKeys: []ForeignKeySnapshot{},
			IsJoinTable: true,
		}
		table.Columns = append(table.Columns,
			ColumnSnapshot{Name: jt.Left.Column, Type: jt.Left.SQLType, Nullable: false},
			ColumnSnapshot{Name: jt.Right.Column, Type: jt.Right.SQLType, Nullable: false},
		)
		table.ForeignKeys = append(table.ForeignKeys,
			ForeignKeySnapshot{
				Column:       jt.Left.Column,
				TargetTable:  jt.Left.TargetTable,
				TargetColumn: jt.Left.TargetColumn,
				Constraint:   fkConstraintName(jt.Name, jt.Left.Column),
			},
			ForeignKeySnapshot{
				Column:       jt.Right.Column,
				TargetTable:  jt.Right.TargetTable,
				TargetColumn: jt.Right.TargetColumn,
				Constraint:   fkConstraintName(jt.Name, jt.Right.Column),
			},
		)
		tables = append(tables, table)
	}

	extensions := make([]string, 0, 3)
	if flags.postgis {
		extensions = append(extensions, "postgis")
	}
	if flags.pgvector {
		extensions = append(extensions, "vector")
	}
	if flags.timescale {
		extensions = append(extensions, "timescaledb")
	}

	snap := SchemaSnapshot{
		Tables:     tables,
		Extensions: extensions,
	}
	normalizeSnapshot(&snap)
	return snap
}

func fkConstraintName(table, column string) string {
	parts := []string{"fk", table, column}
	return strings.Join(parts, "_")
}

func columnDefaultExpr(col ColumnSnapshot) (string, bool) {
	if col.DefaultNow {
		return "now()", true
	}
	if col.DefaultExpr != "" {
		return col.DefaultExpr, true
	}
	return "", false
}
