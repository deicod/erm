package generator

import (
	"os"
	"path/filepath"
	"strings"
)

func patchJSONScalarWrappers(root string) error {
	path := filepath.Join(root, "graphql", "generated.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	original := string(data)
	updated := original

	replacements := map[string]string{
		"func (ec *executionContext) unmarshalNJSONB2encodingᚋjsonᚐRawMessage(ctx context.Context, v any) (json.RawMessage, error) {\n\tres, err := ec.unmarshalInputJSONB(ctx, v)\n\treturn &res, graphql.ErrorOnPath(ctx, err)\n}\n":                                            "func (ec *executionContext) unmarshalNJSONB2encodingᚋjsonᚐRawMessage(ctx context.Context, v any) (json.RawMessage, error) {\n\tres, err := ec.unmarshalInputJSONB(ctx, v)\n\treturn res, graphql.ErrorOnPath(ctx, err)\n}\n",
		"func (ec *executionContext) unmarshalOJSONB2encodingᚋjsonᚐRawMessage(ctx context.Context, v any) (json.RawMessage, error) {\n\tif v == nil {\n\t\treturn nil, nil\n\t}\n\tres, err := ec.unmarshalInputJSONB(ctx, v)\n\treturn &res, graphql.ErrorOnPath(ctx, err)\n}\n": "func (ec *executionContext) unmarshalOJSONB2encodingᚋjsonᚐRawMessage(ctx context.Context, v any) (json.RawMessage, error) {\n\tif v == nil {\n\t\treturn nil, nil\n\t}\n\tres, err := ec.unmarshalInputJSONB(ctx, v)\n\treturn res, graphql.ErrorOnPath(ctx, err)\n}\n",
		"func (ec *executionContext) unmarshalNJSON2encodingᚋjsonᚐRawMessage(ctx context.Context, v any) (json.RawMessage, error) {\n\tres, err := ec.unmarshalInputJSON(ctx, v)\n\treturn &res, graphql.ErrorOnPath(ctx, err)\n}\n":                                              "func (ec *executionContext) unmarshalNJSON2encodingᚋjsonᚐRawMessage(ctx context.Context, v any) (json.RawMessage, error) {\n\tres, err := ec.unmarshalInputJSON(ctx, v)\n\treturn res, graphql.ErrorOnPath(ctx, err)\n}\n",
		"func (ec *executionContext) unmarshalOJSON2encodingᚋjsonᚐRawMessage(ctx context.Context, v any) (json.RawMessage, error) {\n\tif v == nil {\n\t\treturn nil, nil\n\t}\n\tres, err := ec.unmarshalInputJSON(ctx, v)\n\treturn &res, graphql.ErrorOnPath(ctx, err)\n}\n":   "func (ec *executionContext) unmarshalOJSON2encodingᚋjsonᚐRawMessage(ctx context.Context, v any) (json.RawMessage, error) {\n\tif v == nil {\n\t\treturn nil, nil\n\t}\n\tres, err := ec.unmarshalInputJSON(ctx, v)\n\treturn res, graphql.ErrorOnPath(ctx, err)\n}\n",
	}

	for old, replacement := range replacements {
		if strings.Contains(updated, old) {
			updated = strings.ReplaceAll(updated, old, replacement)
		}
	}

	if updated == original {
		return nil
	}
	return os.WriteFile(path, []byte(updated), 0o644)
}
