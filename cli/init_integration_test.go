package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInitCmdAPIScaffoldCompiles(t *testing.T) {
	tmp := t.TempDir()
	modulePath := "github.com/example/app"

	goMod := "module " + modulePath + "\n\ngo 1.21\n\nrequire (\n\tgithub.com/99designs/gqlgen v0.17.80\n\tgithub.com/vektah/gqlparser/v2 v2.5.30\n)\n"
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cmd := newInitCmd()
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("execute init: %v", err)
	}

	scaffoldRuntimeDependencies(t, tmp, modulePath)

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmp
	tidyCmd.Env = append(os.Environ(), "GO111MODULE=on")
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, output)
	}

	buildCmd := exec.Command("go", "build", "./cmd/api")
	buildCmd.Dir = tmp
	buildCmd.Env = append(os.Environ(), "GO111MODULE=on")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/api: %v\n%s", err, output)
	}
}
