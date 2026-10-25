# Design: Local task execution

`orbit local <tasks.md>` is a preset for `internal/orbit.Orbit`, using the existing
Codex agent. It replaces the standalone Python loop in agentic-coding. The
compatibility name `local-next-task` routes to the same command by executable
name. Hosted `orbit run` keeps its current defaults.

`cmd/orbit/local.go` resolves symlinks, validates a regular task file, discovers
its Git root, and builds an explicit configuration. It does not load home or
project Orbit aliases, prompts, or Codex user configuration. Model and provider
are explicit; Codex supplies the tool loop. Model mode uses the installed
`~/.agents/skills/next-task/SKILL.md`, checked before execution.

`internal/rune` exposes task selection alongside phase selection, retaining
children, dependencies and references. `internal/orbit/local.go` builds the
bounded prompt, snapshots the full graph and validates selection/progress.
`runSingle` uses these helpers only when a local preset is present, then calls
the existing `runPhaseWithRetry`/`runPhase` and log manager. Local attempts run
once: automatic retries after partial edits would hide a failed iteration.
Per-iteration prompts are separate from the configured hosted command.

`internal/agents/codex` parses JSONL failed-turn events and final assistant text.
It keeps raw stdout for logs. `RunOptions`/the shared executor accept optional
live output writers and process-group cancellation; existing agents retain
their behavior unless those options are supplied. Separate stdout/stderr files
avoid concurrent writes to one writer. Local execution stores these and the
prompt beside Orbit's standard per-phase artifacts, with unique attempt names.

Task identity includes title, parent, dependencies, details and references;
status is the only field workers should change. Automatic ancestor completion
is allowed. Completing unrelated work, deleting/renumbering tasks or reopening
completed work stops the run. Existing halt markers remain for human review.

Portability assumptions: LM Studio and a recent Codex CLI support `--oss`,
`--local-provider lmstudio`, `--ignore-user-config`, `--ephemeral` and `--json`.
Unix process groups are used on macOS/Linux; Windows cancellation uses Go's
direct-process termination. The model ID is a runtime identifier, not a measure
of capability. Tests use real rune in temporary Git repositories with a fake
Codex; the live run uses the installed `8-bit` model and a disposable ledger.

Risks: local inference can take minutes per tool turn; the timeout is adjustable.
Sandbox enforcement depends on Codex/platform support. Project instructions can
still be read by Codex. Neither task completion nor passing a smoke test proves
correctness of arbitrary generated code. Record observed limitations in the
verification report.
