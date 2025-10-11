package generator

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestRunSkipsUnchangedComponents(t *testing.T) {
	dir := t.TempDir()
	copyTree(t, filepath.Join("testdata", "default_fk"), dir)

	opts := GenerateOptions{Components: []string{string(ComponentORM)}}
	res, err := Run(dir, opts)
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	orm := findComponent(res.Components, ComponentORM)
	if orm == nil {
		t.Fatalf("expected orm component in result")
	}
	if !orm.Changed {
		t.Fatalf("expected orm to be marked as changed on first run")
	}
	if orm.Skipped {
		t.Fatalf("expected orm generation to execute on first run")
	}
	if len(orm.Files) == 0 {
		t.Fatalf("expected orm files to be written on first run")
	}

	res2, err := Run(dir, opts)
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	orm2 := findComponent(res2.Components, ComponentORM)
	if orm2 == nil {
		t.Fatalf("expected orm component in second result")
	}
	if orm2.Changed {
		t.Fatalf("expected orm to be unchanged on second run")
	}
	if !orm2.Skipped {
		t.Fatalf("expected orm generation to be skipped on second run")
	}
	if orm2.Reason != "up-to-date" {
		t.Fatalf("unexpected skip reason: %q", orm2.Reason)
	}
	if len(orm2.Files) != 0 {
		t.Fatalf("expected no files to be written on second run, got %v", orm2.Files)
	}
}

func TestRunStagesUnchangedComponents(t *testing.T) {
	dir := t.TempDir()
	copyTree(t, filepath.Join("testdata", "default_fk"), dir)

	opts := GenerateOptions{Components: []string{string(ComponentORM)}}
	if _, err := Run(dir, opts); err != nil {
		t.Fatalf("seed run failed: %v", err)
	}

	stageDir := filepath.Join(".erm", "stage")
	stageOpts := GenerateOptions{Components: []string{string(ComponentORM)}, StagingDir: stageDir}
	res, err := Run(dir, stageOpts)
	if err != nil {
		t.Fatalf("staging run failed: %v", err)
	}
	orm := findComponent(res.Components, ComponentORM)
	if orm == nil {
		t.Fatalf("expected orm component in staged result")
	}
	if !orm.Staged {
		t.Fatalf("expected orm to be staged")
	}
	if orm.Skipped {
		t.Fatalf("expected staged generation to execute")
	}
	if len(orm.Files) == 0 {
		t.Fatalf("expected staged files to be recorded")
	}
	expectedFile := filepath.Join(dir, stageDir, string(ComponentORM), "orm", "gen", "registry_gen.go")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Fatalf("expected staged file %s to exist: %v", expectedFile, err)
	}
}

func TestRunIdempotentExamplesBlog(t *testing.T) {
	dir := t.TempDir()
	copyTree(t, filepath.Join("..", "examples", "blog"), dir)

	original := gqlRunner
	gqlRunner = func(string) error { return nil }
	defer func() { gqlRunner = original }()

	res1, err := Run(dir, GenerateOptions{})
	if err != nil {
		t.Fatalf("first blog run failed: %v", err)
	}
	if res1.Migration.FilePath == "" {
		t.Fatalf("expected migration to be written on first run")
	}

	res2, err := Run(dir, GenerateOptions{})
	if err != nil {
		t.Fatalf("second blog run failed: %v", err)
	}
	orm := findComponent(res2.Components, ComponentORM)
	if orm == nil || !orm.Skipped || orm.Reason != "up-to-date" {
		t.Fatalf("expected orm to be skipped as up-to-date, got %+v", orm)
	}
	graphql := findComponent(res2.Components, ComponentGraphQL)
	if graphql == nil || !graphql.Skipped || graphql.Reason != "up-to-date" {
		t.Fatalf("expected graphql to be skipped as up-to-date, got %+v", graphql)
	}
	if len(res2.Migration.Operations) != 0 {
		t.Fatalf("expected no migration operations on second run, got %d", len(res2.Migration.Operations))
	}
	if res2.Migration.FilePath != "" {
		t.Fatalf("expected no migration file on second run, got %s", res2.Migration.FilePath)
	}
}

func findComponent(components []ComponentResult, name ComponentName) *ComponentResult {
	for i := range components {
		if components[i].Name == name {
			return &components[i]
		}
	}
	return nil
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			if rel == "." {
				return nil
			}
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copyTree(%s -> %s): %v", src, dst, err)
	}
}
