package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new <Entity>",
		Short: "Scaffold a new entity schema",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if name == "" {
				return fmt.Errorf("entity name required")
			}
			file := filepath.Join("schema", fmt.Sprintf("%s.schema.go", name))
			if _, err := os.Stat(file); err == nil {
				return fmt.Errorf("file exists: %s", file)
			}
			content := strings.ReplaceAll(schemaTemplate, "{{Entity}}", name)
			if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
				return err
			}
			fmt.Println("Created", file)
			return nil
		},
	}
	return cmd
}

var schemaTemplate = `package schema

import "github.com/deicod/erm/internal/orm/dsl"

type {{Entity}} struct{ dsl.Schema }

func ({{Entity}}) Fields() []dsl.Field {
    return []dsl.Field{
        dsl.UUIDv7("id").Primary(),
        dsl.TimestampTZ("created_at").DefaultNow(),
        dsl.TimestampTZ("updated_at").UpdateNow(),
    }
}

func ({{Entity}}) Edges() []dsl.Edge { return nil }
func ({{Entity}}) Indexes() []dsl.Index { return nil }
`
