package generator

type GenerateOptions struct {
	DryRun        bool
	Force         bool
	MigrationName string
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
	if !opts.DryRun {
		if err := writeORMArtifacts(root, entities); err != nil {
			return MigrationResult{}, err
		}
		if err := writeGraphQLArtifacts(root, entities); err != nil {
			return MigrationResult{}, err
		}
		if err := runGQLGen(root); err != nil {
			return MigrationResult{}, err
		}
	}
	result, err := generateMigrations(root, entities, generatorOptions{GenerateOptions: opts})
	if err != nil {
		return MigrationResult{}, err
	}
	return result, nil
}
