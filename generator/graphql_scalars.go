package generator

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
)

var (
	graphqlScalarTemplate []byte
	graphqlScalarMu       sync.RWMutex
)

// RegisterGraphQLScalarTemplate provides the default scalars helper template used when
// ensuring GraphQL runtime helpers are present during generation. The content is copied
// to avoid callers mutating the registered template after registration.
func RegisterGraphQLScalarTemplate(content []byte) {
	graphqlScalarMu.Lock()
	defer graphqlScalarMu.Unlock()

	if len(content) == 0 {
		graphqlScalarTemplate = nil
		return
	}
	graphqlScalarTemplate = append([]byte(nil), content...)
}

func ensureGraphQLScalarHelpers(root string) error {
	graphqlScalarMu.RLock()
	template := graphqlScalarTemplate
	graphqlScalarMu.RUnlock()

	if len(template) == 0 {
		return nil
	}

	path := filepath.Join(root, "graphql", "scalars.go")
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	_, err := writeFile(path, template)
	return err
}
