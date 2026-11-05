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
The preview also checks that Codex starts within five seconds; startup failures
return before creating logs or invoking the model.

**4. Run the implementation.**

```bash
orbit local "$tasks_file" --model 8-bit \
  --context-window 32768 --selection runner --timeout 30m
```

Orbit passes one selected task and its subtasks to each worker session. The
worker implements that selection, runs the project's validation, updates the
ledger with rune and leaves changes for review. Orbit continues while work
remains and each iteration completes at least one selected task.
The default session budget comes from the number of unfinished tasks in the
ledger, so files with more than twenty tasks can finish in one invocation.
Use `--max-iterations N` when you want an explicit smaller bound.

For a shorter Codex worker instruction envelope, use the provided
[worker instructions](local-worker-instructions.md):

```bash
orbit local "$tasks_file" --model 8-bit --leaf \
  --instructions-file /absolute/path/to/orbit/docs/runbooks/local-worker-instructions.md
```

The option replaces Codex's built-in base instructions; the selected task prompt
and sandbox settings remain. Paths resolve from the directory invoking Orbit.
Without this option, existing Codex instructions remain in use. This controls
instruction size, separately from the model's reasoning configuration.

For Qwen-derived models, inspect the actual chat template when simple tool turns
take minutes. A custom model entry may not expose a reasoning control even when
its template defaults to `xhigh`. Lowering a Codex setting does not prove that
the model received it. Retain the original template and validate the effective
runtime settings; the M5 verification record covers the measured configuration.

Set `--context-window` no larger than the context actually loaded in LM Studio.
A smaller worker budget can be appropriate for a model; test it on the same
ledger and retain failed attempts when changing the budget. The
default starts history compaction at 20480 tokens and limits saved tool-output
history. A `TruncateMiddle` server message means context is being removed;
see [context and dependency blockers](local-next-task.md#context-and-dependency-blockers).

For the loaded Bonsai model, use its exact identifier and the smaller worker
budget exercised by the local verification:

```bash
orbit local "$tasks_file" --model bonsai-local --context-window 16384 \
  --instructions-file /absolute/path/to/orbit/docs/runbooks/local-worker-instructions.md
```

Prepare dependencies before running, or add `--network` to allow sandboxed
tools to download them. Package caches must still be writable in the sandbox.
Use small, separately verifiable tasks. By default, a selected parent and all
its children run together. Add `--leaf` to give one ready subtask its own fresh
session, with the parent's details supplied as read-only context:

```bash
orbit local "$tasks_file" --model 8-bit --leaf --network --max-iterations 1
```

Review the result and rerun to dispatch the next ready subtask. Orbit preserves
the ledger and dependencies. For tests-first work, make the expected failing
test explicit in that task's acceptance criteria; implementation tasks still
require passing checks. Leaf mode requires runner selection without `--phase`.

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
Rune queries also have deadlines and stop on interruption; malformed rune
output is an error. A timed-out worker may have left useful code or task
updates, so run independent checks before accepting those changes.
If it already completed the whole ledger, rerunning can finish without another
worker; that rerun does not perform independent code validation for you.

See the [full runbook](local-next-task.md) for diagnostics and other options,
and the [reliability verification](../../specs/local-compute-agent/verification-reliability.md)
for the two-model execution and cancellation evidence.
