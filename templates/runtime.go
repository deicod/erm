package templates

import (
	"bytes"
	"embed"
	"path/filepath"
	"text/template"
)

type runtimeTemplate struct {
	source     string
	target     string
	isTemplate bool
}

//go:embed graphql/** observability/** oidc/**
var runtimeFS embed.FS

var runtimeTemplates = []runtimeTemplate{
	{source: "graphql/dataloaders/loader.go.tmpl", target: "graphql/dataloaders/loader.go", isTemplate: true},
	{source: "graphql/dataloaders/entities_gen.go.tmpl", target: "graphql/dataloaders/entities_gen.go", isTemplate: true},
	{source: "graphql/directives/auth.go.tmpl", target: "graphql/directives/auth.go", isTemplate: true},
	{source: "graphql/generated.go", target: "graphql/generated.go"},
	{source: "graphql/relay/id.go", target: "graphql/relay/id.go"},
	{source: "graphql/scalars.go.tmpl", target: "graphql/scalars.go", isTemplate: true},
	{source: "graphql/types.go.tmpl", target: "graphql/types.go"},
	{source: "graphql/resolvers/resolver.go.tmpl", target: "graphql/resolvers/resolver.go", isTemplate: true},
	{source: "graphql/resolvers/entities_gen.go", target: "graphql/resolvers/entities_gen.go"},
	{source: "graphql/resolvers/entities_hooks.go.tmpl", target: "graphql/resolvers/entities_hooks.go", isTemplate: true},
	{source: "graphql/server/schema.go.tmpl", target: "graphql/server/schema.go", isTemplate: true},
	{source: "graphql/server/server.go", target: "graphql/server/server.go"},
	{source: "graphql/subscriptions/bus.go", target: "graphql/subscriptions/bus.go"},
	{source: "observability/metrics/metrics.go", target: "observability/metrics/metrics.go"},
	{source: "oidc/claims.go", target: "oidc/claims.go"},
}

// RenderRuntimeScaffolds renders the runtime scaffolds using the provided module path.
func RenderRuntimeScaffolds(modulePath string) (map[string][]byte, error) {
	runtime := make(map[string][]byte, len(runtimeTemplates))
	data := struct{ ModulePath string }{ModulePath: modulePath}
	for _, tpl := range runtimeTemplates {
		raw, err := runtimeFS.ReadFile(tpl.source)
		if err != nil {
			return nil, err
		}
		content := raw
		if tpl.isTemplate {
			t, err := template.New(filepath.Base(tpl.source)).Parse(string(raw))
			if err != nil {
				return nil, err
			}
			buf := &bytes.Buffer{}
			if err := t.Execute(buf, data); err != nil {
				return nil, err
			}
			content = buf.Bytes()
		}
		runtime[tpl.target] = content
	}
	return runtime, nil
}

// GraphQLScalarsTemplate returns the default scalars helper template.
func GraphQLScalarsTemplate() ([]byte, error) {
	return runtimeFS.ReadFile("graphql/scalars.go.tmpl")
}
