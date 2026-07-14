package hub

import (
	"fmt"
	"path/filepath"
	"runtime"
)

type CommandSpec struct {
	Executable string
	Args       []string
	Dir        string
}

func BuildCommand(request StartRequest) (CommandSpec, error) {
	turns := request.MaxTurns
	if turns == 0 {
		turns = 12
	}
	if turns < 1 || turns > 50 {
		return CommandSpec{}, fmt.Errorf("maxTurns must be between 1 and 50")
	}

	workspace, err := filepath.Abs(request.Workspace)
	if err != nil {
		return CommandSpec{}, fmt.Errorf("resolve workspace: %w", err)
	}

	switch request.Provider {
	case ProviderGrok:
		args := []string{"--single", request.Task, "--output-format", "json", "--cwd", workspace, "--max-turns", fmt.Sprint(turns), "--disable-web-search", "--no-subagents", "--no-memory"}
		if request.Mode == ModeReadOnly {
			args = append(args, "--permission-mode", "plan")
		} else {
			args = append(args, "--permission-mode", "acceptEdits")
		}
		return CommandSpec{Executable: "grok", Args: args, Dir: workspace}, nil
	case ProviderCursor:
		args := []string{"--workspace", workspace, "--trust", "--print", request.Task, "--output-format", "json"}
		if request.Mode == ModeReadOnly {
			args = append(args, "--mode", "plan")
		} else {
			if runtime.GOOS == "windows" {
				return CommandSpec{}, fmt.Errorf("cursor write tasks are disabled on native Windows because Cursor sandboxing is unavailable; use Grok for writes")
			}
			args = append(args, "--sandbox", "enabled")
		}
		return CommandSpec{Executable: "cursor-agent.cmd", Args: args, Dir: workspace}, nil
	default:
		return CommandSpec{}, fmt.Errorf("provider must be grok or cursor")
	}
}
