package generator

import (
	"path/filepath"
	"sort"
)

type artifactTracker struct {
	root   string
	files  map[ComponentName]map[string]struct{}
	active *componentPlan
	next   fileWriter
}

func newArtifactTracker(root string) *artifactTracker {
	return &artifactTracker{
		root:  root,
		files: make(map[ComponentName]map[string]struct{}),
		next:  defaultFileWriter{},
	}
}

func (t *artifactTracker) begin(plan *componentPlan) {
	t.active = plan
}

func (t *artifactTracker) end() {
	t.active = nil
}

func (t *artifactTracker) Write(path string, content []byte) (bool, error) {
	wrote, err := t.next.Write(path, content)
	if err != nil {
		return false, err
	}
	if wrote {
		t.record(path)
	}
	return wrote, nil
}

func (t *artifactTracker) record(path string) {
	plan := t.active
	if plan == nil {
		return
	}
	rel := path
	if base := t.root; base != "" {
		if r, err := filepath.Rel(base, path); err == nil {
			rel = r
		}
	}
	rel = filepath.ToSlash(rel)
	if t.files[plan.Name] == nil {
		t.files[plan.Name] = make(map[string]struct{})
	}
	t.files[plan.Name][rel] = struct{}{}
}

func (t *artifactTracker) filesFor(name ComponentName) []string {
	entries := t.files[name]
	if len(entries) == 0 {
		return nil
	}
	out := make([]string, 0, len(entries))
	for path := range entries {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}
