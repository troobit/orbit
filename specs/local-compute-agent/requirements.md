# Requirements: Local Compute Agent

Orbit replaces the `local-next-task` prototype in `../agentic-coding` with a
local entry point backed by the existing Codex agent and orchestration loop.

1. `orbit local <tasks.md>` MUST resolve the explicit ledger and its Git root
   from any working directory, including paths containing spaces. It MUST reject
   missing files, directories, and ledgers outside Git before creating logs.
2. Local execution MUST use Codex's LM Studio provider, an explicit model
   (default `8-bit`), fresh ephemeral sessions, isolated user configuration,
   workspace-write sandbox, and unattended approvals. It MUST leave edits for
   review without asking the worker to commit or push. Hosted runs MUST retain
   their existing behavior.
3. Orbit MUST select one task per iteration by default and pass the task,
   descendants, details, references, and explicit rune commands to the worker.
   `--phase` MUST select a whole phase. `--selection model` MAY delegate retrieval
   to the installed next-task skill, bounded by Orbit's preflight selection.
4. Orbit MUST stop before dispatch on a repository `.orbit/halt`, a selected
   STOP gate (including descendants), an external unmet dependency, or an empty
   selection while work remains. Dependencies internal to a phase MAY be
   satisfied sequentially by the worker.
5. After each iteration Orbit MUST stop on a worker halt signal, any changed task
   identity/graph, reopened completed task, progress outside the selection
   (except automatic ancestor completion), or no selected task completed.
   A claimed success in prose alone MUST NOT advance execution.
6. Local execution MUST have positive configurable per-session timeout (default
   30 minutes) and iteration limit (default 20). Timeout, interruption, failed
   Codex events, and process failures MUST stop without automatically rerunning
   partially applied work. Unix cancellation MUST terminate the process group.
7. Local logs MUST use Orbit's log manager, retain the prompt and live stdout
   and stderr per attempt, and survive interruption. Dry runs MUST print the
   selected prompt without invoking inference or creating logs.
8. `local-next-task <tasks.md>` MUST be available as an installed compatibility
   entry point to the same Orbit implementation. Replacing the old prototype
   command MUST require no nested runner or copied agent loop.
9. Verification MUST cover task/phase dispatch, halt/progress checks, failures,
   iteration bounds, paths, and real inference completing at least two tasks in
   a disposable repository. Model, timings, commands and limitations MUST be
   recorded in this spec.

Model downloads, server lifecycle management, variants, comparison changes, and
new transcript formats are outside this change. The local runtime is managed
separately; an installed model is used for live verification.
