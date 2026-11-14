package agents

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// ExecuteConfig holds the parameters for running an agent CLI command.
type ExecuteConfig struct {
	// CLIPath is the path or name of the CLI binary.
	CLIPath string
	// Args are the command-line arguments (not including the binary name).
	Args []string
	// WorkDir sets the working directory for the command. Empty means inherit.
	WorkDir string
	// Env provides additional environment variables to merge with os.Environ().
	// When nil or empty, the command inherits the current environment unchanged.
	Env map[string]string
	// Timeout applies a per-execution timeout via context.WithTimeout.
	// If > 0, the command is killed when the timeout elapses.
	// 0 means no timeout (inherit parent context deadline only).
	Timeout      time.Duration
	Stdout       io.Writer
	Stderr       io.Writer
	ProcessGroup bool
}

// ExecuteResult holds the raw output from running an agent CLI command.
type ExecuteResult struct {
	// Stdout is the captured standard output.
	Stdout []byte
	// Stderr is the captured standard error.
	Stderr []byte
	// ExitCode is the process exit code: 0 for success, -1 if the exit code
	// could not be determined (e.g. the command was not found).
	ExitCode int
	// Duration is the wall-clock time the command took.
	Duration time.Duration
	// Err is the raw error returned by cmd.Run(), nil on success.
	Err error
}

// Execute runs a CLI command with stdout/stderr capture, timing, and exit code
// extraction. This is the shared execution scaffolding used by all agents.
//
// The caller is responsible for building the args (agent-specific) and
// post-processing the result (parsing output, extracting costs, etc.).
func Execute(ctx context.Context, cfg ExecuteConfig) *ExecuteResult {
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, cfg.CLIPath, cfg.Args...)
	if cfg.ProcessGroup {
		cleanup := configureProcessGroup(cmd)
		defer cleanup()
	}

	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

	if len(cfg.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range cfg.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if cfg.Stdout != nil {
		cmd.Stdout = io.MultiWriter(&stdout, cfg.Stdout)
	}
	if cfg.Stderr != nil {
		cmd.Stderr = io.MultiWriter(&stderr, cfg.Stderr)
	}
	cmd.Stdin = nil // Prevent agents from blocking on interactive input

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)
	if cfg.ProcessGroup && ctx.Err() != nil {
		err = fmt.Errorf("agent execution stopped after %s: %w (timeout: %s)", duration.Round(time.Millisecond), ctx.Err(), cfg.Timeout)
	}

	result := &ExecuteResult{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		Duration: duration,
		Err:      err,
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}

	return result
}
