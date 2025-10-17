package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"text/template"

	"github.com/deicod/erm/orm/dsl"
	"github.com/deicod/erm/templates"
	testkit "github.com/deicod/erm/testing"
)

func TestGraphQLTypeMappings(t *testing.T) {
	entities := []Entity{
		{
			Name: "Sample",
			Fields: []dsl.Field{
				dsl.UUIDv7("id").Primary(),
				dsl.Text("title"),
				dsl.BigInt("counter"),
				dsl.Decimal("price", 10, 2),
				dsl.Date("ship_date").Optional(),
				dsl.Time("ship_time").Optional(),
				dsl.TimestampTZ("created_at"),
				dsl.JSONB("metadata"),
				dsl.Array("tags", dsl.TypeText),
				dsl.Vector("embedding", 3),
			},
		},
	}

	schema := buildGraphQLGeneratedSection(entities)

	checks := []string{
		"counter: BigInt!",
		"price: Decimal!",
		"metadata: JSONB!",
		"createdAt: Timestamptz!",
		"shipDate: Date",
		"shipTime: Time",
		"title: String",
		"CreateSampleInput",
		"updateSample",
		"deleteSample",
	}

	for _, needle := range checks {
		if !strings.Contains(schema, needle) {
			t.Fatalf("expected GraphQL schema to contain %q\nactual: %s", needle, schema)
		}
	}

	declaredScalars := []string{
		"scalar BigInt",
		"scalar Decimal",
		"scalar JSONB",
		"scalar Timestamptz",
		"scalar Date",
	}

	for _, scalar := range declaredScalars {
		mustContain(t, schema, scalar)
	}

	mustNotContain(t, schema, "scalar Time\n")
}

func TestGraphQLEnumGeneration(t *testing.T) {
	entities := []Entity{{
		Name: "Task",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
			dsl.Enum("status", "NEW", "DONE"),
		},
	}}
	assignEnumMetadata(entities)

	schema := buildGraphQLGeneratedSection(entities)
	mustContain(t, schema, "enum TaskStatus")
	mustContain(t, schema, "status: TaskStatus!")
}

func TestGraphQLResolverGeneration(t *testing.T) {
	entities := []Entity{{
		Name: "Widget",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
			dsl.Text("name"),
			dsl.Integer("version"),
			dsl.Integer("optional_version").Optional(),
			dsl.SmallInt("login_attempts"),
			dsl.SmallInt("optional_login_attempts").Optional(),
		},
	}}

	root := t.TempDir()
	modulePath := "example.com/app"

	if err := writeGraphQLResolvers(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLResolvers: %v", err)
	}
	if err := writeGraphQLDataloaders(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLDataloaders: %v", err)
	}

	hooksPath := filepath.Join(root, "graphql", "resolvers", "entities_hooks.go")
	hooksSrc, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("read hooks: %v", err)
	}
	mustContain(t, string(hooksSrc), "newEntityHooks")

	resolverPath := filepath.Join(root, "graphql", "resolvers", "entities_gen.go")
	resolverSrc, err := os.ReadFile(resolverPath)
	if err != nil {
		t.Fatalf("read resolvers: %v", err)
	}
	expectations := []string{
		"func (r *mutationResolver) CreateWidget",
		"func (r *queryResolver) Widgets",
		"func decodeWidgetID",
		"type entityHooks struct",
		"applyBeforeCreateWidget",
		"applyBeforeReturnWidget",
		modulePath + "/graphql",
		modulePath + "/graphql/dataloaders",
		"\"math\"",
		"int(record.Version)",
		"toGraphQLIntPtr(record.OptionalVersion)",
		"func toGraphQLIntPtr[",
		"if *input.LoginAttempts < math.MinInt16 || *input.LoginAttempts > math.MaxInt16",
		"return nil, fmt.Errorf(\"loginAttempts must be between %d and %d\", math.MinInt16, math.MaxInt16)",
	}
	for _, needle := range expectations {
		if !strings.Contains(string(resolverSrc), needle) {
			t.Fatalf("expected resolver source to contain %q\n%s", needle, resolverSrc)
		}
	}
	if strings.Contains(string(resolverSrc), "github.com/deicod/erm") {
		t.Fatalf("resolver source still references repository module path\n%s", resolverSrc)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), resolverPath, resolverSrc, parser.AllErrors); err != nil {
		t.Fatalf("resolvers parse: %v", err)
	}

	if err := writeGraphQLResolvers(root, entities, modulePath); err != nil {
		t.Fatalf("second writeGraphQLResolvers: %v", err)
	}
	hooksAgain, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("read hooks second: %v", err)
	}
	if !bytes.Equal(hooksSrc, hooksAgain) {
		t.Fatalf("expected hooks stub to remain unchanged")
	}

	loaderPath := filepath.Join(root, "graphql", "dataloaders", "entities_gen.go")
	loaderSrc, err := os.ReadFile(loaderPath)
	if err != nil {
		t.Fatalf("read dataloaders: %v", err)
	}
	loaderExpectations := []string{
		"configureEntityLoaders",
		"func (l *Loaders) Widget()",
		"orm.Widgets().ByID",
		modulePath + "/observability/metrics",
	}
	for _, needle := range loaderExpectations {
		if !strings.Contains(string(loaderSrc), needle) {
			t.Fatalf("expected dataloader source to contain %q\n%s", needle, loaderSrc)
		}
	}
	if strings.Contains(string(loaderSrc), "github.com/deicod/erm") {
		t.Fatalf("dataloader source still references repository module path\n%s", loaderSrc)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), loaderPath, loaderSrc, parser.AllErrors); err != nil {
		t.Fatalf("dataloaders parse: %v", err)
	}
}

func TestGraphQLInitialismHandling(t *testing.T) {
	entities := []Entity{{
		Name: "Profile",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
			dsl.UUIDv7("post_id"),
			dsl.String("avatar_url"),
			dsl.String("api_token"),
		},
	}}

	schema := buildGraphQLGeneratedSection(entities)
	mustContain(t, schema, "postID")
	mustContain(t, schema, "avatarURL")
	mustContain(t, schema, "apiToken")

	root := t.TempDir()
	modulePath := "example.com/app"
	if err := writeGraphQLResolvers(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLResolvers: %v", err)
	}

	resolverPath := filepath.Join(root, "graphql", "resolvers", "entities_gen.go")
	resolverSrc, err := os.ReadFile(resolverPath)
	if err != nil {
		t.Fatalf("read resolvers: %v", err)
	}

	mustContain(t, string(resolverSrc), "input.PostID")
	mustContain(t, string(resolverSrc), "model.PostID")
	mustContain(t, string(resolverSrc), "input.AvatarURL")
	mustContain(t, string(resolverSrc), "model.AvatarURL")
	mustContain(t, string(resolverSrc), "input.APIToken")
	mustContain(t, string(resolverSrc), "model.APIToken")
}

func TestWriteGraphQLArtifactsEnsuresScalarHelpers(t *testing.T) {
	template, err := os.ReadFile(filepath.Join("..", "templates", "graphql", "scalars.go.tmpl"))
	if err != nil {
		t.Fatalf("read scalars template: %v", err)
	}
	RegisterGraphQLScalarTemplate(template)
	t.Cleanup(func() {
		content, err := templates.GraphQLScalarsTemplate()
		if err != nil {
			t.Fatalf("reload default scalars template: %v", err)
		}
		RegisterGraphQLScalarTemplate(content)
	})

	root := t.TempDir()
	modulePath := "example.com/app"
	entities := []Entity{{
		Name: "Widget",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
		},
	}}

	if err := writeGraphQLArtifacts(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLArtifacts: %v", err)
	}

	path := filepath.Join(root, "graphql", "scalars.go")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected scalars helper to be written: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read scalars helper: %v", err)
	}
	if string(content) != string(template) {
		t.Fatalf("unexpected scalars helper content\nwant:\n%s\ngot:\n%s", template, content)
	}
}

func TestRunGQLGenWithStubbedRuntime(t *testing.T) {
	root := t.TempDir()
	modulePath := "example.com/app"

	goMod := "module " + modulePath + "\n\ngo 1.21\n\nrequire (\n\tgithub.com/99designs/gqlgen v0.17.80\n\tgithub.com/vektah/gqlparser/v2 v2.5.30\n)\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	entities := []Entity{{
		Name: "Gizmo",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
			dsl.Text("name"),
		},
	}}

	if err := writeGraphQLArtifacts(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLArtifacts: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir temp project: %v", err)
	}

	testkit.ScaffoldGraphQLRuntime(t, root, modulePath)

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = root
	if output, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, output)
	}

	if err := runGQLGen(root); err != nil {
		t.Fatalf("runGQLGen: %v", err)
	}
}

func TestRunGQLGenRespectsRegisteredScalarHelpers(t *testing.T) {
	template, err := os.ReadFile(filepath.Join("..", "templates", "graphql", "scalars.go.tmpl"))
	if err != nil {
		t.Fatalf("read scalars template: %v", err)
	}
	RegisterGraphQLScalarTemplate(template)
	t.Cleanup(func() {
		content, err := templates.GraphQLScalarsTemplate()
		if err != nil {
			t.Fatalf("reload default scalars template: %v", err)
		}
		RegisterGraphQLScalarTemplate(content)
	})

	root := t.TempDir()
	modulePath := "example.com/app"

	goMod := "module " + modulePath + "\n\n" +
		"go 1.21\n\n" +
		"require (\n" +
		"\tgithub.com/99designs/gqlgen v0.17.80\n" +
		"\tgithub.com/vektah/gqlparser/v2 v2.5.30\n" +
		")\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	entities := []Entity{{
		Name: "Event",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
			dsl.TimestampTZ("starts_at"),
		},
	}}

	if err := writeGraphQLArtifacts(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLArtifacts: %v", err)
	}

	writeGraphQLDataloaderRuntime(t, root, modulePath)
	writeORMEntityStubs(t, root, entities)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir temp project: %v", err)
	}

	testkit.ScaffoldGraphQLRuntime(t, root, modulePath)

	if err := os.Remove(filepath.Join(root, "graphql", "graphql.go")); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("remove placeholder graphql.go: %v", err)
	}

	writeGraphQLResolverRuntime(t, root, modulePath)
	writeGraphQLRelayRuntime(t, root)

	before := collectGoFiles(t, filepath.Join(root, "graphql"))

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = root
	if output, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, output)
	}

	if err := runGQLGen(root); err != nil {
		t.Fatalf("runGQLGen: %v", err)
	}

	after := collectGoFiles(t, filepath.Join(root, "graphql"))
	var newFiles []string
	for path := range after {
		if _, ok := before[path]; !ok {
			newFiles = append(newFiles, path)
		}
	}
	sort.Strings(newFiles)
	switch len(newFiles) {
	case 1:
		if newFiles[0] != "models_gen.go" {
			t.Fatalf("unexpected new GraphQL files: %v", newFiles)
		}
	case 2:
		if newFiles[0] != "generated.go" || newFiles[1] != "models_gen.go" {
			t.Fatalf("unexpected new GraphQL files: %v", newFiles)
		}
	default:
		t.Fatalf("unexpected new GraphQL files: %v", newFiles)
	}
	if _, ok := after["generated.go"]; !ok {
		t.Fatalf("expected generated.go to exist after gqlgen run")
	}

	scalarsPath := filepath.Join(root, "graphql", "scalars.go")
	content, err := os.ReadFile(scalarsPath)
	if err != nil {
		t.Fatalf("read scalars helper: %v", err)
	}
	if string(content) != string(template) {
		t.Fatalf("scalar helper mutated by gqlgen\nwant:\n%s\n\n got:\n%s", template, content)
	}

	resolverPath := filepath.Join(root, "graphql", "resolvers", "resolver.go")
	resolverSrc, err := os.ReadFile(resolverPath)
	if err != nil {
		t.Fatalf("read resolver runtime: %v", err)
	}
	if !strings.Contains(string(resolverSrc), "hooks         entityHooks") {
		t.Fatalf("unexpected resolver runtime content\n%s", resolverSrc)
	}

	build := exec.Command("go", "build", "./...")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, output)
	}
}

func TestGraphQLJSONBScalarGeneration(t *testing.T) {
	root := t.TempDir()
	modulePath := "example.com/app"

	goMod := "module " + modulePath + "\n\n" +
		"go 1.21\n\n" +
		"require (\n" +
		"\tgithub.com/99designs/gqlgen v0.17.80\n" +
		"\tgithub.com/vektah/gqlparser/v2 v2.5.30\n" +
		")\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	entities := []Entity{{
		Name: "Document",
		Fields: []dsl.Field{
			dsl.UUIDv7("id").Primary(),
			dsl.JSONB("metadata"),
			dsl.JSON("payload").Optional(),
		},
	}}

	if err := writeGraphQLArtifacts(root, entities, modulePath); err != nil {
		t.Fatalf("writeGraphQLArtifacts: %v", err)
	}

	writeGraphQLDataloaderRuntime(t, root, modulePath)
	writeORMEntityStubs(t, root, entities)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir temp project: %v", err)
	}

	testkit.ScaffoldGraphQLRuntime(t, root, modulePath)

	if err := os.Remove(filepath.Join(root, "graphql", "graphql.go")); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("remove placeholder graphql.go: %v", err)
	}

	writeGraphQLResolverRuntime(t, root, modulePath)
	writeGraphQLRelayRuntime(t, root)

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = root
	if output, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, output)
	}

	debugDir := filepath.Join(os.TempDir(), "jsonb-debug")
	if err := os.RemoveAll(debugDir); err != nil {
		t.Fatalf("remove debug dir: %v", err)
	}
	t.Cleanup(func() {
		if !t.Failed() {
			_ = os.RemoveAll(debugDir)
		}
	})
	if err := runGQLGen(root); err != nil {
		t.Fatalf("runGQLGen: %v", err)
	}
	copyDir(t, root, debugDir)

	resolverPath := filepath.Join(root, "graphql", "resolvers", "entities_gen.go")
	resolverSrc, err := os.ReadFile(resolverPath)
	if err != nil {
		t.Fatalf("read resolvers: %v", err)
	}
	if strings.Contains(string(resolverSrc), "*input.Metadata") {
		t.Fatalf("resolver still dereferences JSONB input\n%s", resolverSrc)
	}
	if strings.Contains(string(resolverSrc), "*input.Payload") {
		t.Fatalf("resolver still dereferences JSON input\n%s", resolverSrc)
	}

	generatedPath := filepath.Join(root, "graphql", "generated.go")
	generatedSrc, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("read generated schema: %v", err)
	}
	if strings.Contains(string(generatedSrc), "unmarshalOJSONB2ᚖencoding/json.RawMessage") {
		t.Fatalf("optional JSONB wrapper still returns pointer\n%s", generatedSrc)
	}
	if strings.Contains(string(generatedSrc), "marshalJSONB2ᚖencoding/json.RawMessage") {
		t.Fatalf("JSONB marshal wrapper still expects pointer\n%s", generatedSrc)
	}

	build := exec.Command("go", "build", "./...")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, output)
	}
}

func mustNotContain(t *testing.T, content, needle string) {
	t.Helper()
	if strings.Contains(content, needle) {
		t.Fatalf("expected GraphQL schema to omit %q\nactual: %s", needle, content)
	}
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()

	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("readdir %s: %v", src, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dst, err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("stat %s: %v", srcPath, err)
		}
		if info.IsDir() {
			copyDir(t, srcPath, dstPath)
			continue
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("read %s: %v", srcPath, err)
		}
		if err := os.WriteFile(dstPath, data, info.Mode()); err != nil {
			t.Fatalf("write %s: %v", dstPath, err)
		}
	}
}

func collectGoFiles(t *testing.T, dir string) map[string]struct{} {
	t.Helper()

	files := make(map[string]struct{})
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = struct{}{}
		return nil
	}); err != nil {
		t.Fatalf("collect go files: %v", err)
	}
	return files
}

func writeGraphQLDataloaderRuntime(t *testing.T, root, modulePath string) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "templates", "graphql", "dataloaders", "loader.go.tmpl"))
	if err != nil {
		t.Fatalf("read dataloader runtime template: %v", err)
	}
	tpl, err := template.New("loader.go.tmpl").Parse(string(raw))
	if err != nil {
		t.Fatalf("parse dataloader runtime template: %v", err)
	}

	buf := &bytes.Buffer{}
	data := struct{ ModulePath string }{ModulePath: modulePath}
	if err := tpl.Execute(buf, data); err != nil {
		t.Fatalf("render dataloader runtime template: %v", err)
	}

	path := filepath.Join(root, "graphql", "dataloaders", "loader.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir dataloaders runtime: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write dataloader runtime: %v", err)
	}
}

func writeORMEntityStubs(t *testing.T, root string, entities []Entity) {
	t.Helper()

	if len(entities) == 0 {
		return
	}

	builder := &strings.Builder{}
	builder.WriteString("package gen\n\n")

	imports := map[string]struct{}{"context": {}}
	for _, ent := range entities {
		for _, field := range ent.Fields {
			goType := defaultGoType(field)
			if strings.Contains(goType, "time.") {
				imports["time"] = struct{}{}
			}
			if strings.Contains(goType, "json.RawMessage") {
				imports["encoding/json"] = struct{}{}
			}
		}
	}
	if len(imports) > 0 {
		builder.WriteString("import (\n")
		keys := make([]string, 0, len(imports))
		for key := range imports {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(builder, "    \"%s\"\n", key)
		}
		builder.WriteString(")\n\n")
	}

	for _, ent := range entities {
		plural := exportName(pluralize(ent.Name))
		fmt.Fprintf(builder, "type %s struct {\n", ent.Name)
		for _, field := range ent.Fields {
			fmt.Fprintf(builder, "    %s %s\n", exportName(field.Name), defaultGoType(field))
		}
		builder.WriteString("}\n\n")
		fmt.Fprintf(builder, "type %sClient struct{}\n\n", ent.Name)
		fmt.Fprintf(builder, "func (c *%sClient) ByID(context.Context, string) (*%s, error) { return nil, nil }\n\n", ent.Name, ent.Name)
		fmt.Fprintf(builder, "func (c *%sClient) Count(context.Context) (int, error) { return 0, nil }\n\n", ent.Name)
		fmt.Fprintf(builder, "func (c *%sClient) List(context.Context, int, int) ([]*%s, error) { return nil, nil }\n\n", ent.Name, ent.Name)
		fmt.Fprintf(builder, "func (c *%sClient) Create(context.Context, *%s) (*%s, error) { return nil, nil }\n\n", ent.Name, ent.Name, ent.Name)
		fmt.Fprintf(builder, "func (c *%sClient) Update(context.Context, *%s) (*%s, error) { return nil, nil }\n\n", ent.Name, ent.Name, ent.Name)
		fmt.Fprintf(builder, "func (c *%sClient) Delete(context.Context, string) error { return nil }\n\n", ent.Name)
		fmt.Fprintf(builder, "func (c *Client) %s() *%sClient { return &%sClient{} }\n\n", plural, ent.Name, ent.Name)
	}

	path := filepath.Join(root, "orm", "gen", "entities_stub.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir orm entity stubs: %v", err)
	}
	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		t.Fatalf("write orm entity stubs: %v", err)
	}
}

func writeGraphQLResolverRuntime(t *testing.T, root, modulePath string) {
	t.Helper()

	builder := &strings.Builder{}
	builder.WriteString("package resolvers\n\n")
	builder.WriteString("import (\n")
	builder.WriteString("    \"context\"\n")
	builder.WriteString("    \"errors\"\n")
	builder.WriteString("    \"strconv\"\n")
	builder.WriteString("    \"strings\"\n\n")
	fmt.Fprintf(builder, "    \"%s/graphql\"\n", modulePath)
	fmt.Fprintf(builder, "    \"%s/graphql/dataloaders\"\n", modulePath)
	fmt.Fprintf(builder, "    \"%s/graphql/subscriptions\"\n", modulePath)
	fmt.Fprintf(builder, "    \"%s/observability/metrics\"\n", modulePath)
	fmt.Fprintf(builder, "    \"%s/orm/gen\"\n", modulePath)
	builder.WriteString(")\n\n")
	builder.WriteString("type Options struct {\n")
	builder.WriteString("    ORM           *gen.Client\n")
	builder.WriteString("    Collector     metrics.Collector\n")
	builder.WriteString("    Subscriptions subscriptions.Broker\n")
	builder.WriteString("}\n\n")
	builder.WriteString("type Resolver struct {\n")
	builder.WriteString("    ORM           *gen.Client\n")
	builder.WriteString("    collector     metrics.Collector\n")
	builder.WriteString("    subscriptions subscriptions.Broker\n")
	builder.WriteString("    hooks         entityHooks\n")
	builder.WriteString("}\n\n")
	builder.WriteString("func New(orm *gen.Client) *Resolver {\n")
	builder.WriteString("    return NewWithOptions(Options{ORM: orm})\n}\n\n")
	builder.WriteString("func NewWithOptions(opts Options) *Resolver {\n")
	builder.WriteString("    collector := opts.Collector\n")
	builder.WriteString("    if collector == nil {\n        collector = metrics.NoopCollector{}\n    }\n")
	builder.WriteString("    resolver := &Resolver{ORM: opts.ORM, collector: collector, subscriptions: opts.Subscriptions}\n")
	builder.WriteString("    resolver.hooks = newEntityHooks()\n")
	builder.WriteString("    return resolver\n}\n\n")
	builder.WriteString("func (r *Resolver) WithLoaders(ctx context.Context) context.Context {\n")
	builder.WriteString("    if r == nil || r.ORM == nil {\n        return ctx\n    }\n")
	builder.WriteString("    loaders := dataloaders.New(r.ORM, r.collector)\n")
	builder.WriteString("    return dataloaders.ToContext(ctx, loaders)\n}\n\n")
	builder.WriteString("func (r *Resolver) Mutation() graphql.MutationResolver { return &mutationResolver{r} }\n")
	builder.WriteString("func (r *Resolver) Query() graphql.QueryResolver       { return &queryResolver{r} }\n")
	builder.WriteString("func (r *Resolver) Subscription() graphql.SubscriptionResolver {\n")
	builder.WriteString("    return &subscriptionResolver{r}\n}\n\n")
	builder.WriteString("type mutationResolver struct{ *Resolver }\n")
	builder.WriteString("type queryResolver struct{ *Resolver }\n")
	builder.WriteString("type subscriptionResolver struct{ *Resolver }\n\n")
	builder.WriteString("func (r *Resolver) subscriptionBroker() subscriptions.Broker {\n")
	builder.WriteString("    if r == nil {\n        return nil\n    }\n")
	builder.WriteString("    return r.subscriptions\n}\n")

	builder.WriteString("\nconst defaultPageSize = 50\n\n")
	builder.WriteString("func encodeCursor(offset int) string {\n")
	builder.WriteString("    return strconv.Itoa(offset)\n}\n\n")
	builder.WriteString("func decodeCursor(cursor string) (int, error) {\n")
	builder.WriteString("    return strconv.Atoi(cursor)\n}\n\n")
	builder.WriteString("type SubscriptionTrigger string\n\n")
	builder.WriteString("const (\n")
	builder.WriteString("    SubscriptionTriggerCreated SubscriptionTrigger = \"created\"\n")
	builder.WriteString("    SubscriptionTriggerUpdated SubscriptionTrigger = \"updated\"\n")
	builder.WriteString("    SubscriptionTriggerDeleted SubscriptionTrigger = \"deleted\"\n")
	builder.WriteString(")\n\n")
	builder.WriteString("var ErrSubscriptionsDisabled = errors.New(\"graphql subscriptions disabled\")\n\n")
	builder.WriteString("func publishSubscriptionEvent(ctx context.Context, broker subscriptions.Broker, entity string, trigger SubscriptionTrigger, payload any) {\n")
	builder.WriteString("    if broker == nil || entity == \"\" {\n        return\n    }\n")
	builder.WriteString("    _ = broker.Publish(ctx, subscriptionTopic(entity, trigger), payload)\n}\n\n")
	builder.WriteString("func subscribeToEntity(ctx context.Context, broker subscriptions.Broker, entity string, trigger SubscriptionTrigger) (<-chan any, func(), error) {\n")
	builder.WriteString("    if broker == nil {\n        return nil, nil, ErrSubscriptionsDisabled\n    }\n")
	builder.WriteString("    stream, cancel, err := broker.Subscribe(ctx, subscriptionTopic(entity, trigger))\n")
	builder.WriteString("    if err != nil {\n        return nil, nil, err\n    }\n")
	builder.WriteString("    return stream, cancel, nil\n}\n\n")
	builder.WriteString("func subscriptionTopic(entity string, trigger SubscriptionTrigger) string {\n")
	builder.WriteString("    base := strings.ToLower(entity)\n")
	builder.WriteString("    if base == \"\" {\n        base = \"entity\"\n    }\n")
	builder.WriteString("    return base + \":\" + string(trigger)\n}\n\n")
	builder.WriteString("func Topic(entity string, trigger SubscriptionTrigger) string {\n")
	builder.WriteString("    return subscriptionTopic(entity, trigger)\n}\n")
	builder.WriteString("\nfunc (r *mutationResolver) Noop(context.Context) (*bool, error) {\n")
	builder.WriteString("    value := true\n")
	builder.WriteString("    return &value, nil\n}\n\n")
	builder.WriteString("func (r *queryResolver) Health(context.Context) (string, error) {\n")
	builder.WriteString("    return \"ok\", nil\n}\n\n")
	builder.WriteString("func (r *subscriptionResolver) Noop(context.Context) (<-chan *bool, error) {\n")
	builder.WriteString("    ch := make(chan *bool)\n")
	builder.WriteString("    close(ch)\n")
	builder.WriteString("    return ch, nil\n}\n")

	path := filepath.Join(root, "graphql", "resolvers", "resolver.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir resolver runtime: %v", err)
	}
	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		t.Fatalf("write resolver runtime: %v", err)
	}
}

func writeGraphQLRelayRuntime(t *testing.T, root string) {
	t.Helper()

	builder := &strings.Builder{}
	builder.WriteString("package relay\n\n")
	builder.WriteString("import (\n")
	builder.WriteString("    \"encoding/base64\"\n")
	builder.WriteString("    \"fmt\"\n")
	builder.WriteString("    \"strings\"\n")
	builder.WriteString(")\n\n")
	builder.WriteString("func ToGlobalID(typ, id string) string {\n")
	builder.WriteString("    return base64.StdEncoding.EncodeToString([]byte(typ + \":\" + id))\n}\n\n")
	builder.WriteString("func FromGlobalID(value string) (string, string, error) {\n")
	builder.WriteString("    decoded, err := base64.StdEncoding.DecodeString(value)\n")
	builder.WriteString("    if err != nil {\n        return \"\", \"\", err\n    }\n")
	builder.WriteString("    parts := strings.SplitN(string(decoded), \":\", 2)\n")
	builder.WriteString("    if len(parts) != 2 {\n        return \"\", \"\", fmt.Errorf(\"invalid relay id: %s\", value)\n    }\n")
	builder.WriteString("    return parts[0], parts[1], nil\n}\n\n")
	builder.WriteString("func MarshalID(typ, id string) string {\n")
	builder.WriteString("    return ToGlobalID(typ, id)\n}\n\n")
	builder.WriteString("func UnmarshalID(value string) (string, string, error) {\n")
	builder.WriteString("    return FromGlobalID(value)\n}\n")

	path := filepath.Join(root, "graphql", "relay", "id.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir relay runtime: %v", err)
	}
	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		t.Fatalf("write relay runtime: %v", err)
	}
}
