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
   `--leaf` MUST optionally dispatch just the first ready unfinished leaf within
   the selected task, retaining ancestor context without assigning sibling work.
   It MUST respect ancestor dependencies and STOP gates, preserve the ledger,
   and reject combination with phase or model-driven selection before logging.
4. Orbit MUST stop before dispatch on a repository `.orbit/halt`, a selected
   STOP gate (including descendants), an external unmet dependency, or an empty
   selection while work remains. Dependencies internal to a phase MAY be
   satisfied sequentially by the worker.
5. After each iteration Orbit MUST stop on a worker halt signal, any changed task
   identity/graph, reopened completed task, progress outside the selection
   (except automatic ancestor completion), or no selected task completed.
   A claimed success in prose alone MUST NOT advance execution.
6. Local execution MUST have a positive configurable per-session timeout (default
   30 minutes) and a finite iteration budget. By default (`--max-iterations 0`),
   derive the budget from the initial number of incomplete tasks, including
   descendants. A positive explicit limit MUST retain its requested bound;
   negative limits MUST fail preflight. Timeout, interruption, failed
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
10. The local preset MUST explicitly disable built-in web search, delegation
    tools, connected-app tools and reasoning summaries. Setup documentation MUST
    distinguish these
    unsupported capabilities from remaining Codex request metadata warnings,
    without promising warning-free compatibility or removing task instructions.
11. Local workers MUST receive an explicit context window (default 32768), an
    earlier compaction threshold (default five eighths of that window), and a
    bounded tool-output history. The CLI MUST reject invalid context/compaction
    values before creating logs. These settings MUST be documented as budgets,
    not a guarantee against server-side truncation or tokenizer differences.
12. Sandbox tool networking MUST remain disabled unless `--network` is passed.
    Worker guidance MUST distinguish blocked dependencies from a model failure,
    preserve validation command exit status, and halt when prerequisites cannot
    be satisfied. Local inference does not imply offline dependency resolution.
13. Failed worker attempts MUST retain diagnostics without advancing the
    summary's completed-phase count. Existing JSONL files remain the source for
    detailed tool activity; additional terminal activity display is out of scope.
    A new invocation reusing a log directory MUST clear the previous terminal
    error and completion time while preserving session history and run numbering.
14. Worker guidance MUST review existing partial changes before repeating work,
    avoid rereading the supplied ledger, and honor explicit tests-first task
    acceptance criteria. An expected red test is acceptable only for a task
    explicitly asking for it; implementation tasks still require passing checks.
15. <a name="15.1"></a>`--instructions-file` MUST optionally select an explicit
    Codex base-instruction file without changing the selected task prompt,
    sandbox, model routing or default behaviour. Resolve relative paths from the
    invoking directory, including spaces and quotes. Missing, non-regular,
    empty or invalid UTF-8 files MUST fail before logs or inference.
16. Rune subprocess queries MUST have a finite deadline and respect Orbit's
    shutdown context, including selection, initial phase display and progress
    verification. Failed, malformed or missing task-list output MUST NOT be
    interpreted as an empty completed ledger. Local preflight MUST check Codex
    startup with a five-second deadline before logs or inference; a broken CLI
    MUST fail promptly instead of consuming the worker timeout.
17. Verification MUST include a ledger exceeding twenty selected tasks and
    custom nested filenames with shell punctuation, relative and absolute paths,
    and symlinks invoked outside the target repository. Live code acceptance
    MUST independently check both available local models, `8-bit` and
    `bonsai-local`, and retain failed attempts as well as successful results.

Model downloads, server lifecycle management, variants, comparison changes, and
new transcript formats are outside this change. The local runtime is managed
separately; an installed model is used for live verification.

## Relationship to subsequent work

This spec owns the `orbit local`
preset and its runner-side selection, ledger validation and cancellation.
[Local run comparison](../local-run-comparison/smolspec.md) owns the pending
OpenCode adapter repair, standard variant-loop completion check and comparison
fixtures. Copying local CLI flags into a variant alias does not enable this
spec's runner guards. Neither spec implements local SDD authoring.

See [readiness and next steps](review.md) for the evidence boundary and the
small next milestone toward generating and executing approved rune ledgers.
