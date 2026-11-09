package orbit

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/arjenschwarz/orbit/internal/agents"
	"github.com/arjenschwarz/orbit/internal/rune"
)

// LocalConfig opts into bounded task dispatch for a local worker.
type LocalConfig struct {
	Selection     string
	Phase         bool
	MaxIterations int
	SkillPath     string
	Network       bool
}

type localIteration struct {
	before   map[string]rune.Task
	selected map[string]bool
	prompt   string
}

var stopTask = regexp.MustCompile(`(?i)^\s*(?:\[STOP\]|STOP\b)`)

func (o *Orbit) checkLocalHalt() error {
	path := filepath.Join(o.config.WorkingDir, ".orbit", "halt")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read halt marker: %w", err)
	}
	return fmt.Errorf("ORBIT-HALT: %s: %s", path, strings.TrimSpace(string(data)))
}

func taskSnapshot(tasks []rune.Task) (map[string]rune.Task, error) {
	result := make(map[string]rune.Task)
	var visit func([]rune.Task, string) error
	visit = func(tasks []rune.Task, parent string) error {
		for _, task := range tasks {
			if _, exists := result[task.ID]; task.ID == "" || exists || task.Status < rune.StatusPending || task.Status > rune.StatusCompleted {
				return fmt.Errorf("invalid or duplicate task ID/status: %s", task.ID)
			}
			task.ParentID = parent
			result[task.ID] = task
			if err := visit(task.Children, task.ID); err != nil {
				return err
			}
		}
		return nil
	}
	err := visit(tasks, "")
	return result, err
}

// Rune's pending filter excludes in-progress tasks, which still need a worker
// after interruption. Inspect all statuses before declaring a local run done.
func (o *Orbit) localIncompleteTasks() ([]rune.Task, error) {
	tasks, err := o.runeClient.ListAll()
	if err != nil {
		return nil, err
	}
	snapshot, err := taskSnapshot(tasks)
	if err != nil {
		return nil, err
	}
	var incomplete []rune.Task
	for _, task := range snapshot {
		if task.Status != rune.StatusCompleted {
			incomplete = append(incomplete, task)
		}
	}
	return incomplete, nil
}

func selectedTask(task rune.Task) rune.PhaseTask {
	result := rune.PhaseTask{ID: task.ID, Title: task.Title, Details: task.Details,
		References: task.References, Reqs: task.Reqs, BlockedBy: task.BlockedBy, Status: "Pending"}
	switch task.Status {
	case rune.StatusCompleted:
		result.Status = "Completed"
	case rune.StatusInProgress:
		result.Status = "In progress"
	}
	for _, child := range task.Children {
		result.Children = append(result.Children, selectedTask(child))
	}
	return result
}

func (o *Orbit) prepareLocalIteration() (*rune.NextPhaseResult, error) {
	tasks, err := o.runeClient.ListAll()
	if err != nil {
		return nil, err
	}
	before, err := taskSnapshot(tasks)
	if err != nil {
		return nil, err
	}
	work, err := o.runeClient.Select(o.config.Local.Phase)
	if err != nil {
		return nil, err
	}
	if o.config.Local.Selection == "model" && !o.config.Local.Phase && len(work.Tasks) == 1 && !strings.Contains(work.Tasks[0].ID, ".") {
		work, err = o.runeClient.Select(true)
		if err != nil {
			return nil, err
		}
	}
	iteration := &localIteration{before: before, selected: make(map[string]bool)}
	var collect func(rune.Task)
	collect = func(task rune.Task) {
		iteration.selected[task.ID] = true
		for _, child := range task.Children {
			collect(child)
		}
	}
	for i, selected := range work.Tasks {
		task, exists := before[selected.ID]
		if !exists {
			return nil, fmt.Errorf("selected task %s is absent from the ledger", selected.ID)
		}
		collect(task)
		work.Tasks[i] = selectedTask(task)
		if work.PhaseName == "" {
			for current := task; current.ID != ""; current = before[current.ParentID] {
				if current.Phase != "" {
					work.PhaseName = current.Phase
					break
				}
			}
		}
	}
	if len(iteration.selected) == 0 {
		return nil, fmt.Errorf("incomplete tasks have no ready work")
	}
	ready := false
	for id := range iteration.selected {
		task := before[id]
		if task.Status == rune.StatusCompleted {
			continue
		}
		if stopTask.MatchString(task.Title) {
			return nil, fmt.Errorf("task %s requires human review: %s", id, task.Title)
		}
		blocked := false
		for _, dependency := range task.BlockedBy {
			dep, exists := before[dependency]
			if !exists || dep.Status != rune.StatusCompleted {
				blocked = true
				if !exists || !iteration.selected[dependency] {
					return nil, fmt.Errorf("task %s is blocked by %s outside the selection", id, dependency)
				}
			}
		}
		ready = ready || !blocked
	}
	if !ready {
		return nil, fmt.Errorf("selected tasks have no ready work")
	}
	iteration.prompt, err = o.localPrompt(work)
	if err != nil {
		return nil, err
	}
	o.localIteration = iteration
	return work, nil
}

func workerHalt(output string) (string, bool) {
	for line := range strings.SplitSeq(output, "\n") {
		if reason, ok := strings.CutPrefix(strings.TrimSpace(line), "ORBIT-HALT:"); ok {
			return strings.TrimSpace(reason), true
		}
	}
	return "", false
}

func (o *Orbit) localOutput(opts *agents.RunOptions, phase int, sessionID string) (func(), error) {
	base := filepath.Join(o.logManager.SessionDir(), fmt.Sprintf("phase-%d-%s", phase, sessionID))
	if err := os.WriteFile(base+".prompt.txt", []byte(opts.Prompt), 0600); err != nil {
		return nil, err
	}
	stdout, err := os.OpenFile(base+".stdout.jsonl", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	stderr, err := os.OpenFile(base+".stderr.txt", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		_ = stdout.Close()
		return nil, err
	}
	opts.Stdout, opts.Stderr = stdout, stderr
	log.Printf("Live output: %s.stdout.jsonl", base)
	done := make(chan struct{})
	start := time.Now()
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Printf("Local phase %d running: %s elapsed", phase, time.Since(start).Round(time.Second))
			case <-done:
				return
			}
		}
	}()
	return func() {
		close(done)
		_ = stdout.Close()
		_ = stderr.Close()
	}, nil
}

// shellQuote keeps generated rune commands safe for paths with shell syntax.
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func (o *Orbit) localPrompt(work *rune.NextPhaseResult) (string, error) {
	data, err := json.MarshalIndent(work, "", "  ")
	if err != nil {
		return "", err
	}
	tasks := shellQuote(o.config.TasksFile)
	var commands strings.Builder
	var visit func([]rune.PhaseTask)
	visit = func(selected []rune.PhaseTask) {
		for _, task := range selected {
			visit(task.Children)
			if !strings.EqualFold(task.Status, "completed") {
				fmt.Fprintf(&commands, "Start: rune progress %s %s\nAfter validation: rune complete %s %s\n", tasks, shellQuote(task.ID), tasks, shellQuote(task.ID))
			}
		}
	}
	visit(work.Tasks)
	retrieval := "Implement the work already selected below. Do not retrieve or select work again."
	if o.config.Local.Selection == "model" {
		phase := ""
		if o.config.Local.Phase || len(work.Tasks) > 1 || (len(work.Tasks) == 1 && !strings.Contains(work.Tasks[0].ID, ".")) {
			phase = " --phase"
		}
		retrieval = fmt.Sprintf("Read the next-task skill at %s. Retrieve the same selection using rune next %s%s --format json. If selected IDs differ, halt for review. Do not advance beyond this selection.", o.config.Local.SkillPath, tasks, phase)
	}
	network := "Tool network access is disabled. Do not retry online dependency downloads."
	if o.config.Local.Network {
		network = "Tool network access is enabled; filesystem writes remain sandboxed."
	}
	return fmt.Sprintf(`%s

Explicit tasks file: %s
Repository root: %s
Selected work (rune JSON):
<selected-work>
%s
</selected-work>

Rune commands (complete children before parents):
%s
Execute only this selection and its subtasks sequentially; do not delegate.
Read project instructions and the referenced sections and relevant agent notes
needed for this selection. Use bounded searches and excerpts; avoid dumping
whole documents or command output. Resolve references from the repository root
or relative to the task file. Use the absolute tasks path in all rune commands.
Validate using the project's Makefile when present. Mark work complete only
after validation. Keep task IDs, titles, dependencies and graph unchanged.
Run validation commands directly. If a pipeline is necessary, enable pipefail
and preserve the failing command's exit status; a trailing echo or tail is not
proof of success. %s If required dependencies or writable caches are unavailable,
halt with the blocker rather than cycling through downloads or copying caches.
Leave changes for review without committing or pushing. Update relevant notes.
This is an unattended iteration; do not ask questions. If blocked or approval
is required, write .orbit/halt in the repository root with a one-line reason
and finish with ORBIT-HALT: <reason>. Otherwise report completed task IDs,
changed files and validation, then stop. Orbit starts the next session.
`, retrieval, o.config.TasksFile, o.config.WorkingDir, data, commands.String(), network), nil
}

func (o *Orbit) validateLocalProgress() error {
	if err := o.checkLocalHalt(); err != nil {
		return err
	}
	tasks, err := o.runeClient.ListAll()
	if err != nil {
		return err
	}
	after, err := taskSnapshot(tasks)
	if err != nil {
		return err
	}
	before := o.localIteration.before
	if len(before) != len(after) {
		return fmt.Errorf("task graph changed; review the last iteration")
	}
	allowed := make(map[string]bool)
	for id := range o.localIteration.selected {
		for current := id; current != ""; current = before[current].ParentID {
			allowed[current] = true
		}
	}
	completed := []string{}
	for id, old := range before {
		now, exists := after[id]
		if !exists {
			return fmt.Errorf("task IDs changed; review the last iteration")
		}
		oldStatus, newStatus := old.Status, now.Status
		old.Status, now.Status = 0, 0
		old.Children, now.Children = nil, nil
		if !reflect.DeepEqual(old, now) {
			return fmt.Errorf("task %s identity or graph changed; review the last iteration", id)
		}
		if oldStatus == rune.StatusCompleted && newStatus != rune.StatusCompleted {
			return fmt.Errorf("completed task %s was reopened", id)
		}
		if oldStatus != newStatus && !allowed[id] {
			return fmt.Errorf("task %s advanced outside the selection", id)
		}
		if oldStatus != rune.StatusCompleted && newStatus == rune.StatusCompleted && o.localIteration.selected[id] {
			completed = append(completed, id)
		}
	}
	if len(completed) == 0 {
		return fmt.Errorf("no selected tasks completed; review the last iteration before resuming")
	}
	log.Printf("Completed tasks: %s", strings.Join(completed, ", "))
	return nil
}
