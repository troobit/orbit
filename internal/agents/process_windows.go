package agents

import "os/exec"

// Windows retains exec.CommandContext's direct-process cancellation.
func configureProcessGroup(_ *exec.Cmd) func() {
	return func() {}
}
