package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/arjenschwarz/orbit/internal/rune"
	"github.com/stretchr/testify/require"
)

func localRune(t *testing.T, root string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("rune", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "rune %v: %s", args, output)
	return output
}

func localFixture(t *testing.T) (string, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake Codex launcher uses a Unix shell")
	}
	if _, err := exec.LookPath("rune"); err != nil {
		t.Skip("requires rune CLI")
	}
	root := filepath.Join(t.TempDir(), "project with ' spaces")
	require.NoError(t, os.MkdirAll(root, 0755))
	var err error
	root, err = filepath.EvalSymlinks(root)
	require.NoError(t, err)
	output, err := exec.Command("git", "init", "-q", root).CombinedOutput()
	require.NoError(t, err, "%s", output)
	tasks := filepath.Join(root, "tasks.md")
	localRune(t, root, "create", "tasks.md", "--title", "Test tasks", "--reference", "design.md")
	localRune(t, root, "add-phase", tasks, "Implementation")
	for _, title := range []string{"First task", "Second task"} {
		localRune(t, root, "add", tasks, "--title", title, "--phase", "Implementation")
	}
	localRune(t, root, "update", tasks, "2", "--blocked-by", "1")
	bin := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bin, "codex"), []byte("#!/bin/sh\nORBIT_LOCAL_TEST_WORKER=1 exec \"$ORBIT_LOCAL_TEST_BINARY\" -test.run=^TestLocalWorkerProcess$ -- \"$@\"\n"), 0755))
	executable, err := os.Executable()
	require.NoError(t, err)
	t.Setenv("ORBIT_LOCAL_TEST_BINARY", executable)
	t.Setenv("ORBIT_LOCAL_TEST_MODE", "success")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HOME", t.TempDir())
	return root, tasks
}

func localCalls(t *testing.T, root string) [][]string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "calls.jsonl"))
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	var calls [][]string
	for line := range strings.SplitSeq(strings.TrimSpace(string(data)), "\n") {
		var args []string
		require.NoError(t, json.Unmarshal([]byte(line), &args))
		calls = append(calls, args)
	}
	return calls
}

func TestLocalCommandDispatch(t *testing.T) {
	for _, phase := range []bool{false, true} {
		t.Run(fmt.Sprintf("phase=%v", phase), func(t *testing.T) {
			root, tasks := localFixture(t)
			args := []string{tasks}
			if phase {
				args = append(args, "--phase")
			}
			require.NoError(t, localCommand(args))
			calls := localCalls(t, root)
			want := 2
			if phase {
				want = 1
			}
			require.Len(t, calls, want)
			for _, args := range calls {
				for _, flag := range []string{"--oss", "--local-provider", "lmstudio", "--ignore-user-config", "--ephemeral", "--json", "workspace-write"} {
					require.Contains(t, args, flag)
				}
				// Codex's OSS defaults still advertise tools LM Studio drops.
				// These must be explicit config overrides on every iteration.
				for _, setting := range []string{`web_search="disabled"`, "features.multi_agent=false", "features.apps=false", `model_reasoning_summary="none"`,
					"model_context_window=32768", "model_auto_compact_token_limit=20480", "tool_output_token_limit=2048", "sandbox_workspace_write.network_access=false"} {
					index := slices.Index(args, setting)
					require.Positive(t, index, "missing local compatibility setting %s", setting)
					require.Equal(t, "-c", args[index-1], "setting must be a Codex config override")
				}
				require.NotContains(t, args, "--dangerously-bypass-approvals-and-sandbox")
				prompt := args[len(args)-1]
				require.Contains(t, prompt, "Do not retrieve or select work again")
				require.Contains(t, prompt, "design.md")
				require.Contains(t, prompt, "Tool network access is disabled")
				require.Contains(t, prompt, "enable pipefail")
				require.Contains(t, prompt, "'\"'\"'")
			}
			if !phase {
				require.NotContains(t, calls[0][len(calls[0])-1], "Second task")
				require.Contains(t, calls[1][len(calls[1])-1], "Second task")
			}
			pending := localRune(t, root, "list", tasks, "--filter", "pending", "--format", "json")
			var ledger rune.ListResult
			require.NoError(t, json.Unmarshal(pending, &ledger))
			require.Empty(t, ledger.Tasks)
			prompts, err := filepath.Glob(filepath.Join(root, ".orbit", "local", "*.prompt.txt"))
			require.NoError(t, err)
			require.Len(t, prompts, want)
			logs, err := filepath.Glob(filepath.Join(root, ".orbit", "local", "phase-*-session.json"))
			require.NoError(t, err)
			require.Len(t, logs, want)
			for _, path := range logs {
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				require.Contains(t, string(data), `"type":"turn.completed"`)
			}
		})
	}
}

func TestLocalCommandContextAndNetwork(t *testing.T) {
	root, tasks := localFixture(t)
	require.NoError(t, localCommand([]string{tasks, "--context-window", "65536", "--compact-at", "40000", "--network"}))
	for _, args := range localCalls(t, root) {
		for _, setting := range []string{"model_context_window=65536", "model_auto_compact_token_limit=40000", "sandbox_workspace_write.network_access=true"} {
			require.Contains(t, args, setting)
		}
		require.Contains(t, args[len(args)-1], "Tool network access is enabled")
	}
}

func TestLocalCommandInstructionsFile(t *testing.T) {
	root, tasks := localFixture(t)
	instructions := filepath.Join(root, `worker ' " instructions.md`)
	require.NoError(t, os.WriteFile(instructions, []byte("Implement only the selected work and run its checks.\n"), 0600))
	t.Chdir(root)
	require.NoError(t, localCommand([]string{tasks, "--instructions-file", filepath.Base(instructions)}))
	calls := localCalls(t, root)
	require.Len(t, calls, 2)
	for _, args := range calls {
		found := false
		for index, arg := range args {
			if value, ok := strings.CutPrefix(arg, "model_instructions_file="); ok {
				var path string
				require.NoError(t, json.Unmarshal([]byte(value), &path))
				require.Equal(t, instructions, path)
				require.Positive(t, index)
				require.Equal(t, "-c", args[index-1])
				found = true
			}
		}
		require.True(t, found, "worker must receive the instruction file")
		require.Contains(t, args, "--oss")
		require.Contains(t, args, "workspace-write")
		require.Contains(t, args[len(args)-1], "Do not retrieve or select work again")
	}
}

func TestLocalCommandInstructionsPreflight(t *testing.T) {
	for _, kind := range []string{"missing", "directory", "empty", "invalid-utf8"} {
		t.Run(kind, func(t *testing.T) {
			root, tasks := localFixture(t)
			path := filepath.Join(root, "worker.md")
			switch kind {
			case "directory":
				require.NoError(t, os.Mkdir(path, 0755))
			case "empty":
				require.NoError(t, os.WriteFile(path, []byte(" \n\t"), 0600))
			case "invalid-utf8":
				require.NoError(t, os.WriteFile(path, []byte{0xff, 0xfe}, 0600))
			}
			require.ErrorContains(t, localCommand([]string{tasks, "--instructions-file", path}), "instructions file")
			require.Empty(t, localCalls(t, root))
			_, err := os.Stat(filepath.Join(root, ".orbit", "local"))
			require.True(t, os.IsNotExist(err), "invalid instructions created logs")
		})
	}
}

func TestLocalCommandStartupPreflight(t *testing.T) {
	for _, mode := range []string{"startup-error", "startup-timeout"} {
		t.Run(mode, func(t *testing.T) {
			root, tasks := localFixture(t)
			t.Setenv("ORBIT_LOCAL_TEST_MODE", mode)
			started := time.Now()
			err := localCommand([]string{tasks})
			require.ErrorContains(t, err, "codex CLI startup check failed")
			if mode == "startup-timeout" {
				require.ErrorContains(t, err, "deadline exceeded")
				require.Less(t, time.Since(started), 7*time.Second)
			}
			require.Empty(t, localCalls(t, root), "startup failure dispatched inference")
			_, err = os.Stat(filepath.Join(root, ".orbit", "local"))
			require.True(t, os.IsNotExist(err), "startup failure created logs")
		})
	}
}

func TestLocalCommandStops(t *testing.T) {
	for mode, expected := range map[string]string{
		"no-progress":     "no selected tasks completed",
		"event-error":     "worker failed",
		"halt-output":     "ORBIT-HALT: review needed",
		"halt-file":       "review marker",
		"graph-change":    "task graph changed",
		"reopen":          "was reopened",
		"unselected":      "outside the selection",
		"incomplete-turn": "without a completed turn",
		"process-error":   "worker process failed",
		"rename":          "identity or graph changed",
	} {
		t.Run(mode, func(t *testing.T) {
			root, tasks := localFixture(t)
			t.Setenv("ORBIT_LOCAL_TEST_MODE", mode)
			if mode == "reopen" {
				localRune(t, root, "complete", tasks, "2")
			}
			err := localCommand([]string{tasks})
			require.ErrorContains(t, err, expected)
			require.Len(t, localCalls(t, root), 1)
		})
	}
}

func TestLocalCommandPreflight(t *testing.T) {
	for _, mode := range []string{"halt", "stop-child", "blocked", "dry-run", "bad-model", "bad-selection", "bad-limit", "bad-timeout", "bad-context", "bad-compact", "compact-at-limit", "directory", "missing", "not-git"} {
		t.Run(mode, func(t *testing.T) {
			root, tasks := localFixture(t)
			args := []string{tasks, "--dry-run"}
			var expected string
			switch mode {
			case "halt":
				require.NoError(t, os.MkdirAll(filepath.Join(root, ".orbit"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(root, ".orbit", "halt"), []byte("existing stop"), 0600))
				expected = "existing stop"
			case "stop-child":
				localRune(t, root, "add", tasks, "--title", "[STOP] Review", "--parent", "1")
				expected = "requires human review"
			case "blocked":
				localRune(t, root, "add", tasks, "--title", "Blocked child", "--parent", "1", "--blocked-by", "2")
				expected = "outside the selection"
			case "bad-model":
				args = append(args, "--model", " ")
				expected = "model must be nonempty"
			case "bad-selection":
				args = append(args, "--selection", "invalid")
				expected = "selection must be"
			case "bad-limit":
				args = append(args, "--max-iterations=-1")
				expected = "max-iterations must be nonnegative"
			case "bad-timeout":
				args = append(args, "--timeout=0s")
				expected = "must be positive"
			case "bad-context":
				args = append(args, "--context-window=0")
				expected = "context window"
			case "bad-compact":
				args = append(args, "--compact-at=-1")
				expected = "compaction threshold"
			case "compact-at-limit":
				args = append(args, "--context-window=32768", "--compact-at=32768")
				expected = "compaction threshold"
			case "directory":
				args[0] = root
				expected = "regular tasks file"
			case "missing":
				args[0] = filepath.Join(root, "missing.md")
				expected = "resolve tasks file"
			case "not-git":
				args[0] = filepath.Join(t.TempDir(), "tasks.md")
				require.NoError(t, os.WriteFile(args[0], []byte("# Tasks\n"), 0600))
				expected = "Git working tree"
			}
			err := localCommand(args)
			if expected == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, expected)
			}
			require.Empty(t, localCalls(t, root))
			_, err = os.Stat(filepath.Join(root, ".orbit", "local"))
			require.True(t, os.IsNotExist(err), "dry-run created logs")
		})
	}
}

func TestLocalCommandIterationLimitAndResume(t *testing.T) {
	root, tasks := localFixture(t)
	require.ErrorContains(t, localCommand([]string{tasks, "--max-iterations", "1"}), "iteration limit")
	require.Len(t, localCalls(t, root), 1)
	require.NoError(t, localCommand([]string{tasks, "--max-iterations", "1"}))
	require.Len(t, localCalls(t, root), 2)
}

func TestLocalCommandAutomaticIterationBudget(t *testing.T) {
	t.Run("default completes more than twenty tasks", func(t *testing.T) {
		root, tasks := localFixture(t)
		for i := 3; i <= 21; i++ {
			localRune(t, root, "add", tasks, "--title", fmt.Sprintf("Task %d", i), "--phase", "Implementation")
		}

		require.NoError(t, localCommand([]string{tasks}))
		require.Len(t, localCalls(t, root), 21)
		var ledger rune.ListResult
		require.NoError(t, json.Unmarshal(localRune(t, root, "list", tasks, "--format", "json"), &ledger))
		for _, task := range ledger.Tasks {
			require.Equal(t, rune.StatusCompleted, task.Status)
		}
	})
	t.Run("explicit zero derives a finite budget", func(t *testing.T) {
		root, tasks := localFixture(t)
		require.NoError(t, localCommand([]string{tasks, "--max-iterations", "0"}))
		require.Len(t, localCalls(t, root), 2)
	})
}

func TestLocalCommandResumesInProgressTask(t *testing.T) {
	root, tasks := localFixture(t)
	localRune(t, root, "complete", tasks, "1")
	localRune(t, root, "progress", tasks, "2")
	require.NoError(t, localCommand([]string{tasks}))
	require.Len(t, localCalls(t, root), 1, "unfinished work should be dispatched")
	var ledger rune.ListResult
	require.NoError(t, json.Unmarshal(localRune(t, root, "list", tasks, "--format", "json"), &ledger))
	for _, task := range ledger.Tasks {
		require.Equal(t, rune.StatusCompleted, task.Status)
	}
}

func TestLocalCommandChildrenAndAncestorCompletion(t *testing.T) {
	root, tasks := localFixture(t)
	localRune(t, root, "add", tasks, "--title", "First child", "--parent", "1")
	localRune(t, root, "add", tasks, "--title", "Second child", "--parent", "1", "--blocked-by", "1.1")
	require.NoError(t, localCommand([]string{tasks}))
	calls := localCalls(t, root)
	require.Len(t, calls, 2)
	prompt := calls[0][len(calls[0])-1]
	require.Contains(t, prompt, "First child")
	require.Contains(t, prompt, "Second child")
	require.NotContains(t, prompt, "Second task")
}

func TestLocalCommandLeafDispatchAndResume(t *testing.T) {
	root, tasks := localFixture(t)
	localRune(t, root, "update", tasks, "1", "--details", "Shared parent context")
	localRune(t, root, "add", tasks, "--title", "Write failing tests", "--parent", "1")
	localRune(t, root, "add", tasks, "--title", "Implement behavior", "--parent", "1", "--blocked-by", "1.1")

	require.ErrorContains(t, localCommand([]string{tasks, "--leaf", "--max-iterations", "1"}), "iteration limit")
	var ledger rune.ListResult
	require.NoError(t, json.Unmarshal(localRune(t, root, "list", tasks, "--format", "json"), &ledger))
	require.NotEqual(t, rune.StatusCompleted, ledger.Tasks[0].Status)
	require.Equal(t, rune.StatusCompleted, ledger.Tasks[0].Children[0].Status)
	require.Equal(t, rune.StatusPending, ledger.Tasks[0].Children[1].Status)

	require.NoError(t, localCommand([]string{tasks, "--leaf", "--max-iterations", "2"}))
	calls := localCalls(t, root)
	require.Len(t, calls, 3)
	for i, id := range []string{"1.1", "1.2", "2"} {
		prompt := calls[i][len(calls[i])-1]
		start := strings.Index(prompt, "<selected-work>\n") + len("<selected-work>\n")
		end := strings.Index(prompt, "\n</selected-work>")
		var work rune.NextPhaseResult
		require.NoError(t, json.Unmarshal([]byte(prompt[start:end]), &work))
		require.Len(t, work.Tasks, 1)
		require.Equal(t, id, work.Tasks[0].ID)
		require.Empty(t, work.Tasks[0].Children)
		if i < 2 {
			require.Contains(t, prompt, "Ancestor context (read only)")
			require.Contains(t, prompt, "Shared parent context")
			require.NotContains(t, prompt, "Second task")
		}
	}
	require.NotContains(t, calls[0][len(calls[0])-1], "Implement behavior")
	require.NoError(t, json.Unmarshal(localRune(t, root, "list", tasks, "--format", "json"), &ledger))
	for _, task := range ledger.Tasks {
		require.Equal(t, rune.StatusCompleted, task.Status)
	}
}

func TestLocalCommandLeafGuards(t *testing.T) {
	for _, mode := range []string{"phase", "model", "stop", "ancestor-blocked", "no-ready-leaf", "outside-selection"} {
		t.Run(mode, func(t *testing.T) {
			root, tasks := localFixture(t)
			localRune(t, root, "add", tasks, "--title", "First child", "--parent", "1")
			args := []string{tasks, "--leaf"}
			expected := ""
			switch mode {
			case "phase":
				args = append(args, "--phase")
				expected = "--leaf requires runner selection without --phase"
			case "model":
				args = append(args, "--selection", "model")
				expected = "--leaf requires runner selection without --phase"
			case "stop":
				localRune(t, root, "add", tasks, "--title", "[STOP] Review later child", "--parent", "1")
				expected = "requires human review"
			case "ancestor-blocked":
				localRune(t, root, "add", tasks, "--title", "External prerequisite")
				localRune(t, root, "update", tasks, "1", "--blocked-by", "3")
				expected = "no ready leaf"
			case "no-ready-leaf":
				localRune(t, root, "update", tasks, "1.1", "--blocked-by", "2")
				expected = "no ready leaf"
			case "outside-selection":
				t.Setenv("ORBIT_LOCAL_TEST_MODE", "unselected")
				expected = "outside the selection"
			}
			err := localCommand(args)
			require.ErrorContains(t, err, expected)
			if mode == "outside-selection" {
				require.Len(t, localCalls(t, root), 1)
			} else {
				require.Empty(t, localCalls(t, root))
			}
			if mode == "phase" || mode == "model" {
				_, err := os.Stat(filepath.Join(root, ".orbit", "local"))
				require.True(t, os.IsNotExist(err), "invalid flags created logs")
			}
		})
	}
}

func TestLocalCommandLeafSkipsBlockedSibling(t *testing.T) {
	root, tasks := localFixture(t)
	localRune(t, root, "add", tasks, "--title", "Blocked child", "--parent", "1")
	localRune(t, root, "add", tasks, "--title", "Ready child", "--parent", "1")
	localRune(t, root, "update", tasks, "1.1", "--blocked-by", "1.2")
	localRune(t, root, "progress", tasks, "1.2")

	require.ErrorContains(t, localCommand([]string{tasks, "--leaf", "--max-iterations", "1"}), "iteration limit")
	calls := localCalls(t, root)
	require.Len(t, calls, 1)
	prompt := calls[0][len(calls[0])-1]
	require.Contains(t, prompt, "Ready child")
	require.NotContains(t, prompt, "Blocked child")
	var ledger rune.ListResult
	require.NoError(t, json.Unmarshal(localRune(t, root, "list", tasks, "--format", "json"), &ledger))
	require.Equal(t, rune.StatusPending, ledger.Tasks[0].Children[0].Status)
	require.Equal(t, rune.StatusCompleted, ledger.Tasks[0].Children[1].Status)
}

func TestLocalCommandRelativePathFromAnotherDirectory(t *testing.T) {
	root, tasks := localFixture(t)
	t.Chdir(filepath.Dir(root))
	relative, err := filepath.Rel(filepath.Dir(root), tasks)
	require.NoError(t, err)
	require.NoError(t, localCommand([]string{relative}))
	require.Len(t, localCalls(t, root), 2)
}

func TestLocalCommandArbitraryTaskPaths(t *testing.T) {
	for _, mode := range []string{"absolute", "relative", "symlink"} {
		t.Run(mode, func(t *testing.T) {
			root, original := localFixture(t)
			tasks := filepath.Join(root, "nested ' plans", "custom ' work-list.txt")
			require.NoError(t, os.MkdirAll(filepath.Dir(tasks), 0755))
			require.NoError(t, os.Rename(original, tasks))
			t.Chdir(t.TempDir())
			outside, err := os.Getwd()
			require.NoError(t, err)
			input := tasks
			switch mode {
			case "relative":
				input, err = filepath.Rel(outside, tasks)
				require.NoError(t, err)
			case "symlink":
				input = filepath.Join(outside, "linked ledger.md")
				require.NoError(t, os.Symlink(tasks, input))
			}

			require.NoError(t, localCommand([]string{input}))
			calls := localCalls(t, root)
			require.Len(t, calls, 2)
			for _, call := range calls {
				prompt := call[len(call)-1]
				require.Contains(t, prompt, "Explicit tasks file: "+tasks)
				require.Contains(t, prompt, "Repository root: "+root)
			}
			var ledger rune.ListResult
			require.NoError(t, json.Unmarshal(localRune(t, root, "list", tasks, "--format", "json"), &ledger))
			for _, task := range ledger.Tasks {
				require.Equal(t, rune.StatusCompleted, task.Status)
			}
			logs, err := filepath.Glob(filepath.Join(filepath.Dir(tasks), ".orbit", "local", "*.prompt.txt"))
			require.NoError(t, err)
			require.Len(t, logs, 2)
		})
	}
}

func TestLocalCommandTimeout(t *testing.T) {
	root, tasks := localFixture(t)
	t.Setenv("ORBIT_LOCAL_TEST_MODE", "timeout")
	start := time.Now()
	require.ErrorContains(t, localCommand([]string{tasks, "--timeout", "200ms"}), "deadline exceeded")
	require.Less(t, time.Since(start), 5*time.Second)
	require.Len(t, localCalls(t, root), 1)
	logs, err := filepath.Glob(filepath.Join(root, ".orbit", "local", "*.stdout.jsonl"))
	require.NoError(t, err)
	require.Len(t, logs, 1)
	data, err := os.ReadFile(logs[0])
	require.NoError(t, err)
	require.Contains(t, string(data), "working")
}

func TestLocalCommandModelSelection(t *testing.T) {
	root, tasks := localFixture(t)
	skill := filepath.Join(os.Getenv("HOME"), ".agents", "skills", "next-task", "SKILL.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(skill), 0755))
	require.NoError(t, os.WriteFile(skill, []byte("Read the explicit ledger and implement the selected tasks."), 0600))
	require.NoError(t, localCommand([]string{tasks, "--selection", "model"}))
	calls := localCalls(t, root)
	require.Len(t, calls, 1)
	prompt := calls[0][len(calls[0])-1]
	require.Contains(t, prompt, skill)
	require.Contains(t, prompt, "--phase --format json")
	require.Contains(t, prompt, "Second task")
}

// TestLocalWorkerProcess runs as a separate process behind the fake CLI. The
// production command still resolves a real Git tree and uses the real rune CLI.
func TestLocalWorkerProcess(t *testing.T) {
	if os.Getenv("ORBIT_LOCAL_TEST_WORKER") != "1" {
		return
	}
	index := slices.Index(os.Args, "--")
	args := os.Args[index+1:]
	mode := os.Getenv("ORBIT_LOCAL_TEST_MODE")
	if len(args) == 1 && args[0] == "--version" {
		switch mode {
		case "startup-timeout":
			time.Sleep(8 * time.Second)
		case "startup-error":
			os.Exit(7)
		}
		fmt.Println("codex-cli test")
		os.Exit(0)
	}
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	f, err := os.OpenFile("calls.jsonl", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	require.NoError(t, err)
	_, err = f.Write(append(encoded, '\n'))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	if mode == "timeout" {
		fmt.Println(`{"type":"working"}`)
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	if mode == "event-error" {
		fmt.Println(`{"type":"turn.failed","error":{"message":"worker failed"}}`)
		os.Exit(0)
	}
	if mode == "process-error" {
		fmt.Fprintln(os.Stderr, "worker process failed")
		os.Exit(2)
	}
	if mode == "incomplete-turn" {
		fmt.Println(`{"type":"item.started"}`)
		os.Exit(0)
	}
	prompt := args[len(args)-1]
	_, taskLine, ok := strings.Cut(prompt, "Explicit tasks file: ")
	require.True(t, ok, "worker must receive the explicit task file")
	ledgerPath, _, _ := strings.Cut(taskLine, "\n")
	start := strings.Index(prompt, "<selected-work>\n") + len("<selected-work>\n")
	end := strings.Index(prompt, "\n</selected-work>")
	var work rune.NextPhaseResult
	require.NoError(t, json.Unmarshal([]byte(prompt[start:end]), &work))
	run := func(args ...string) {
		output, err := exec.Command("rune", args...).CombinedOutput()
		require.NoError(t, err, "%s", output)
	}
	if mode != "no-progress" {
		var complete func([]rune.PhaseTask)
		complete = func(tasks []rune.PhaseTask) {
			for _, task := range tasks {
				complete(task.Children)
				if !strings.EqualFold(task.Status, "completed") {
					run("complete", ledgerPath, task.ID)
				}
			}
		}
		complete(work.Tasks)
	}
	switch mode {
	case "halt-file":
		require.NoError(t, os.WriteFile(filepath.Join(".orbit", "halt"), []byte("review marker"), 0600))
	case "graph-change":
		run("add", ledgerPath, "--title", "Unexpected task")
	case "reopen":
		run("uncomplete", ledgerPath, "2")
	case "unselected":
		run("complete", ledgerPath, "2")
	case "rename":
		run("update", ledgerPath, "1", "--title", "Renamed task")
	}
	message := "Done"
	if mode == "halt-output" {
		message = "ORBIT-HALT: review needed"
	}
	event := map[string]any{"type": "item.completed", "item": map[string]string{"type": "agent_message", "text": message}}
	data, err := json.Marshal(event)
	require.NoError(t, err)
	fmt.Println(string(data))
	fmt.Println(`{"type":"turn.completed"}`)
	os.Exit(0)
}
