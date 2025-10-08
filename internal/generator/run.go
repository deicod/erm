package generator

import "fmt"

func Run(root string) error {
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
	if err := ensureMigrationsPlaceholder(root, entities); err != nil {
		return err
	}
	fmt.Println("generator: wrote ORM and GraphQL artifacts")
	return nil
}
