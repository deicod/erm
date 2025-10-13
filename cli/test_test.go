package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

func TestTestCmdRunsGoTest(t *testing.T) {
	originalGoTest := goTestCommand
	originalGoList := goListPackages
	defer func() {
		goTestCommand = originalGoTest
		goListPackages = originalGoList
	}()

	var capturedArgs [][]string
	goTestCommand = func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
		captured := make([]string, len(args))
		copy(captured, args)
		capturedArgs = append(capturedArgs, captured)
		return nil
	}
	goListPackages = func(ctx context.Context, patterns []string) ([]string, error) {
		return nil, errors.New("unexpected go list call")
	}

	cmd := newTestCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.RunE(cmd, []string{"./orm/..."}); err != nil {
		t.Fatalf("run test command: %v", err)
	}

	if len(capturedArgs) != 1 {
		t.Fatalf("expected 1 go test invocation, got %d", len(capturedArgs))
	}
	want := []string{"test", "./orm/..."}
	got := capturedArgs[0]
	if len(got) != len(want) {
		t.Fatalf("unexpected go test args: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("go test arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("==> go test ./orm/...")) {
		t.Fatalf("expected command banner in output, got: %s", output)
	}
}

func TestTestCmdRaceBatchesPackages(t *testing.T) {
	originalGoTest := goTestCommand
	originalGoList := goListPackages
	defer func() {
		goTestCommand = originalGoTest
		goListPackages = originalGoList
	}()

	goListPackages = func(ctx context.Context, patterns []string) ([]string, error) {
		return []string{"github.com/deicod/erm/a", "github.com/deicod/erm/b", "github.com/deicod/erm/c"}, nil
	}
	var batches [][]string
	goTestCommand = func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
		captured := make([]string, len(args))
		copy(captured, args)
		batches = append(batches, captured)
		return nil
	}

	cmd := newTestCmd()
	if err := cmd.Flags().Set("race", "true"); err != nil {
		t.Fatalf("set race flag: %v", err)
	}
	if err := cmd.Flags().Set("batch-size", "2"); err != nil {
		t.Fatalf("set batch-size: %v", err)
	}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.RunE(cmd, []string{"./..."}); err != nil {
		t.Fatalf("run race batches: %v", err)
	}

	if len(batches) != 2 {
		t.Fatalf("expected 2 batches, got %d", len(batches))
	}
	first := batches[0]
	second := batches[1]
	wantFirst := []string{"test", "-race", "github.com/deicod/erm/a", "github.com/deicod/erm/b"}
	wantSecond := []string{"test", "-race", "github.com/deicod/erm/c"}
	if !equalArgs(first, wantFirst) {
		t.Fatalf("first batch = %v, want %v", first, wantFirst)
	}
	if !equalArgs(second, wantSecond) {
		t.Fatalf("second batch = %v, want %v", second, wantSecond)
	}
	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("==> go test -race github.com/deicod/erm/a github.com/deicod/erm/b")) {
		t.Fatalf("expected first batch banner, got: %s", output)
	}
	if !bytes.Contains(buf.Bytes(), []byte("==> go test -race github.com/deicod/erm/c")) {
		t.Fatalf("expected second batch banner, got: %s", output)
	}
}

func equalArgs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
