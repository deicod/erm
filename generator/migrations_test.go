package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deicod/erm/orm/dsl"
)

func TestGenerateMigrations_TableAddition(t *testing.T) {
	root := t.TempDir()
	base := []Entity{{
		Name:   "User",
		Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
	}}
	if _, err := generateMigrations(root, base, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 1, 1, 0, 0, 0)}); err != nil {
		t.Fatalf("initial migration: %v", err)
	}

	updated := []Entity{
		base[0],
		{
			Name:   "Post",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
		},
	}
	res, err := generateMigrations(root, updated, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 1, 1, 1, 0, 0)})
	if err != nil {
		t.Fatalf("table addition: %v", err)
	}
	if res.FilePath == "" {
		t.Fatalf("expected migration file to be written")
	}
	sql := readSQL(t, res.FilePath)
	if !strings.Contains(sql, "CREATE TABLE posts") {
		t.Fatalf("expected migration to create posts table, got:\n%s", sql)
	}
	snap := mustLoadSnapshot(t, root)
	if !snapshotHasTable(snap, "posts") {
		t.Fatalf("expected snapshot to contain posts table, got %#v", snap.Tables)
	}
}

func TestGenerateMigrations_ColumnDiffs(t *testing.T) {
	root := t.TempDir()
	base := []Entity{{
		Name:   "User",
		Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
	}}
	if _, err := generateMigrations(root, base, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 2, 1, 0, 0, 0)}); err != nil {
		t.Fatalf("initial migration: %v", err)
	}

	withEmail := []Entity{{
		Name:   "User",
		Fields: []dsl.Field{dsl.UUIDv7("id").Primary(), dsl.Text("email").Optional()},
	}}
	addRes, err := generateMigrations(root, withEmail, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 2, 1, 1, 0, 0)})
	if err != nil {
		t.Fatalf("add column: %v", err)
	}
	if !strings.Contains(addRes.SQL, "ALTER TABLE users ADD COLUMN email text") {
		t.Fatalf("expected add column SQL, got:\n%s", addRes.SQL)
	}
	snap := mustLoadSnapshot(t, root)
	user := mustFindTable(t, snap, "users")
	if !tableHasColumn(user, "email") {
		t.Fatalf("expected snapshot to include email column")
	}

	updated := []Entity{{
		Name:   "User",
		Fields: []dsl.Field{dsl.UUIDv7("id").Primary(), dsl.VarChar("email", 128)},
	}}
	modRes, err := generateMigrations(root, updated, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 2, 1, 2, 0, 0)})
	if err != nil {
		t.Fatalf("modify column: %v", err)
	}
	if !strings.Contains(modRes.SQL, "ALTER TABLE users ALTER COLUMN email TYPE varchar(128);") {
		t.Fatalf("expected type alteration SQL, got:\n%s", modRes.SQL)
	}
	if !strings.Contains(modRes.SQL, "ALTER TABLE users ALTER COLUMN email SET NOT NULL;") {
		t.Fatalf("expected not-null alteration SQL, got:\n%s", modRes.SQL)
	}
	snap = mustLoadSnapshot(t, root)
	user = mustFindTable(t, snap, "users")
	if !tableHasColumn(user, "email") {
		t.Fatalf("expected column to remain after modification")
	}
	col := findColumn(user, "email")
	if col.Type != "varchar(128)" || col.Nullable {
		t.Fatalf("expected snapshot column to be varchar(128) NOT NULL, got %#v", col)
	}

	remove := []Entity{{
		Name:   "User",
		Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
	}}
	dropRes, err := generateMigrations(root, remove, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 2, 1, 3, 0, 0)})
	if err != nil {
		t.Fatalf("drop column: %v", err)
	}
	if !strings.Contains(dropRes.SQL, "ALTER TABLE users DROP COLUMN IF EXISTS email CASCADE;") {
		t.Fatalf("expected drop column SQL, got:\n%s", dropRes.SQL)
	}
	snap = mustLoadSnapshot(t, root)
	user = mustFindTable(t, snap, "users")
	if tableHasColumn(user, "email") {
		t.Fatalf("expected column removal reflected in snapshot")
	}
}

func TestGenerateMigrations_IndexDiffs(t *testing.T) {
	root := t.TempDir()
	base := []Entity{{
		Name: "User",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
			dsl.Text("email").Optional(),
		},
	}}
	if _, err := generateMigrations(root, base, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 3, 1, 0, 0, 0)}); err != nil {
		t.Fatalf("initial migration: %v", err)
	}

	addIdx := []Entity{{
		Name: "User",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
			dsl.Text("email").Optional(),
		},
		Indexes: []dsl.Index{
			dsl.Idx("users_email_idx").On("email"),
		},
	}}
	addRes, err := generateMigrations(root, addIdx, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 3, 1, 1, 0, 0)})
	if err != nil {
		t.Fatalf("add index: %v", err)
	}
	if !strings.Contains(addRes.SQL, "CREATE INDEX IF NOT EXISTS users_email_idx ON users (email);") {
		t.Fatalf("expected create index SQL, got:\n%s", addRes.SQL)
	}
	snap := mustLoadSnapshot(t, root)
	user := mustFindTable(t, snap, "users")
	if !tableHasIndex(user, "users_email_idx") {
		t.Fatalf("expected snapshot to track new index")
	}

	modIdx := []Entity{{
		Name: "User",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
			dsl.Text("email").Optional(),
		},
		Indexes: []dsl.Index{
			dsl.Idx("users_email_idx").On("email").Unique(),
		},
	}}
	modRes, err := generateMigrations(root, modIdx, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 3, 1, 2, 0, 0)})
	if err != nil {
		t.Fatalf("modify index: %v", err)
	}
	if !strings.Contains(modRes.SQL, "DROP INDEX IF EXISTS users_email_idx;") {
		t.Fatalf("expected drop index SQL, got:\n%s", modRes.SQL)
	}
	if !strings.Contains(modRes.SQL, "CREATE UNIQUE INDEX IF NOT EXISTS users_email_idx ON users (email);") {
		t.Fatalf("expected recreate unique index SQL, got:\n%s", modRes.SQL)
	}
	snap = mustLoadSnapshot(t, root)
	user = mustFindTable(t, snap, "users")
	idx := findIndex(user, "users_email_idx")
	if idx == nil || !idx.Unique {
		t.Fatalf("expected snapshot index to be unique, got %#v", idx)
	}

	dropIdx := []Entity{base[0]}
	dropRes, err := generateMigrations(root, dropIdx, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 3, 1, 3, 0, 0)})
	if err != nil {
		t.Fatalf("drop index: %v", err)
	}
	if !strings.Contains(dropRes.SQL, "DROP INDEX IF EXISTS users_email_idx;") {
		t.Fatalf("expected drop index SQL, got:\n%s", dropRes.SQL)
	}
	snap = mustLoadSnapshot(t, root)
	user = mustFindTable(t, snap, "users")
	if tableHasIndex(user, "users_email_idx") {
		t.Fatalf("expected index removal reflected in snapshot")
	}
}

func TestGenerateMigrations_JoinTableDiffs(t *testing.T) {
	root := t.TempDir()
	base := []Entity{
		{
			Name:   "User",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
		},
		{
			Name:   "Group",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
		},
	}
	if _, err := generateMigrations(root, base, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 4, 1, 0, 0, 0)}); err != nil {
		t.Fatalf("initial migration: %v", err)
	}

	withJoin := []Entity{
		{
			Name:   "User",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
			Edges: []dsl.Edge{
				dsl.ManyToMany("groups", "Group"),
			},
		},
		base[1],
	}
	addRes, err := generateMigrations(root, withJoin, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 4, 1, 1, 0, 0)})
	if err != nil {
		t.Fatalf("add join table: %v", err)
	}
	if !strings.Contains(addRes.SQL, "CREATE TABLE groups_users") {
		t.Fatalf("expected join table creation, got:\n%s", addRes.SQL)
	}
	snap := mustLoadSnapshot(t, root)
	if !snapshotHasJoinTable(snap, "groups_users") {
		t.Fatalf("expected snapshot to include join table")
	}

	dropRes, err := generateMigrations(root, base, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 4, 1, 2, 0, 0)})
	if err != nil {
		t.Fatalf("drop join table: %v", err)
	}
	if !strings.Contains(dropRes.SQL, "DROP TABLE IF EXISTS groups_users CASCADE;") {
		t.Fatalf("expected join table drop SQL, got:\n%s", dropRes.SQL)
	}
	snap = mustLoadSnapshot(t, root)
	if snapshotHasJoinTable(snap, "groups_users") {
		t.Fatalf("expected join table removal in snapshot")
	}
}

func TestGenerateMigrations_ExtensionToggles(t *testing.T) {
	root := t.TempDir()
	base := []Entity{{
		Name:   "Place",
		Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
	}}
	if _, err := generateMigrations(root, base, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 5, 1, 0, 0, 0)}); err != nil {
		t.Fatalf("initial migration: %v", err)
	}

	withPostGIS := []Entity{{
		Name: "Place",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
			dsl.Geometry("boundary"),
		},
	}}
	addRes, err := generateMigrations(root, withPostGIS, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 5, 1, 1, 0, 0)})
	if err != nil {
		t.Fatalf("enable postgis: %v", err)
	}
	if !strings.Contains(addRes.SQL, "CREATE EXTENSION IF NOT EXISTS postgis;") {
		t.Fatalf("expected extension enable SQL, got:\n%s", addRes.SQL)
	}
	snap := mustLoadSnapshot(t, root)
	if !sliceContains(snap.Extensions, "postgis") {
		t.Fatalf("expected snapshot to record postgis extension")
	}

	revert := base
	dropRes, err := generateMigrations(root, revert, generatorOptions{GenerateOptions: GenerateOptions{}, Now: fixedClock(2024, 5, 1, 2, 0, 0)})
	if err != nil {
		t.Fatalf("disable postgis: %v", err)
	}
	if !strings.Contains(dropRes.SQL, "DROP EXTENSION IF EXISTS postgis;") {
		t.Fatalf("expected extension drop SQL, got:\n%s", dropRes.SQL)
	}
	snap = mustLoadSnapshot(t, root)
	if sliceContains(snap.Extensions, "postgis") {
		t.Fatalf("expected snapshot to remove postgis extension")
	}
}

func TestRenderInitialMigration_OneToManyEdges(t *testing.T) {
	entities := []Entity{
		{
			Name:   "User",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
			Edges: []dsl.Edge{
				dsl.ToMany("posts", "Post").Ref("author_id"),
			},
		},
		{
			Name:   "Post",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
		},
	}

	sql := renderInitialMigration(entities, extensionFlags{})

	if !strings.Contains(sql, "author_id uuid NOT NULL") {
		t.Fatalf("expected posts table to include author_id column, got:\n%s", sql)
	}
	if !strings.Contains(sql, "CONSTRAINT fk_posts_author_id FOREIGN KEY (author_id) REFERENCES users (id)") {
		t.Fatalf("expected posts table to include foreign key constraint, got:\n%s", sql)
	}
}

func TestRenderInitialMigration_ToManyDerivesRefColumn(t *testing.T) {
	entities := []Entity{
		{
			Name:   "User",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
			Edges: []dsl.Edge{
				dsl.ToMany("pets", "Pet"),
			},
		},
		{
			Name:   "Pet",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
		},
	}

	sql := renderInitialMigration(entities, extensionFlags{})

	if !strings.Contains(sql, "user_id uuid NOT NULL") {
		t.Fatalf("expected pets table to include derived user_id column, got:\n%s", sql)
	}
	if !strings.Contains(sql, "CONSTRAINT fk_pets_user_id FOREIGN KEY (user_id) REFERENCES users (id)") {
		t.Fatalf("expected pets table to include foreign key constraint, got:\n%s", sql)
	}
}

func TestRenderInitialMigration_ManyToManyEdges(t *testing.T) {
	entities := []Entity{
		{
			Name:   "User",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
			Edges: []dsl.Edge{
				dsl.ManyToMany("groups", "Group"),
			},
		},
		{
			Name:   "Group",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
			Edges: []dsl.Edge{
				dsl.ManyToMany("members", "User").ThroughTable("memberships"),
			},
		},
	}

	sql := renderInitialMigration(entities, extensionFlags{})

	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS groups_users") {
		t.Fatalf("expected default join table groups_users to be created, got:\n%s", sql)
	}
	if !strings.Contains(sql, "CONSTRAINT fk_groups_users_user_id FOREIGN KEY (user_id) REFERENCES users (id)") {
		t.Fatalf("expected groups_users table to reference users(id), got:\n%s", sql)
	}
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS memberships") {
		t.Fatalf("expected custom through table memberships to be created, got:\n%s", sql)
	}
	if !strings.Contains(sql, "CONSTRAINT fk_memberships_group_id FOREIGN KEY (group_id) REFERENCES groups (id)") {
		t.Fatalf("expected memberships table to reference groups(id), got:\n%s", sql)
	}
}

func TestRenderInitialMigration_ForeignKeyCascade(t *testing.T) {
	entities := []Entity{
		{
			Name: "Parent",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
			},
		},
		{
			Name: "Child",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.UUIDv7("parent_id"),
			},
			Edges: []dsl.Edge{
				dsl.ToOne("parent", "Parent").Field("parent_id").OnDeleteCascade().OnUpdateRestrict(),
			},
		},
	}

	sql := renderInitialMigration(entities, extensionFlags{})

	if !strings.Contains(sql, "ON DELETE CASCADE") {
		t.Fatalf("expected cascade delete clause in migration:\n%s", sql)
	}
	if !strings.Contains(sql, "ON UPDATE RESTRICT") {
		t.Fatalf("expected restrict update clause in migration:\n%s", sql)
	}
}

func TestRenderInitialMigration_JoinTableCascade(t *testing.T) {
	entities := []Entity{
		{
			Name:   "User",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
			Edges: []dsl.Edge{
				dsl.ManyToMany("teams", "Team").OnDeleteCascade().OnUpdateCascade(),
			},
		},
		{
			Name:   "Team",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
		},
	}

	sql := renderInitialMigration(entities, extensionFlags{})

	if !strings.Contains(sql, "CONSTRAINT fk_teams_users_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE ON UPDATE CASCADE") {
		t.Fatalf("expected join table constraint to include cascades, got:\n%s", sql)
	}
}

func TestFieldSQLType_PostgresFamilies(t *testing.T) {
	tests := []struct {
		name  string
		field dsl.Field
		want  string
	}{
		{"text", dsl.Text("title"), "text"},
		{"varchar", dsl.VarChar("code", 12), "varchar(12)"},
		{"decimal", dsl.Decimal("price", 10, 2), "decimal(10,2)"},
		{"timestamp", dsl.TimestampTZ("created_at"), "timestamptz"},
		{"identity", dsl.BigIntIdentity("id", dsl.IdentityAlways), "bigint GENERATED ALWAYS AS IDENTITY"},
		{"jsonb", dsl.JSONB("payload"), "jsonb"},
		{"bit", dsl.Bit("mask", 8), "bit(8)"},
		{"array", dsl.Array("tags", dsl.TypeText), "text[]"},
		{"vector", dsl.Vector("embedding", 3), "vector(3)"},
		{"computed", dsl.Text("full_name").Computed(dsl.Computed(dsl.Expression("first_name || ' ' || last_name"))), "text GENERATED ALWAYS AS (first_name || ' ' || last_name) STORED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fieldSQLType(tt.field)
			if got != tt.want {
				t.Fatalf("fieldSQLType(%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestDiffColumn_ComputedTriggersDropAdd(t *testing.T) {
	prev := ColumnSnapshot{
		Name:          "full_name",
		Type:          "text GENERATED ALWAYS AS (first_name || ' ' || last_name) STORED",
		GeneratedExpr: "first_name || ' ' || last_name",
		ReadOnly:      true,
	}
	next := ColumnSnapshot{
		Name:          "full_name",
		Type:          "text GENERATED ALWAYS AS (first_name || ' ' || COALESCE(last_name, '')) STORED",
		GeneratedExpr: "first_name || ' ' || COALESCE(last_name, '')",
		ReadOnly:      true,
	}

	ops := diffColumn("users", prev, next)
	if len(ops) != 2 {
		t.Fatalf("expected drop/add operations, got %d", len(ops))
	}
	if ops[0].Kind != OpDropColumn || ops[1].Kind != OpAddColumn {
		t.Fatalf("expected drop/add operations, got %+v", ops)
	}
}

func fixedClock(year int, month time.Month, day, hour, min, sec int) func() time.Time {
	return func() time.Time {
		return time.Date(year, month, day, hour, min, sec, 0, time.UTC)
	}
}

func readSQL(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func mustLoadSnapshot(t *testing.T, root string) SchemaSnapshot {
	t.Helper()
	snap, err := loadSchemaSnapshot(root)
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	return snap
}

func snapshotHasTable(snap SchemaSnapshot, name string) bool {
	for _, tbl := range snap.Tables {
		if tbl.Name == name {
			return true
		}
	}
	return false
}

func mustFindTable(t *testing.T, snap SchemaSnapshot, name string) TableSnapshot {
	t.Helper()
	for _, tbl := range snap.Tables {
		if tbl.Name == name {
			return tbl
		}
	}
	t.Fatalf("table %s not found", name)
	return TableSnapshot{}
}

func tableHasColumn(tbl TableSnapshot, name string) bool {
	for _, col := range tbl.Columns {
		if col.Name == name {
			return true
		}
	}
	return false
}

func findColumn(tbl TableSnapshot, name string) ColumnSnapshot {
	for _, col := range tbl.Columns {
		if col.Name == name {
			return col
		}
	}
	return ColumnSnapshot{}
}

func tableHasIndex(tbl TableSnapshot, name string) bool {
	return findIndex(tbl, name) != nil
}

func findIndex(tbl TableSnapshot, name string) *IndexSnapshot {
	for i := range tbl.Indexes {
		if tbl.Indexes[i].Name == name {
			return &tbl.Indexes[i]
		}
	}
	return nil
}

func snapshotHasJoinTable(snap SchemaSnapshot, name string) bool {
	for _, tbl := range snap.Tables {
		if tbl.Name == name && tbl.IsJoinTable {
			return true
		}
	}
	return false
}

func sliceContains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func TestRenderInitialMigration_BlogSchemaForeignKeyOrder(t *testing.T) {
	entities := loadBlogEntities(t)
	plan, _ := buildMigrationPlan(entities)
	sql := renderInitialMigration(entities, extensionFlags{})
	order := createStatementOrder(sql)
	for _, ent := range plan {
		table := pluralize(ent.Entity.Name)
		tableIdx, ok := order[table]
		if !ok {
			t.Fatalf("missing create statement for %s", table)
		}
		for _, fk := range ent.ForeignKeys {
			if fk.TargetTable == table {
				continue
			}
			targetIdx, ok := order[fk.TargetTable]
			if !ok {
				t.Fatalf("missing create statement for referenced table %s", fk.TargetTable)
			}
			if targetIdx > tableIdx {
				t.Fatalf("table %s (order %d) references %s (order %d) before it is created", table, tableIdx, fk.TargetTable, targetIdx)
			}
		}
	}
}

func TestRenderInitialMigration_BlogSchema_NoDuplicateEdgeColumns(t *testing.T) {
	entities := loadBlogEntities(t)
	sql := renderInitialMigration(entities, extensionFlags{})

	comments := tableDefinition(t, sql, "comments")
	if strings.Contains(comments, "post uuid") {
		t.Fatalf("unexpected synthetic column in comments table: %s", comments)
	}
	if strings.Contains(comments, "parent uuid") {
		t.Fatalf("unexpected synthetic column in comments table: %s", comments)
	}
	if count := strings.Count(comments, "post_id uuid"); count != 1 {
		t.Fatalf("expected one post_id column in comments table, found %d\n%s", count, comments)
	}
	if count := strings.Count(comments, "parent_id uuid"); count != 1 {
		t.Fatalf("expected one parent_id column in comments table, found %d\n%s", count, comments)
	}

	posts := tableDefinition(t, sql, "posts")
	if strings.Contains(posts, "author uuid") {
		t.Fatalf("unexpected synthetic column in posts table: %s", posts)
	}
	if count := strings.Count(posts, "author_id uuid"); count != 1 {
		t.Fatalf("expected one author_id column in posts table, found %d\n%s", count, posts)
	}
}

func loadBlogEntities(t *testing.T) []Entity {
	t.Helper()
	dir := t.TempDir()
	copyTree(t, filepath.Join("..", "examples", "blog"), dir)
	entities, err := loadEntities(dir)
	if err != nil {
		t.Fatalf("loadEntities: %v", err)
	}
	return entities
}

func tableDefinition(t *testing.T, sql, table string) string {
	t.Helper()
	marker := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", table)
	start := strings.Index(sql, marker)
	if start == -1 {
		t.Fatalf("table definition for %s not found", table)
	}
	start += len(marker)
	rest := sql[start:]
	end := strings.Index(rest, ");")
	if end == -1 {
		t.Fatalf("unterminated table definition for %s", table)
	}
	return rest[:end]
}

func createStatementOrder(sql string) map[string]int {
	order := make(map[string]int)
	lines := strings.Split(sql, "\n")
	position := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "CREATE TABLE IF NOT EXISTS ") {
			name := strings.TrimPrefix(trimmed, "CREATE TABLE IF NOT EXISTS ")
			if idx := strings.IndexAny(name, " ("); idx >= 0 {
				name = name[:idx]
			}
			order[name] = position
			position++
		}
	}
	return order
}
