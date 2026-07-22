//go:build !linux && !darwin

package checker

import "os/exec"

func configureCommand(cmd *exec.Cmd) {
	cmd.Cancel = func() error { return terminateProcessTree(cmd) }
}

func terminateProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
