package generator

import "slices"

type GenerateOptions struct {
	DryRun        bool
	Force         bool
	MigrationName string
	Components    []string
}

func (opts GenerateOptions) includes(component string) bool {
	if len(opts.Components) == 0 {
		return true
	}
	return slices.Contains(opts.Components, component)
}

var forceWrite bool

func Run(root string, opts GenerateOptions) (MigrationResult, error) {
	prevForce := forceWrite
	forceWrite = opts.Force
	defer func() { forceWrite = prevForce }()

	entities, err := loadEntities(root)
	if err != nil {
		return MigrationResult{}, err
	}
	if !opts.DryRun && opts.includes("orm") {
		if err := writeORMArtifacts(root, entities); err != nil {
			return MigrationResult{}, err
		}
	}
	if !opts.DryRun && opts.includes("graphql") {
		if err := writeGraphQLArtifacts(root, entities); err != nil {
			return MigrationResult{}, err
		}
		if err := runGQLGen(root); err != nil {
			return MigrationResult{}, err
		}
	}
	var result MigrationResult
	if opts.includes("migrations") {
		var err error
		result, err = generateMigrations(root, entities, generatorOptions{GenerateOptions: opts})
		if err != nil {
			return MigrationResult{}, err
		}
	}
	return result, nil
}
