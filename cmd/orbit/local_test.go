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
				require.NotContains(t, args, "--dangerously-bypass-approvals-and-sandbox")
				prompt := args[len(args)-1]
				require.Contains(t, prompt, "Do not retrieve or select work again")
				require.Contains(t, prompt, "design.md")
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
	for _, mode := range []string{"halt", "stop-child", "blocked", "dry-run", "bad-model", "bad-selection", "bad-limit", "bad-timeout", "directory", "missing", "not-git"} {
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
				args = append(args, "--max-iterations=0")
				expected = "must be positive"
			case "bad-timeout":
				args = append(args, "--timeout=0s")
				expected = "must be positive"
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

func TestLocalCommandRelativePathFromAnotherDirectory(t *testing.T) {
	root, tasks := localFixture(t)
	t.Chdir(filepath.Dir(root))
	relative, err := filepath.Rel(filepath.Dir(root), tasks)
	require.NoError(t, err)
	require.NoError(t, localCommand([]string{relative}))
	require.Len(t, localCalls(t, root), 2)
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
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	f, err := os.OpenFile("calls.jsonl", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	require.NoError(t, err)
	_, err = f.Write(append(encoded, '\n'))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	mode := os.Getenv("ORBIT_LOCAL_TEST_MODE")
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
					run("complete", "tasks.md", task.ID)
				}
			}
		}
		complete(work.Tasks)
	}
	switch mode {
	case "halt-file":
		require.NoError(t, os.WriteFile(filepath.Join(".orbit", "halt"), []byte("review marker"), 0600))
	case "graph-change":
		run("add", "tasks.md", "--title", "Unexpected task")
	case "reopen":
		run("uncomplete", "tasks.md", "2")
	case "unselected":
		run("complete", "tasks.md", "2")
	case "rename":
		run("update", "tasks.md", "1", "--title", "Renamed task")
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
