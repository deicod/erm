package generator

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deicod/erm/orm/dsl"
)

func TestWriteORMClients_EdgeHelpers(t *testing.T) {
	entities := []Entity{
		{
			Name: "Post",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.UUIDv7("author_id"),
				dsl.String("title"),
			},
			Edges: []dsl.Edge{
				dsl.ToOne("author", "User").Field("author_id").Inverse("posts"),
			},
		},
		{
			Name: "User",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.String("name"),
			},
		},
		{
			Name: "Group",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.String("name"),
			},
			Edges: []dsl.Edge{
				dsl.ManyToMany("members", "User").Inverse("groups"),
			},
		},
	}

	synthesizeInverseEdges(entities)
	for i := range entities {
		ensureDefaultQuery(&entities[i])
	}

	root := t.TempDir()
	if err := writeModels(root, entities); err != nil {
		t.Fatalf("writeModels: %v", err)
	}
	if err := writeClients(root, entities); err != nil {
		t.Fatalf("writeClients: %v", err)
	}

	clientPath := filepath.Join(root, "orm", "gen", "client_gen.go")
	clientSrc, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("read client: %v", err)
	}

	content := string(clientSrc)
	mustContain(t, content, "var ValidationRegistry = validation.NewRegistry()")
	mustContain(t, content, "const userInsertQuery = `INSERT INTO users")
	mustContain(t, content, "const userSelectQuery = `SELECT id")
	mustContain(t, content, "const userListQuery = `SELECT id")
	mustContain(t, content, "const userUpdateQuery = `UPDATE users SET")
	mustContain(t, content, "const userCountQuery = `SELECT COUNT(*) FROM users`")
	mustContain(t, content, "const userDeleteQuery = `DELETE FROM users WHERE id = $1`")
	mustContain(t, content, "ValidationRegistry.Validate(ctx, \"User\", validation.OpCreate, userValidationRecord(input), input)")
	mustContain(t, content, "ValidationRegistry.Validate(ctx, \"User\", validation.OpUpdate, userValidationRecord(input), input)")
	mustContain(t, content, "func userValidationRecord(input *User) validation.Record {")
	mustContain(t, content, "const postAuthorRelationQuery = `SELECT id, name FROM users WHERE id IN (%s)`")
	mustContain(t, content, "func (c *PostClient) LoadAuthor(")
	mustContain(t, content, "const userPostsRelationQuery = `SELECT id, author_id, title FROM posts WHERE author_id IN (%s)`")
	mustContain(t, content, "func (c *UserClient) LoadPosts(")
	mustContain(t, content, "const userGroupsRelationQuery = `SELECT id, name, jt.user_id FROM groups AS t JOIN groups_users AS jt ON t.id = jt.group_id WHERE jt.user_id IN (%s)`")
	mustContain(t, content, "func (c *UserClient) LoadGroups(")
	mustContain(t, content, "type PostQuery struct {")
	mustContain(t, content, "func (c *PostClient) Query() *PostQuery")
	mustContain(t, content, "func (q *PostQuery) WhereIDEq(")
	mustContain(t, content, "func (q *PostQuery) Count(")

	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, clientPath, clientSrc, parser.AllErrors); err != nil {
		t.Fatalf("parse client: %v", err)
	}

	modelsPath := filepath.Join(root, "orm", "gen", "models_gen.go")
	modelsSrc, err := os.ReadFile(modelsPath)
	if err != nil {
		t.Fatalf("read models: %v", err)
	}
	models := string(modelsSrc)
	mustContain(t, models, "json:\"edges,omitempty\"")
	mustContain(t, models, "type PostEdges struct {")
	mustContain(t, models, "func ensurePostEdges(")
	mustContain(t, models, "func (m *Post) SetAuthor(")
	if _, err := parser.ParseFile(fset, modelsPath, modelsSrc, parser.AllErrors); err != nil {
		t.Fatalf("parse models: %v", err)
	}
}

func TestToOneLoaderHandlesOptionalForeignKeys(t *testing.T) {
	entities := []Entity{
		{
			Name: "Task",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.UUIDv7("owner_id").Optional(),
				dsl.String("title"),
			},
			Edges: []dsl.Edge{
				dsl.ToOne("owner", "User").Field("owner_id").Optional().Inverse("tasks"),
			},
		},
		{
			Name: "User",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.String("name"),
			},
		},
	}

	synthesizeInverseEdges(entities)
	for i := range entities {
		ensureDefaultQuery(&entities[i])
	}

	root := t.TempDir()
	if err := writeORMArtifacts(root, entities); err != nil {
		t.Fatalf("writeORMArtifacts: %v", err)
	}

	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	modulePath := "example.com/app"
	goMod := fmt.Sprintf("module %s\n\ngo 1.21\n\nrequire github.com/deicod/erm v0.0.0\n\nreplace github.com/deicod/erm => %s\n", modulePath, filepath.ToSlash(repoRoot))
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goModTidy := exec.Command("go", "mod", "tidy")
	goModTidy.Dir = root
	goModTidy.Env = append(os.Environ(), "GOWORK=off")
	if output, err := goModTidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, output)
	}

	gofmt := exec.Command("gofmt", "-w", filepath.Join(root, "orm"))
	if output, err := gofmt.CombinedOutput(); err != nil {
		t.Fatalf("gofmt: %v\n%s", err, output)
	}

	goTest := exec.Command("go", "test", "./...")
	goTest.Dir = root
	goTest.Env = append(os.Environ(), "GOWORK=off")
	if output, err := goTest.CombinedOutput(); err != nil {
		t.Fatalf("go test: %v\n%s", err, output)
	}
}

func TestWriteORMClients_ComputedFields(t *testing.T) {
	entities := []Entity{
		{
			Name: "User",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.String("first_name"),
				dsl.String("last_name"),
				dsl.Text("full_name").Computed(dsl.Computed(dsl.Expression("first_name || ' ' || last_name", "first_name", "last_name"))),
			},
		},
	}

	for i := range entities {
		ensureDefaultQuery(&entities[i])
	}

	root := t.TempDir()
	if err := writeClients(root, entities); err != nil {
		t.Fatalf("writeClients: %v", err)
	}

	clientPath := filepath.Join(root, "orm", "gen", "client_gen.go")
	clientSrc, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("read client: %v", err)
	}

	content := string(clientSrc)
	mustContain(t, content, "INSERT INTO users (id, first_name, last_name) VALUES ($1, $2, $3) RETURNING id, first_name, last_name, full_name")
	mustContain(t, content, "if !runtime.IsZeroValue(input.FullName)")
	mustContain(t, content, "fmt.Errorf(\"User.FullName is computed and cannot be set\")")
	mustContain(t, content, "row := []any{input.ID, input.FirstName, input.LastName}")
}

func TestWriteRegistry_EdgeMetadata(t *testing.T) {
	entities := []Entity{
		{
			Name:   "Workspace",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary()},
			Edges: []dsl.Edge{
				dsl.ToMany("members", "User").Ref("workspace").OnDeleteCascade().OnUpdateRestrict().Polymorphic(
					dsl.PolymorphicTarget("User", "role = 'admin'"),
					dsl.PolymorphicTarget("User", "role = 'member'"),
				),
			},
		},
		{
			Name:   "User",
			Fields: []dsl.Field{dsl.UUIDv7("id").Primary(), dsl.UUIDv7("workspace_id").Optional()},
			Edges: []dsl.Edge{
				dsl.ToOne("workspace", "Workspace").Field("workspace_id").OnDeleteCascade().OnUpdateRestrict(),
			},
		},
	}

	root := t.TempDir()
	if err := writeRegistry(root, entities); err != nil {
		t.Fatalf("writeRegistry: %v", err)
	}

	path := filepath.Join(root, "orm", "gen", "registry_gen.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	out := string(content)
	mustContain(t, out, "PolymorphicTargets: []runtime.EdgeTargetSpec{{Entity: \"User\", Condition: \"role = 'admin'\"}, {Entity: \"User\", Condition: \"role = 'member'\"}}")
	mustContain(t, out, "Cascade: runtime.CascadeSpec{OnDelete: runtime.CascadeCascade, OnUpdate: runtime.CascadeRestrict}")
}

func TestExportNamePreservesInitialisms(t *testing.T) {
	tests := map[string]string{
		"post_id":    "PostID",
		"avatar_url": "AvatarURL",
		"api_token":  "APIToken",
		"id":         "ID",
		"URL":        "URL",
	}
	for input, want := range tests {
		if got := exportName(input); got != want {
			t.Fatalf("exportName(%q) = %q, want %q", input, got, want)
		}
	}
}

func mustContain(t *testing.T, content, needle string) {
	t.Helper()
	normalize := func(s string) string {
		fields := strings.Fields(s)
		return strings.Join(fields, " ")
	}
	if !strings.Contains(normalize(content), normalize(needle)) {
		t.Fatalf("expected generated content to contain %q\nactual: %s", needle, content)
	}
}

func TestDefaultGoType_PostgresFamilies(t *testing.T) {
	tests := []struct {
		name  string
		field dsl.Field
		want  string
	}{
		{"uuid", dsl.UUID("id"), "string"},
		{"text", dsl.Text("title"), "string"},
		{"bigint", dsl.BigInt("counter"), "int64"},
		{"decimal", dsl.Decimal("price", 10, 2), "string"},
		{"real", dsl.Real("ratio"), "float32"},
		{"timestamp", dsl.TimestampTZ("created_at"), "time.Time"},
		{"jsonb", dsl.JSONB("payload"), "json.RawMessage"},
		{"vector", dsl.Vector("embedding", 3), "[]float32"},
		{"array_text", dsl.Array("tags", dsl.TypeText), "[]string"},
		{"array_int", dsl.Array("scores", dsl.TypeInteger), "[]int32"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultGoType(tt.field); got != tt.want {
				t.Fatalf("defaultGoType(%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestDefaultGoType_Nullable(t *testing.T) {
	pointerField := dsl.Text("nickname").Optional()
	if got := defaultGoType(pointerField); got != "*string" {
		t.Fatalf("expected optional text field to map to *string, got %q", got)
	}

	nullTime := dsl.TimestampTZ("last_seen").Optional()
	nullTime.Annotations = map[string]any{annotationNullableGoType: nullableStrategySQLNull}
	if got := defaultGoType(nullTime); got != "sql.NullTime" {
		t.Fatalf("expected optional timestamptz with sql null strategy to map to sql.NullTime, got %q", got)
	}
}

func TestNullableFieldGeneration_EndToEnd(t *testing.T) {
	entities := []Entity{{
		Name: "Account",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
			dsl.Text("nickname").Optional(),
			func() dsl.Field {
				f := dsl.TimestampTZ("last_seen").Optional()
				f.Annotations = map[string]any{annotationNullableGoType: nullableStrategySQLNull}
				return f
			}(),
			func() dsl.Field {
				f := dsl.Boolean("active").Optional()
				f.Annotations = map[string]any{annotationNullableGoType: nullableStrategySQLNull}
				return f
			}(),
			func() dsl.Field {
				f := dsl.Integer("login_attempts").Optional()
				f.Annotations = map[string]any{annotationNullableGoType: nullableStrategySQLNull}
				return f
			}(),
		},
	}}

	for i := range entities {
		ensureDefaultQuery(&entities[i])
	}

	root := t.TempDir()
	if err := writeModels(root, entities); err != nil {
		t.Fatalf("writeModels: %v", err)
	}
	if err := writeClients(root, entities); err != nil {
		t.Fatalf("writeClients: %v", err)
	}
	modulePath := "example.com/app"
	if err := writeGraphQLResolvers(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLResolvers: %v", err)
	}
	if err := writeGraphQLDataloaders(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLDataloaders: %v", err)
	}

	modelsPath := filepath.Join(root, "orm", "gen", "models_gen.go")
	modelsSrc, err := os.ReadFile(modelsPath)
	if err != nil {
		t.Fatalf("read models: %v", err)
	}
	models := string(modelsSrc)
	mustContain(t, models, "\"database/sql\"")
	mustContain(t, models, "\"time\"")
	mustContain(t, models, "Nickname *string")
	mustContain(t, models, "`db:\"nickname,omitempty\" json:\"nickname,omitempty\"`")
	mustContain(t, models, "LastSeen sql.NullTime")
	mustContain(t, models, "`db:\"last_seen,omitempty\" json:\"last_seen,omitempty\"`")
	mustContain(t, models, "Active sql.NullBool")
	mustContain(t, models, "LoginAttempts sql.NullInt32")
	if _, err := parser.ParseFile(token.NewFileSet(), modelsPath, modelsSrc, parser.AllErrors); err != nil {
		t.Fatalf("parse models: %v", err)
	}

	resolversPath := filepath.Join(root, "graphql", "resolvers", "entities_gen.go")
	resolversSrc, err := os.ReadFile(resolversPath)
	if err != nil {
		t.Fatalf("read resolvers: %v", err)
	}
	resolvers := string(resolversSrc)
	mustContain(t, resolvers, "\"database/sql\"")
	mustContain(t, resolvers, "\"time\"")
	mustContain(t, resolvers, "model.Nickname = input.Nickname")
	mustContain(t, resolvers, "model.LastSeen = sql.NullTime{Time: *input.LastSeen, Valid: true}")
	mustContain(t, resolvers, "model.Active = sql.NullBool{Bool: *input.Active, Valid: true}")
	mustContain(t, resolvers, "model.LoginAttempts = sql.NullInt32{Int32: int32(*input.LoginAttempts), Valid: true}")
	mustContain(t, resolvers, "LastSeen: nullableTime(record.LastSeen)")
	mustContain(t, resolvers, "Active: nullableBool(record.Active)")
	mustContain(t, resolvers, "LoginAttempts: nullableInt32(record.LoginAttempts)")
	mustContain(t, resolvers, "func nullableTime(input sql.NullTime) *time.Time")
	mustContain(t, resolvers, "func nullableBool(input sql.NullBool) *bool")
	mustContain(t, resolvers, "func nullableInt32(input sql.NullInt32) *int")
	if _, err := parser.ParseFile(token.NewFileSet(), resolversPath, resolversSrc, parser.AllErrors); err != nil {
		t.Fatalf("parse resolvers: %v", err)
	}
}

func TestFieldWithGoTypePropagates(t *testing.T) {
	entities := []Entity{
		{
			Name: "User",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").WithGoType("UserID").Primary(),
				dsl.String("name"),
			},
		},
		{
			Name: "Task",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.Enum("status", "NEW", "DONE").WithGoType("Status"),
				dsl.UUIDv7("owner_id").Optional().WithGoType("UserID"),
			},
			Edges: []dsl.Edge{
				dsl.ToOne("owner", "User").Field("owner_id").Optional().Inverse("tasks"),
			},
		},
	}

	assignEnumMetadata(entities)
	synthesizeInverseEdges(entities)
	for i := range entities {
		ensureDefaultQuery(&entities[i])
	}

	root := t.TempDir()
	if err := writeORMArtifacts(root, entities); err != nil {
		t.Fatalf("writeORMArtifacts: %v", err)
	}

	modulePath := "example.com/app"
	if err := writeGraphQLResolvers(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLResolvers: %v", err)
	}
	if err := writeGraphQLDataloaders(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLDataloaders: %v", err)
	}
	if err := writeGraphQLSchema(root, entities); err != nil {
		t.Fatalf("writeGraphQLSchema: %v", err)
	}

	registryPath := filepath.Join(root, "orm", "gen", "registry_gen.go")
	registrySrc, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	registry := string(registrySrc)
	mustContain(t, registry, "GoType: \"Status\"")
	mustContain(t, registry, "GoType: \"UserID\"")

	modelsPath := filepath.Join(root, "orm", "gen", "models_gen.go")
	modelsSrc, err := os.ReadFile(modelsPath)
	if err != nil {
		t.Fatalf("read models: %v", err)
	}
	models := string(modelsSrc)
	mustContain(t, models, "Status Status `db:\"status\" json:\"status\"`")
	mustContain(t, models, "OwnerID *UserID `db:\"owner_id,omitempty\" json:\"owner_id,omitempty\"`")

	resolversPath := filepath.Join(root, "graphql", "resolvers", "entities_gen.go")
	resolversSrc, err := os.ReadFile(resolversPath)
	if err != nil {
		t.Fatalf("read resolvers: %v", err)
	}
	resolvers := string(resolversSrc)
	mustContain(t, resolvers, "model.Status = Status(fromGraphQLEnum[graphql.TaskStatus](*input.Status))")
	mustContain(t, resolvers, "value := UserID(*input.OwnerID)")
	mustContain(t, resolvers, "model.OwnerID = &value")
	mustContain(t, resolvers, "toGraphQLEnum[graphql.TaskStatus]")

	dataloadersPath := filepath.Join(root, "graphql", "dataloaders", "entities_gen.go")
	dataloadersSrc, err := os.ReadFile(dataloadersPath)
	if err != nil {
		t.Fatalf("read dataloaders: %v", err)
	}
	dataloaders := string(dataloadersSrc)
	mustContain(t, dataloaders, "*gen.Task")

	migrationEntities := []Entity{
		{
			Name: "User",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").WithGoType("UserID").Primary(),
			},
		},
		{
			Name: "Task",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.Enum("status", "NEW", "DONE").WithGoType("Status"),
			},
			Edges: []dsl.Edge{
				dsl.ToOne("owner", "User").Optional().Inverse("tasks"),
			},
		},
	}
	assignEnumMetadata(migrationEntities)
	synthesizeInverseEdges(migrationEntities)
	plan, _ := buildMigrationPlan(migrationEntities)
	var taskMigration entityMigration
	for _, ent := range plan {
		if ent.Entity.Name == "Task" {
			taskMigration = ent
			break
		}
	}
	if taskMigration.Entity.Name != "Task" {
		t.Fatalf("expected migration plan for Task entity")
	}
	var ownerField dsl.Field
	for _, field := range taskMigration.Fields {
		if field.Name == "owner_id" {
			ownerField = field
			break
		}
	}
	if ownerField.Name == "" {
		t.Fatalf("expected synthetic owner_id field in migration plan")
	}
	if ownerField.GoType != "UserID" {
		t.Fatalf("expected migration field GoType UserID, got %q", ownerField.GoType)
	}
}
