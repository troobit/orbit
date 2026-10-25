# Local execution

- `docs/runbooks/local-implementation-quickstart.md` is the short user guide for
  arbitrary rune ledgers with the explicit LM Studio `8-bit` model. The longer
  `local-next-task.md` runbook covers installation, diagnostics and recovery.
- `cmd/orbit/local.go` constructs an explicit `orbit.Config` with `LocalConfig`
  and the existing Codex type. It deliberately bypasses hosted Orbit aliases,
  configured prompts/hooks, `SkipPermissions` defaults and continuation. CLI
  validation and Git discovery happen before Orbit creates logs.
- `local-next-task` is an executable-name entry point in `main.go` to the same
  command. `make install-local-next-task` installs the binary and relative link,
  replacing only its own link or the known agentic-coding prototype link.
- `runSingle` still owns the loop. Local helpers select with rune, enrich the
  selection from the full ledger, construct a bounded prompt and compare the
  graph/status afterward. `rune next` omits information present in `rune list`,
  so do not build the worker prompt from `next_task` alone.
- Snapshots flatten children but retain parent links and task metadata. Only
  status changes are allowed; automatic ancestor completion is expected. At
  least one selected task must become complete per successful iteration.
- Rune's `list --filter pending` excludes in-progress tasks. Local completion
  checks inspect all tasks and include every status other than completed, so
  an interrupted final task is dispatched again when the command is rerun.
- Local failures are fatal and run once. Hosted retries retain their defaults.
  `.orbit/halt` is at the repository root, while logs are beside the ledger.
- `phase-N` log units are local iterations, not source phase ordinals. A single
  source phase can require several worker sessions. This avoids overwriting
  Orbit's standard phase artifacts during task-level execution.
- Local prompts and live stdout/stderr have UUID filenames. The Codex adapter
  passes optional writers into the shared executor while retaining in-memory
  output for Orbit's normal summaries. Stdout and stderr use separate writers.
- Only top-level Codex `error`/`turn.failed` events fail a JSONL turn. Item-level
  tool errors may be recovered by the model. Parse final assistant text for
  ORBIT-HALT without matching quoted halt instructions inside tool output.
  Local zero-exit output without a completed turn is a failure too.
- Local process-group cancellation is opt-in and implemented per platform.
  On Unix the separate group is killed on timeout/cancellation and cleaned up
  on exit. Windows keeps direct-process termination.
- `make test-local` runs real-rune integration and agent tests. The fake Codex
  is a child test process and changes only disposable ledgers. The global
  launcher and live model are verified separately; see the feature report.
- Rune write commands validate paths against their current directory. Create
  and mutate fixture ledgers from the target Git root, and resolve macOS `/var`
  symlinks before passing absolute task paths.
- Live verification with Codex 0.164.0 and LM Studio's installed `8-bit` model
  completed two dependent tasks in separate sessions (487.529 s and 358.339 s).
  All eight generated unittest cases passed independently. Local turns can
  take minutes; elapsed-time output matters even with no new JSONL events.
- That Codex version emits model-metadata refresh warnings against this LM
  Studio server. These did not prevent tool execution or completed turns.
  JSONL execution also sets RawJSON so Orbit's standard session JSON files are
  populated, alongside the unique live stdout artifacts.
- Codex 0.164.0 OSS defaults still advertise `web_search` and a `multi_agent_v1`
  namespace. Local ExtraArgs explicitly set `web_search="disabled"`,
  `features.multi_agent=false` and `model_reasoning_summary="none"`. Keep these
  scoped to the local preset. The dispatch test checks their actual argv in
  both task and phase mode.
- A real-Codex loopback request capture reproduced the LM Studio warnings.
  Cache keys, developer roles and `include: ["reasoning.encrypted_content"]`
  remain with the overrides. OSS mode ignored
  `model_supports_reasoning_summaries=false`; summary `none` removed the summary
  request but not the include. Do not claim these settings make requests fully
  compatible or strip project instructions to silence role-conversion logs.
  The runbook explains each warning; the feature verification records the
  protocol check separately from real-model inference.
- Local budgets are explicit: `--context-window` defaults to 32768 and
  `--compact-at 0` derives five eighths (20480). Tool-output history is capped
  at 2048 tokens. Match loaded LM Studio context, not the advertised maximum.
  A Codex 0.164.0 loopback check reporting 22000 input tokens triggered a local
  summary request with no tools, followed by a normal request with tools.
  This verifies config behavior, not tokenizer agreement or model summary quality.
- `--network` opts shell tools into networking without removing workspace-write
  restrictions. Cache writes outside that sandbox can still fail. Bounded
  prompts ask for direct validation commands or pipefail, and halt on unavailable
  prerequisites rather than repeat online/offline/cache workarounds.
- `SaveSession` retains failed diagnostics but must not increase completed phases
  for an error, nonzero exit or execution error. A successful Codex turn can
  still fail local ledger validation; code edits are not task completion.
- Tool activity already lives in JSONL; additional terminal rendering was
  excluded after clarification. Smaller top-level tasks reduce dispatch;
  nested children are still bundled. OpenCode remains an alternative to evaluate,
  but its current Orbit adapter expects one JSON document and does not pass live
  writers or process-group cancellation through.
