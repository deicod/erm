package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitGenTidyWithoutManualScaffolds(t *testing.T) {
	tmpDir := t.TempDir()
	modulePath := "github.com/example/app"

	goMod := "module " + modulePath + "\n\ngo 1.21\n\nrequire (\n\tgithub.com/99designs/gqlgen v0.17.80\n\tgithub.com/vektah/gqlparser/v2 v2.5.30\n)\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	initCmd := newInitCmd()
	if err := initCmd.RunE(initCmd, []string{}); err != nil {
		t.Fatalf("execute init: %v", err)
	}

	newCmd := newNewCmd()
	if err := newCmd.RunE(newCmd, []string{"User"}); err != nil {
		t.Fatalf("execute new: %v", err)
	}

	download := exec.Command("go", "mod", "download")
	download.Dir = tmpDir
	download.Env = append(os.Environ(), "GO111MODULE=on")
	if output, err := download.CombinedOutput(); err != nil {
		t.Fatalf("go mod download: %v\n%s", err, output)
	}

	genCmd := newGenCmd()
	if err := genCmd.RunE(genCmd, []string{}); err != nil {
		cfg, readErr := os.ReadFile(filepath.Join(tmpDir, "graphql", "gqlgen.yml"))
		if readErr == nil {
			t.Logf("gqlgen.yml:\n%s", cfg)
		}
		t.Fatalf("execute gen: %v", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	tidyCmd.Env = append(os.Environ(), "GO111MODULE=on")
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, output)
	}

	buildCmd := exec.Command("go", "build", "./cmd/api")
	buildCmd.Dir = tmpDir
	buildCmd.Env = append(os.Environ(), "GO111MODULE=on")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/api: %v\n%s", err, output)
	}
}

func TestInitGenAutoTidyGraphQLDeps(t *testing.T) {
	tmpDir := t.TempDir()
	modulePath := "github.com/example/app"

	goMod := "module " + modulePath + "\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	initCmd := newInitCmd()
	if err := initCmd.RunE(initCmd, []string{}); err != nil {
		t.Fatalf("execute init: %v", err)
	}

	newCmd := newNewCmd()
	if err := newCmd.RunE(newCmd, []string{"User"}); err != nil {
		t.Fatalf("execute new: %v", err)
	}

	genCmd := newGenCmd()
	if err := genCmd.RunE(genCmd, []string{}); err != nil {
		cfg, readErr := os.ReadFile(filepath.Join(tmpDir, "graphql", "gqlgen.yml"))
		if readErr == nil {
			t.Logf("gqlgen.yml:\n%s", cfg)
		}
		t.Fatalf("execute gen: %v", err)
	}

	goModBytes, err := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	goModContents := string(goModBytes)
	for _, dep := range []string{"github.com/99designs/gqlgen", "github.com/vektah/gqlparser/v2"} {
		if !strings.Contains(goModContents, dep) {
			t.Fatalf("expected go.mod to include %s after generation\n%s", dep, goModContents)
		}
	}

	buildCmd := exec.Command("go", "build", "./cmd/api")
	buildCmd.Dir = tmpDir
	buildCmd.Env = append(os.Environ(), "GO111MODULE=on")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/api: %v\n%s", err, output)
	}
}
