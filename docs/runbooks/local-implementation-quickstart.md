# Implement a task file locally with Orbit

Use `orbit local` with LM Studio's `8-bit` model. Orbit selects the next task,
Codex runs the model's editing and testing tools, and Orbit checks the task
ledger before starting another fresh session.

**1. Install the command once, or rebuild it after updating Orbit.**

From the Orbit checkout, with Go, rune and Codex installed:

```bash
make install-local-next-task
```

Ensure `~/.local/bin` is on PATH. The installed `local-next-task` command is an
equivalent entry point: replace `orbit local` with `local-next-task` in the
commands below. No `.orbit.yaml` is needed for this workflow.

Reinstall after updating Orbit to pick up its local compatibility settings.
The preset disables unsupported web search, delegation and connected-app tools. LM Studio
may still warn about cache keys, developer-role conversion and encrypted
reasoning metadata; see the [warning guide](local-next-task.md#lm-studio-compatibility-warnings)
for what each means.

**2. Check LM Studio.**

```bash
lms ps
lms server status
```

If necessary, load the installed model and start the API server:

```bash
lms load 8-bit --identifier 8-bit --context-length 32768 --yes
lms server start --port 1234 --bind 127.0.0.1
```

**3. Choose and preview your task file.**

The file must be a rune-compatible task list inside a Git working tree. Its
references should point to the requirements, design and other context the
worker needs. Orbit finds the target repository from this file, so you can
invoke it from any directory; the filename need not be `tasks.md`.

```bash
tasks_file="/absolute/path/to/project/specs/my-feature/tasks.md"
rune list "$tasks_file"
orbit local "$tasks_file" --model 8-bit --dry-run
```

Check the preview's repository root, selected task and references.

**4. Run the implementation.**

```bash
orbit local "$tasks_file" --model 8-bit \
  --context-window 32768 --selection runner --timeout 30m --max-iterations 20
```

Orbit passes one selected task and its subtasks to each worker session. The
worker implements that selection, runs the project's validation, updates the
ledger with rune and leaves changes for review. Orbit continues while work
remains and each iteration completes at least one selected task.

Match `--context-window` to the context actually loaded in LM Studio. The
default starts history compaction at 20480 tokens and limits saved tool-output
history. A `TruncateMiddle` server message means context is being removed;
see [context and dependency blockers](local-next-task.md#context-and-dependency-blockers).

Prepare dependencies before running, or add `--network` to allow sandboxed
tools to download them. Package caches must still be writable in the sandbox.
Use small, separately verifiable top-level tasks: a selected parent and all its
children run together, so adding subtasks alone does not reduce session scope.

For a larger selection, add `--phase` to execute a whole phase per session.
Increase `--timeout` to `1h` if needed. To inspect the first iteration before
continuing, use `--max-iterations 1`; reaching that limit returns an error if
more work remains. The verified two-task `8-bit` run took about 14 minutes.

**5. Review or resume.**

Orbit prints a live stdout path and elapsed time every 30 seconds. Logs,
prompts and the run summary live in `.orbit/local/` beside the task file.
Inspect the Git diff, run the project's checks and review the updated ledger.

A failure, timeout, STOP gate, halt signal or lack of completed work stops the
run. Review partial changes before rerunning the same command; it selects the
remaining work from the ledger. If the repository has `.orbit/halt`, resolve
its reason and remove that marker before resuming. Ctrl+C interrupts a run.

See the [full runbook](local-next-task.md) for diagnostics and other options.
