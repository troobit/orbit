package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/arjenschwarz/orbit/internal/agents"
	"github.com/arjenschwarz/orbit/internal/orbit"
)

// localCommand is a local preset of Orbit, independent of hosted user defaults.
func localCommand(args []string) error {
	fs := flag.NewFlagSet("local", flag.ContinueOnError)
	model := fs.String("model", "8-bit", "Exact LM Studio model identifier")
	selection := fs.String("selection", "runner", "Who retrieves work: runner or model")
	phase := fs.Bool("phase", false, "Select a whole phase per iteration")
	limit := fs.Int("max-iterations", 20, "Maximum worker sessions per invocation")
	timeout := fs.Duration("timeout", 30*time.Minute, "Timeout per worker session (e.g. 30m, 2h)")
	dryRun := fs.Bool("dry-run", false, "Print selected work and prompt without writing logs or running inference")
	fs.Usage = func() {
		_, _ = fmt.Fprintln(fs.Output(), "Usage: orbit local <tasks.md> [options]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(reorderArgs(fs, args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 || strings.TrimSpace(fs.Arg(0)) == "" {
		return fmt.Errorf("provide one explicit tasks file: orbit local <tasks.md>")
	}
	if strings.TrimSpace(*model) == "" || (*selection != "runner" && *selection != "model") || *limit < 1 || *timeout <= 0 {
		return fmt.Errorf("model must be nonempty, selection must be runner or model, and timeout and max-iterations must be positive")
	}
	tasks, root, err := resolveLocalTasks(fs.Arg(0))
	if err != nil {
		return err
	}
	for _, name := range []string{"rune", "codex"} {
		if _, err := exec.LookPath(name); err != nil {
			return fmt.Errorf("required command %s: %w", name, err)
		}
	}
	local := &orbit.LocalConfig{Selection: *selection, Phase: *phase, MaxIterations: *limit}
	if *selection == "model" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		local.SkillPath = filepath.Join(home, ".agents", "skills", "next-task", "SKILL.md")
		if info, err := os.Stat(local.SkillPath); err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("model selection requires the installed next-task skill: %s", local.SkillPath)
		}
	}
	branchCmd := exec.Command("git", "-C", root, "branch", "--show-current")
	branch, err := branchCmd.Output()
	if err != nil {
		return fmt.Errorf("get target branch: %w", err)
	}
	cfg := orbit.Config{
		TasksFile: tasks, WorkingDir: root, RepoRoot: root,
		LogDir:     filepath.Join(filepath.Dir(tasks), ".orbit", "local"),
		BranchName: strings.TrimSpace(string(branch)), Version: version,
		Agent: "local", Local: local, DryRun: *dryRun,
		AgentConfig: agents.AgentConfig{
			Type: "codex", Timeout: *timeout, Options: map[string]string{"model": *model},
			ExtraArgs: []string{"--oss", "--local-provider", "lmstudio", "--ignore-user-config",
				"--ephemeral", "--json", "--sandbox", "workspace-write", "-c", `approval_policy="never"`,
				// OSS mode retains Codex defaults for tools LM Studio does not support.
				"-c", `web_search="disabled"`, "-c", "features.multi_agent=false",
				"-c", `model_reasoning_summary="none"`},
		},
	}
	log.Printf("Local agent: codex / lmstudio / %s; selection: %s; timeout: %s", *model, *selection, *timeout)
	o, err := orbit.New(cfg)
	if err != nil {
		return err
	}
	defer o.Close()
	return o.Run()
}

func resolveLocalTasks(path string) (string, string, error) {
	tasks, err := filepath.Abs(path)
	if err == nil {
		tasks, err = filepath.EvalSymlinks(tasks)
	}
	if err != nil {
		return "", "", fmt.Errorf("resolve tasks file: %w", err)
	}
	info, err := os.Stat(tasks)
	if err != nil {
		return "", "", err
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("expected a regular tasks file: %s", tasks)
	}
	cmd := exec.Command("git", "-C", filepath.Dir(tasks), "rev-parse", "--show-toplevel")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("tasks file must be inside a Git working tree: %s", strings.TrimSpace(string(output)))
	}
	root, err := filepath.EvalSymlinks(strings.TrimSpace(string(output)))
	if err != nil {
		return "", "", err
	}
	return tasks, root, nil
}
