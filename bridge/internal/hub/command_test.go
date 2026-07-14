package hub

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestBuildGrokReadOnlyCommand(t *testing.T) {
	workspace := t.TempDir()
	spec, err := BuildCommand(StartRequest{Provider: ProviderGrok, Mode: ModeReadOnly, Workspace: workspace, Task: "inspect the code", MaxTurns: 7})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--single", "inspect the code", "--output-format", "json", "--cwd", filepath.Clean(workspace), "--max-turns", "7", "--disable-web-search", "--no-subagents", "--no-memory", "--permission-mode", "plan"}
	if spec.Executable != "grok" || !reflect.DeepEqual(spec.Args, want) {
		t.Fatalf("unexpected command: %#v", spec)
	}
}

func TestBuildGrokWriteCommandUsesAcceptEdits(t *testing.T) {
	spec, err := BuildCommand(StartRequest{Provider: ProviderGrok, Mode: ModeWrite, Workspace: t.TempDir(), Task: "make a scoped edit"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := spec.Args[len(spec.Args)-2:], []string{"--permission-mode", "acceptEdits"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("permission args: got %v want %v", got, want)
	}
}

func TestBuildCursorWriteDoesNotForceApproval(t *testing.T) {
	spec, err := BuildCommand(StartRequest{Provider: ProviderCursor, Mode: ModeWrite, Workspace: t.TempDir(), Task: "edit"})
	if runtime.GOOS == "windows" {
		if err == nil {
			t.Fatal("expected Cursor writes to be rejected without a native Windows sandbox")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, arg := range spec.Args {
		if arg == "--force" || arg == "--yolo" {
			t.Fatalf("unsafe default argument %q", arg)
		}
	}
}

func TestBuildCursorReadOnlyCommand(t *testing.T) {
	workspace := t.TempDir()
	spec, err := BuildCommand(StartRequest{Provider: ProviderCursor, Mode: ModeReadOnly, Workspace: workspace, Task: "inspect"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--workspace", filepath.Clean(workspace), "--trust", "--print", "inspect", "--output-format", "json", "--mode", "plan"}
	if spec.Executable != "cursor-agent.cmd" || !reflect.DeepEqual(spec.Args, want) {
		t.Fatalf("unexpected command: %#v", spec)
	}
}
