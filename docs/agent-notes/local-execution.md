# Local execution

- Reliability verification for 2026-10-30 lives in
  `specs/local-compute-agent/verification-reliability.md`. Default
  `--max-iterations 0` derives a finite budget from initial incomplete tasks,
  including descendants; explicit positive limits retain their cap. Each
  accepted iteration completes selected work and cannot grow the graph.
- Both new two-task model fixtures pass eight unchanged host tests. The final
  installed Bonsai timeout/recovery passes too. An additional eight-bit recovery
  completed its code/statuses but kept verifying until the ten-minute deadline;
  host tests passed and an independently accepted no-work replay finalised the
  ledger. Preserve that failed turn: a successful replay with no incomplete
  tasks is not evidence that the previous worker succeeded or validated code.
- Rune queries now use the shared process-group executor, a 30-second deadline
  and Orbit's shutdown context. Reject explicit failures and malformed/missing
  task payloads; valid rune empty-list envelopes remain accepted. A worker's
  timeout alone cannot bound the rune calls between sessions.
- Local preflight probes Codex with a five-second bounded version check before
  logging. Tests use fake CLI success, nonzero exits and a stalled child rather
  than invoking a host-installed CLI in an assertion-free test.
- A host Codex cask launch stalled before user code while its byte-identical,
  signed app copy worked. Removing quarantine or replacing the inode did not
  repair that resolved path. The original cask file/attributes were restored;
  `/opt/homebrew/bin/codex` now points at the working app copy, version 0.165.0.
  Original link target and verification evidence are retained under
  `.orbit/local-reliability/codex-launch-quarantine-backup.json`. A future
  Homebrew upgrade may replace this link; verify startup if that happens.
- M5 verification is in `specs/local-compute-agent/verification-m5.md`: six
  successful two-task runs, including the direct new `--instructions-file` option
  and retained `AGENTS.md` guidance. Native dispatch needs no PATH wrapper.
- The installed Qwen3.8 custom 8-bit MLX template defaulted to `xhigh`, while
  its LM Studio entry did not expose a reasoning API setting. Both template
  files were backed up and their fallback changed to `low`, then the same model
  was reloaded. Runtime defaults and Codex base-instruction size are separate
  controls; inspect actual model metadata/template rather than infer from alias.
- Optional `--instructions-file` resolves from the invoking directory, validates
  a regular nonempty UTF-8 file before logging, and uses an encoded absolute path
  in Codex's supported `model_instructions_file` override. The example compact
  policy is `docs/runbooks/local-worker-instructions.md`; defaults stay intact.
- A supervised local planning prototype generated JSON that the host validated,
  reviewed and materialised through rune for Sanarte. A truncated response was
  rejected; a parseable follow-up still had a warning-argument arity mistake,
  corrected in a separate reviewed plan. Its real RED/GREEN workers completed
  and host checks passed all 16 tests, type checking and build. Structural
  validity is not semantic acceptance. `orbit prepare` remains unimplemented.
- A failed worker can leave a completed rune task even when no Orbit phase was
  accepted: the Bonsai 32K trial timed out after its first implementation/task
  update. Recheck retained code independently before resume. The sibling 16K
  trial completed both tasks; the failed case then resumed at 16K and finished
  its remaining task in 69.773s with all eight host tests passing. Do not infer
  a worker budget from model max context.
- `specs/local-task-preparation/` plans the next layer: source-ledger decomposition
  into reviewed packets, host-owned rune completion, independent checks and
  shared prepared dispatch in local/single/variant paths. It is not implemented.
  Ordinary local execution still asks the model to update rune statuses.
- Authorised Sanarte wt-2 trials on 2026-10-29 retained a 19m18.893s interrupted
  broad task and a 10-minute five-scenario RED timeout, both without source edits.
  Narrowing a title alone did not remove context retrieval/design overhead.
  Further trials supply an exact regression packet; results and limitations
  belong in the preparation spec's evaluation record. Do not treat narrow
  diagnostics work as original iframe/provider/browser acceptance.
- `specs/local-compute-agent/review.md` records execution readiness and the
  boundary to future local SDD authoring. Its implementation ledger is complete;
  `specs/local-run-comparison/tasks.md` owns pending OpenCode event parsing,
  standard variant completion checks and disposable comparison fixtures.
- Variant aliases copy worker arguments, not `LocalConfig` selection, graph
  validation, retry bounds or process-group cleanup. Keep direct local smoke
  acceptance separate from comparison results. The comparison evaluation ledger
  has STOP gates; the older run-termination ledger must be replanned before use.
- Rune JSON exposes numeric task IDs and deliberately omits stable IDs;
  canonical Markdown retains stable markers. Batch `requirements_file` is a
  request-level field, requirement IDs are numeric, and exit 0 can accompany
  `success: false`. Read back the ledger after writes. Stream readiness does not
  inherit parent blockers, so children that depend on earlier phases need
  explicit prerequisites too.

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
  `features.multi_agent=false`, `features.apps=false` and
  `model_reasoning_summary="none"`. Keep these
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
  nested children are bundled by default; `--leaf` dispatches them separately.
  OpenCode remains an alternative to evaluate,
  but its current Orbit adapter expects one JSON document and does not pass live
  writers or process-group cancellation through.
- The longer authenticated run exposed seven additional connected-app namespaces
  (Pets, Sites, app management and others). `--ignore-user-config` still uses
  normal authentication, which can expose apps absent from a clean Codex home.
  An authenticated loopback capture reproduced them; `features.apps=false`
  removed them and retained seven function tools with shell networking enabled.
  This is separate from the `--network` permission; the user prefers networking
  enabled for subsequent invocations of the observed run.
- Flat-directory restarts must clear summary Error and CompletedAt when setting
  Status to running. The live run otherwise showed a prior failure alongside
  the new active session. Preserve history and phase-resumption state; do not
  patch a summary file owned by an active runner.
- The longer authenticated iteration `87438257-97a0-4883-a6cb-cb3b6dc88814`
  completed task 1 in 1098.955 seconds. Live compaction at 17:26:04 resumed
  at 17:30:34, reducing conversation items from 55 to five and server prompt
  count from 20243 to 10947. No middle truncation appeared in that iteration;
  tests and Ruff checks passed, rune completed the task and Orbit began iteration
  2. This does not validate the entire continuing ledger run. The worker still
  ignored some pipeline/echo guidance, so treat prompts as advisory.
- `--leaf` is runner-only and incompatible with `--phase`. It descends through
  rune's selected subtree in document order, skips blocked leaves, and checks
  dependencies on every ancestor. STOP checks inspect the original subtree
  before narrowing so a later gate cannot disappear. No ready leaf is an error;
  the runner does not move to an unrelated top-level task.
- Leaf prompts include ancestor details/references as read-only context, omit
  sibling work, and give rune commands only for the leaf. The existing snapshot
  and progress validator still own graph/status checks and allow automatic
  ancestor completion. Use normalized snapshot children when descending so
  their parent links remain available for context construction.
- Tests-first acceptance must explicitly describe the expected failure. Worker
  guidance permits that red checkpoint without implementing the next task;
  implementation tasks still need passing checks. Verify the actual failure
  independently: live workers still use trailing echoes and reread the ledger.
  A one-iteration invocation exits with a limit error if work remains even when
  its recorded worker session succeeded; rerun after reviewing that checkpoint.
- Native delegation remains disabled because LM Studio ignores its namespace.
  Leaf workers run sequentially with fresh contexts. Extra session setup can
  cost more than bundling tiny tasks; the live efficacy report in
  `specs/local-compute-agent/verification.md` records supervised fixture runs,
  limitations and retained artifacts under the target's `.orbit/local/evaluations`.
