package cli

import "embed"

type runtimeTemplate struct {
	source     string
	target     string
	isTemplate bool
}

//go:embed templates/graphql/**
var graphqlTemplateFS embed.FS

var graphqlRuntimeTemplates = []runtimeTemplate{
	{source: "templates/graphql/dataloaders/loader.go.tmpl", target: "graphql/dataloaders/loader.go", isTemplate: true},
	{source: "templates/graphql/directives/auth.go.tmpl", target: "graphql/directives/auth.go", isTemplate: true},
	{source: "templates/graphql/relay/id.go", target: "graphql/relay/id.go"},
	{source: "templates/graphql/server/schema.go.tmpl", target: "graphql/server/schema.go", isTemplate: true},
	{source: "templates/graphql/server/server.go", target: "graphql/server/server.go"},
	{source: "templates/graphql/subscriptions/bus.go", target: "graphql/subscriptions/bus.go"},
}
