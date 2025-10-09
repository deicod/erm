package generator

import (
	"fmt"
	"path/filepath"
)

func Run(root string, opts GeneratorOptions) error {
	entities, err := loadEntities(root)
	if err != nil {
		return err
	}
	if err := writeORMArtifacts(root, entities); err != nil {
		return err
	}
	if err := writeGraphQLArtifacts(root, entities); err != nil {
		return err
	}
	if err := runGQLGen(root); err != nil {
		return err
	}
	result, err := generateMigrations(root, entities, opts)
	if err != nil {
		return err
	}
	if opts.DryRun {
		if len(result.Operations) == 0 {
			fmt.Println("generator: no schema changes detected (dry-run)")
		} else {
			fmt.Println("generator: migration dry-run preview")
			fmt.Println(result.SQL)
		}
	} else {
		if result.FilePath != "" {
			fmt.Printf("generator: wrote migration %s\n", filepath.Base(result.FilePath))
		} else {
			fmt.Println("generator: no schema changes detected")
		}
	}
	fmt.Println("generator: wrote ORM and GraphQL artifacts")
	return nil
}
