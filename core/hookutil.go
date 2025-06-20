package core

import (
	"context"
	"os"
	"os/exec"
	"runtime"
)

func ExecCommandString(ctx context.Context, cmd string) error {
	if cmd == "" {
		return nil
	}
	var execCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		execCmd = exec.CommandContext(ctx, "cmd.exe", "/C", cmd)
	} else {
		execCmd = exec.CommandContext(ctx, "sh", "-c", cmd)
	}
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}
