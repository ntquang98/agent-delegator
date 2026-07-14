package hub

import (
	"os/exec"
	"runtime"
	"strconv"
)

func killProcessTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F").Run()
		return
	}
	_ = cmd.Process.Kill()
}
