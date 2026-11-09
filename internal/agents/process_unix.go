//go:build !windows

package agents

import (
	"os/exec"
	"syscall"
	"time"
)

// A separate process group keeps cancellation from leaving worker commands alive.
func configureProcessGroup(cmd *exec.Cmd) func() {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	kill := func() error {
		if cmd.Process == nil {
			return nil
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if err == syscall.ESRCH {
			return nil
		}
		return err
	}
	cmd.Cancel = kill
	cmd.WaitDelay = time.Second
	return func() { _ = kill() }
}
