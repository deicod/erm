package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

type GenerateOptions struct {
	DryRun        bool
	Force         bool
	MigrationName string
	Components    []string
	StagingDir    string
}

func (opts GenerateOptions) includes(component string) bool {
	if len(opts.Components) == 0 {
		return true
	}
	return slices.Contains(opts.Components, component)
}

var (
	forceWrite bool
	gqlRunner  = runGQLGen
)

type ComponentResult struct {
	Name    ComponentName
	Changed bool
	Skipped bool
	Staged  bool
	Files   []string
	Reason  string
}

type RunResult struct {
	Components []ComponentResult
	Migration  MigrationResult
}

type componentPlan struct {
	Name      ComponentName
	InputHash string
	Changed   bool
	Enabled   bool
	Stage     bool
	WriteRoot string
	Reason    string
}

func Run(root string, opts GenerateOptions) (RunResult, error) {
	prevForce := forceWrite
	forceWrite = opts.Force
	defer func() { forceWrite = prevForce }()

	result := RunResult{}

	state, err := loadGeneratorState(root)
	if err != nil {
		return result, err
	}
	if state.Components == nil {
		state.Components = make(map[ComponentName]componentState)
	}

	entities, err := loadEntities(root)
	if err != nil {
		return result, err
	}

	baseHash, err := schemaInputHash(entities)
	if err != nil {
		return result, err
	}

	plans := make(map[ComponentName]*componentPlan)
	for _, name := range []ComponentName{ComponentORM, ComponentGraphQL} {
		plan := buildComponentPlan(root, opts, state, baseHash, name)
		plans[name] = &plan
	}

	tracker, restoreWriter := activateTracker(root, opts, plans)
	if restoreWriter != nil {
		defer restoreWriter()
	}

	var modulePath string
	for _, name := range []ComponentName{ComponentORM, ComponentGraphQL} {
		plan := plans[name]
		if plan == nil || !plan.Enabled {
			continue
		}
		if plan.Stage {
			if err := os.RemoveAll(plan.WriteRoot); err != nil {
				return result, err
			}
		}
		if tracker != nil {
			tracker.begin(plan)
		}
		var genErr error
		switch name {
		case ComponentORM:
			genErr = writeORMArtifacts(plan.WriteRoot, entities)
		case ComponentGraphQL:
			if modulePath == "" {
				modulePath, err = detectModulePath(root)
				if err != nil {
					genErr = err
					break
				}
				if modulePath == "" {
					genErr = fmt.Errorf("module path not found; set module in erm.yaml or go.mod")
					break
				}
			}
			genErr = writeGraphQLArtifacts(plan.WriteRoot, entities, modulePath)
			if genErr == nil && !plan.Stage {
				genErr = gqlRunner(plan.WriteRoot)
			}
		}
		if tracker != nil {
			tracker.end()
		}
		if genErr != nil {
			return result, genErr
		}
		if opts.DryRun || plan.Stage {
			continue
		}
		state.Components[name] = componentState{InputHash: plan.InputHash}
	}

	if opts.includes(string(ComponentMigrations)) {
		migration, err := runMigrations(root, entities, opts)
		if err != nil {
			return result, err
		}
		result.Migration = migration
	} else {
		result.Migration = MigrationResult{}
	}

	result.Components = assembleComponentResults(plans, tracker)

	migResult := ComponentResult{Name: ComponentMigrations}
	if !opts.includes(string(ComponentMigrations)) {
		migResult.Skipped = true
		migResult.Reason = "filtered (--only)"
	} else {
		migResult.Changed = len(result.Migration.Operations) > 0
		if opts.DryRun {
			migResult.Reason = "dry-run"
		} else if len(result.Migration.Files) > 0 {
			paths := make([]string, 0, len(result.Migration.Files))
			for _, file := range result.Migration.Files {
				if file.Path == "" {
					continue
				}
				if rel, err := filepath.Rel(root, file.Path); err == nil {
					paths = append(paths, filepath.ToSlash(rel))
				} else {
					paths = append(paths, filepath.ToSlash(file.Path))
				}
			}
			migResult.Files = paths
		} else if len(result.Migration.Operations) == 0 {
			migResult.Reason = "up-to-date"
		}
	}
	result.Components = append(result.Components, migResult)

	if !opts.DryRun {
		if err := saveGeneratorState(root, state); err != nil {
			return result, err
		}
	}

	return result, nil
}

func buildComponentPlan(root string, opts GenerateOptions, state generatorState, baseHash string, name ComponentName) componentPlan {
	plan := componentPlan{Name: name, InputHash: componentInputHash(baseHash, name)}
	prev := state.Components[name]
	plan.Changed = opts.Force || prev.InputHash != plan.InputHash

	if !opts.includes(string(name)) {
		plan.Reason = "filtered (--only)"
		return plan
	}
	if opts.DryRun {
		plan.Reason = "dry-run"
		plan.Enabled = false
		return plan
	}

	if plan.Changed {
		plan.Enabled = true
		plan.WriteRoot = root
		return plan
	}

	if opts.StagingDir != "" {
		stageRoot := opts.StagingDir
		if !filepath.IsAbs(stageRoot) {
			stageRoot = filepath.Join(root, stageRoot)
		}
		plan.Stage = true
		plan.Enabled = true
		plan.WriteRoot = filepath.Join(stageRoot, string(name))
		plan.Reason = "staged"
		return plan
	}

	plan.Reason = "up-to-date"
	return plan
}

func detectModulePath(root string) (string, error) {
	if module, err := moduleFromConfig(root); err != nil {
		return "", err
	} else if module != "" {
		return module, nil
	}
	return moduleFromGoMod(root)
}

func moduleFromConfig(root string) (string, error) {
	path := filepath.Join(root, "erm.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	var cfg struct {
		Module string `yaml:"module"`
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return "", fmt.Errorf("parse %s: %w", path, err)
	}
	return strings.TrimSpace(cfg.Module), nil
}

func moduleFromGoMod(root string) (string, error) {
	path := filepath.Join(root, "go.mod")
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			module := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if module != "" {
				return module, nil
			}
		}
	}
	return "", nil
}

func activateTracker(root string, opts GenerateOptions, plans map[ComponentName]*componentPlan) (*artifactTracker, func()) {
	if opts.DryRun {
		return nil, nil
	}
	needsWriter := false
	for _, plan := range plans {
		if plan != nil && plan.Enabled {
			needsWriter = true
			break
		}
	}
	if !needsWriter {
		return nil, nil
	}
	tracker := newArtifactTracker(root)
	prev := activeWriter
	activeWriter = tracker
	return tracker, func() { activeWriter = prev }
}

func assembleComponentResults(plans map[ComponentName]*componentPlan, tracker *artifactTracker) []ComponentResult {
	out := make([]ComponentResult, 0, len(plans)+1)
	order := []ComponentName{ComponentORM, ComponentGraphQL}
	for _, name := range order {
		plan := plans[name]
		if plan == nil {
			continue
		}
		files := []string{}
		if tracker != nil {
			files = tracker.filesFor(name)
		}
		res := ComponentResult{
			Name:    name,
			Changed: plan.Changed,
			Staged:  plan.Stage,
			Files:   files,
		}
		if !plan.Enabled {
			res.Skipped = true
		}
		res.Reason = plan.Reason
		out = append(out, res)
	}
	return out
}

func runMigrations(root string, entities []Entity, opts GenerateOptions) (MigrationResult, error) {
	result, err := generateMigrations(root, entities, generatorOptions{GenerateOptions: opts})
	if err != nil {
		return MigrationResult{}, err
	}
	return result, nil
}
