# Local Run Comparison

## Overview
Orbit can reach the LM Studio `8-bit` model in two ways: the `orbit local` Codex preset ([local-compute-agent](../local-compute-agent/)) and an OpenCode alias using `lmstudio/8-bit`. This spec runs both harnesses on the same ledger through upstream Orbit's variant mode (`orbit run --variants 2 --parallel`), so `orbit compare` reports on them like any other variant run. The first target is a disposable two-phase fixture. The sanarte `simplify-for-square` ledger in `~/repos/sanarte/.worktrees/wt-0` is a later, separately gated trial. Two Orbit defects block this today and are fixed here:
- the OpenCode adapter rejects OpenCode 2.2.0's JSON event stream;
- the variant phase loop dispatches a phase forever when a session completes nothing (`specs/bugfixes/variant-phase-loop-no-progress-guard/report.md`).

Everything else reuses existing configuration, prompts, guidance and hooks. A
Codex variant alias copies local worker flags; it does not run `orbit local` or
inherit runner selection, graph checks, iteration limits, run-once retries or
process-group cancellation. This spec measures local harness execution of an
approved ledger. It does not implement SDD authoring; the incremental path is
in [local execution readiness](../local-compute-agent/review.md).

## Delivery order

1. `tasks.md`: four coding phases for event parsing, variant completion checks,
   result commits, then the reusable fixture/configuration/runbook. Each phase
   has one top-level task; red/green children can run separately with `--leaf`.
   These tasks require Go, Git and rune, plus Python 3 for helper tests, but no
   local model or Sanarte checkout.
2. `tasks-evaluation.md`: explicit opt-in evaluation. First prove `orbit local`
   executes the fixture, then test each alias and the parallel comparison. Prove
   timeout cleanup before the long trial. Runtime prerequisites and gate release
   are in [prerequisites.md](prerequisites.md). A failed smoke check blocks Sanarte.
3. Stage and run the narrower Sanarte trial only after the smoke gate passes.
   Evaluation failure is a recorded result, not permission to relax acceptance.

The implementation review does not authorize running models or changing Sanarte.
All paths below under `docs/runbooks/` are planned artifacts, not existing files.

Rune portability: JSON intentionally exposes task numbers rather than stable
IDs; canonical Markdown retains stable markers. Batch `requirements_file`
belongs at request level and requirement IDs must be numeric (the `N.1` anchors
below name each requirement group). A batch can return exit 0 with
`success: false`; generators must check the JSON result and read back task counts,
phases and blockers instead of treating process exit alone as success.

## Requirements
- <a name="opencode-output"></a><a name="1.1"></a>**OpenCode output:**
  - Orbit MUST treat an OpenCode `--format json` session as successful only when all three hold:
    - it exits 0;
    - stdout contains at least one JSON object with a nonempty string `type` (a single event is valid);
    - no line is a `"type":"error"` event.
  - A session MUST be an error when it exits non-zero, emits an error event (even with exit 0), or contains no valid event.
  - Non-JSON lines MUST be ignored when valid events exist. Empty output, plaintext-only output, arrays, scalars, `null` and objects without an event type MUST NOT count as valid events.
  - Classification MUST use error-event messages, adapter errors and stderr, not reply or tool-output text. Plaintext stdout MAY be classified only when no valid event exists, preserving older CLI error diagnostics. An error event without a message still fails. The result MUST NOT depend on the model provider.
  - Orbit SHOULD record the event's `sessionID` as the session ID.
- <a name="variant-progress"></a><a name="2.1"></a>**Stalled variant:**
  - Before each variant session, Orbit records numeric IDs, titles, ancestor paths and statuses of incomplete tasks at every level of the dispatched phase, plus the next top-level task for diagnostics. After success, it reads the full ledger and requires at least one matching ID/title/ancestor identity to have become completed. Numeric IDs alone are insufficient because rune renumbers tasks. Phase statuses are case-insensitive; full-ledger statuses use rune's typed values. A leaf completion counts even while its parent is incomplete, and completing the final phase counts normally.
  - Pending-to-in-progress changes, deletion, task replacement, changed ordering, newly added tasks or completion outside the selection MUST NOT alone count as progress. Full graph immutability remains specific to `orbit local`.
  - When no selected task completed, Orbit MUST stop the variant through the existing variant failure path: status `failed`, a registry failure, and the log manager's failure record. The error MUST name the phase, the next task and the word "stalled". Save diagnostics with the validation error before success accounting, so the stalled session does not increment completed phases.
  - A failure to re-read the ledger MUST take the same path with the read error.
  - Other variants MUST continue.
  - Orbit MUST NOT add a new variant status or read the `.orbit/halt` sentinel.
- <a name="unattended-ledger"></a><a name="3.1"></a>**Unattended ledger:** The dispatched ledger MUST contain only tasks a worker can finish without owner credentials, production Square access, network access or browser checks.
- <a name="phase-size"></a><a name="4.1"></a>**Session-sized phases:** Each phase MUST hold one top-level task sized for one session. Use leaf subtasks only for separable checkpoints such as red/green acceptance. Acceptance MUST name `make` targets, plus tests where code can be tested. A view-only change MAY rely on `make check` and `make build`.
- <a name="owner-gates"></a><a name="5.1"></a>**Owner-gated work kept:**
  - Owner-gated tasks MUST move, rune-parseable, to `tasks-*.md` in the same spec folder.
  - Original stable IDs MUST stay with the task that keeps each original outcome.
  - Dependencies across files MUST become references, with the dependency enforced as a gate in the consuming procedure. A reference alone does not enforce readiness. No dangling `blocked_by` IDs may remain.
- <a name="shared-inputs"></a><a name="6.1"></a>**Shared inputs:** Both variants MUST start from the same commit and use the same ledger, prompt, guidance, model ID (`8-bit`), per-session timeout and declared context budget. After individual smoke checks, the measured variants MUST run in parallel on the same LM Studio load. Timings describe contention, not isolated harness throughput; model identity and loaded context must be verified, not inferred from the `8-bit` alias.
- <a name="harness-boundaries"></a><a name="7.1"></a>**Harness differences recorded:** The record MUST list each harness's restrictions and the instruction files it loaded.
  - The Codex variant MUST run sandboxed. It applies the local worker flags from `orbit local`: LM Studio provider, ignored user config, workspace-write sandbox, no tool networking, and web search, multi-agent and apps disabled.
  - The OpenCode variant MUST deny web fetch, subagents and `git commit`/`git push`, and keep its helper model local. Its shell is not sandboxed.
- <a name="commits"></a><a name="8.1"></a>**Commits:** Workers MUST NOT commit. The staging branch MUST ignore `.orbit/` at any depth, because Orbit writes `summary.json` there before variant setup checks for uncommitted files.
  - An Orbit post-command MUST commit each completed variant's changes, only when changes exist. The reusable helper takes the ledger and exact base commit explicitly, checks that base is an ancestor, and uses `ORBIT_AGENT` for the alias. Validate state recursively with rune JSON and map numeric IDs to stable IDs using `rune list --format markdown` canonical task lines. JSON deliberately omits stable IDs. Join each revision separately before comparing stable identities; do not equate a task number with a stable ID. Missing/duplicate stable IDs fail the helper rather than being invented.
  - The helper MUST fail on a missing/unreadable base ledger or malformed current ledger without making a commit. A second invocation with no changes MUST be a no-op. Tests use temporary repositories with nested tasks and paths containing spaces. Only expected fixture/source/ledger paths are staged; logs, secrets and owner ledgers are excluded.
  - The commit message MUST name the alias and the IDs of the tasks completed since the base commit, found by comparing rune state with the base commit's ledger.
  - A failed post-command only logs a warning, so the procedure MUST confirm that each `orbit-impl/*` branch has a commit beyond the base before using the report.
- <a name="comparison"></a><a name="9.1"></a>**Comparison:** When both variants complete, the run MUST produce an Orbit comparison report without hand-edited `variants.json`. The judge is the staging repository's default agent (`claude-code`). It is not under test and sees summaries rather than diffs. When fewer than two complete, judging is skipped and there is no comparison recommendation; Orbit may still generate a partial/status report. Record each variant's status/error and distinguish that report from a successful judge result. Even with two completed variants, HTML generation alone does not prove judging succeeded.
- <a name="timeouts"></a><a name="10.1"></a>**Timeouts:** 60m is a per-attempt limit, not a run deadline. Standard variant retries remain adapter-dependent: a `signal: killed` can be reclassified from transcript text. Up to five normal attempts are possible, with additional bounded rate-limit waits. Each observed attempt, timeout and retry MUST be recorded with its cause; do not promise timeout means no retry. A timeout-cleanup failure blocks long runs.
- <a name="record"></a><a name="11.1"></a>**Record:** `runs.md` MUST record each run in a fixed table:
  - the Orbit commit, the Codex and OpenCode versions, and the rune repository commit
  - `lms ps` output and all run settings
  - per variant: status and error, tasks completed (rune state in the worktree), session durations, and finish time. `summary.json` `sessions[]` keeps only each dispatch's final retry attempt.
  - retries and timeouts with their causes, taken from Orbit console output captured with `tee`
  - host-run Makefile results against each final state: `make test` for the standard-library fixture; `make check`, `make test` and `make build` for Sanarte
  - the comparison report path and its recommendation, or the explicit reason judging/report generation was unavailable
  - local runner versus alias/variant dispatch mode, independent red/green checks, and any skipped owner acceptance

  Single-run conclusions MUST be labelled indicative.
- <a name="isolation"></a><a name="12.1"></a>**Reusable and isolated:** The procedure MUST be documented for other ledgers, including how to reset before a repeat run. It MUST NOT change the `simplify-for-square` branch or the main sanarte checkout. The `orbit-impl/*` branches and worktree metadata it creates MUST be removed by the reset.

## Disposable fixture contract

Generate a standalone Python standard-library project with `word_utils.py`,
`test_word_utils.py`, a Makefile and `specs/word-utils/` containing a fixed small
approved design/decision record and rune ledger. Both functions begin as
`NotImplementedError` stubs; tests are provided before worker execution.

- Phase 1 implements `normalize_label(text: str) -> str`: trim surrounding
  whitespace, collapse internal whitespace with `str.split()`/join and lowercase.
  Cases include `"  HeLLo \\t WORLD  "` → `"hello world"`, whitespace-only →
  `""`, and unchanged normalized text. Acceptance is `make test-normalize`;
  the untouched phase-2 stub is still expected to fail its separate tests.
- Phase 2 implements `unique_labels(values: list[str]) -> list[str]`: use
  normalization, omit empty labels, deduplicate and preserve first-seen order.
  For `[" BETA ", "alpha", "Beta", "", " ALPHA "]` expect
  `["beta", "alpha"]`; also test empty input and whitespace-only labels.
  It is blocked by phase 1. Acceptance is `make test` for all cases.

Use `unittest` with separate test classes and Makefile targets
`test-normalize`, `test-unique` and `test`; no packages are downloaded. Create
one top-level code task per phase using rune with stable IDs, requirement links
to that fixture's approved design, explicit acceptance and no owner gates.
The generator must refuse to overwrite a nonempty destination. Its automated
tests validate generated files, actual failing baseline checks, phase/dependency
structure and repeat-run refusal without invoking any model.

Direct local acceptance uses `orbit local <ledger> --phase` with runner selection
and a two-iteration limit to prove actual phase dispatch. Aliases use standard
variant orchestration on fresh copies of the same fixture base. Never treat the
phase-2 expected baseline failure as a reason to implement it during phase 1.

## Implementation Approach
- **OpenCode fix:**
  - In `internal/agents/opencode/agent.go` (result handling near line 412), replace the whole-output `json.Valid` check with a line-by-line event scan: collect `error.message` from `type: error` events, take the session ID, and accept a single typed event. Arrays/scalars and untyped objects are not events.
  - `Classify` in `internal/agents/opencode/errors.go` uses the same scan.
  - Follow `internal/agents/codex/agent.go:261`.
  - Test with OpenCode 2.2.0 fixtures:
    - a two-event success stream
    - `{"type":"error","error":{"type":"provider.no-route","message":"Model unavailable: …"}}` with exit 1
    - the same error event with exit 0
    - a stream mixing valid events and non-JSON lines
    - plaintext and empty output
    - a success stream whose text mentions "timeout" and "api key"
- **Stall guard:**
  - In `runVariant` (`internal/orbit/variants.go`), capture incomplete numeric IDs, titles, ancestor paths and statuses recursively from the dispatched `GetNextPhase` result. After a successful session use `variantRuneClient.ListAll()` to find an actual selected-identity transition to completed, including the final task and nested leaves. Do not compare the next phase to the dispatched phase.
  - Do this before `SaveSession`/`CompletePhase`. On no completion or a ledger read error, attach the validation error to the retained result and use the existing failure branch. Do not rerun through the agent retry loop after a successful but stalled turn. Keep siblings running.
  - This spec owns this guard. The older `run-termination` count/next-ID fingerprint and blocked-state proposal are superseded for this variant guard; that spec's remaining work needs reconciliation, not a second implementation.
  - Compare statuses with `strings.EqualFold`, as `internal/orbit/local.go` does. Rune emits title-cased statuses.
  - Test with `internal/testutil` scenarios:
    - an agent that never completes a task stops after one session, and its sibling variant completes;
    - completing one leaf counts as progress, followed by completion of the final phase;
    - in-progress-only, deletion/replacement, additions and unrelated completion do not pass;
    - post-session rune failure records the read error and terminates;
    - stalled diagnostics remain and summary completion does not advance.
  - The count-plus-next-ID proposal is **superseded by Decision 4**: identity churn can change that fingerprint without completing work.
- **Staging (evaluation only):** First pass the disposable fixture gates. Then inspect wt-0 for a clean tree and confirm its base is still `fd203be6415c2e41af0ec7a04ee03e155fc55b02`; if it differs, record and review the actual base rather than reset it. Commit the reviewed ledger/configuration and `.orbit/` ignore rule on wt-0. Capture the new staging commit as the variant base. From wt-0, run with Bash pipeline failure propagation:

  ```
  set -o pipefail
  orbit run --variants 2 --parallel --skip-permissions=false \
    --no-pre-prompt --no-auto-consolidate \
    --variant-agents codex-local,opencode-local \
    --guidance-file specs/simplify-for-square/local-variant-guidance.yaml \
    --tasks-file specs/simplify-for-square/tasks.md \
    --no-post-prompt 2>&1 | tee <log>
  ```

  `--skip-permissions` defaults to true, and `variants.go:459` then forces auto-approve. That adds Codex's sandbox bypass (`internal/agents/codex/agent.go:214`). `wt-1` stays at base for a repeatability run.
- **Superseded ledger instruction (Decision 4):** The original plan was: "Rewrite sanarte tasks 1 (iframe fallback) and 2 (Book Now header) into session-sized phases with rune." It did not preserve ownership of their full acceptance criteria. The code locations and deferred checks remain useful context: Use the eight code locations in sanarte's `specs/simplify-for-square/smolspec.md`: `src/lib/server/sanarte.ts`, `src/lib/server/square-appointments.ts`, `src/lib/square-booking.ts`, `src/lib/blocks/SquareBooking.svelte`, `src/lib/engine/registry.ts`, `src/routes/[...path]/+page.server.ts`, `src/lib/components/Nav.svelte` and `src/lib/styles/base.css`. `make test` runs only `src/lib/server/square-appointments.test.ts`, so a new test file is added to that target. Move tasks 3, 4, 6 and 7, task 1's widget-policy confirmation and the 320px, 375px and desktop browser checks to `specs/simplify-for-square/tasks-owner-gated.md`.
- **Current ledger rule:** Keep original IDs for tasks 1 and 2 with their full browser/widget acceptance in `tasks-owner-gated.md`. Create narrower unattended code tasks with new stable IDs and references back to those outcomes; completing them does not complete the originals. Keep already-completed historical tasks complete. Do not change the source Sanarte smolspec or treat iframe policy as settled. Each code task has make-based checks and explicit exclusions. Move every other owner-dependent outcome out of the dispatched ledger and verify no blocker references a task now in another file.
- **Aliases** (`AgentAliasConfig` fields only, `internal/config/config.go:53`):
  - `codex-local`: type `codex`, model `8-bit`, and `auto-approve: false`, because alias auto-approve defaults to true (`internal/config/config.go:545`). Its `extra-args` copy `cmd/orbit/local.go:91-100`, including `--json`, with context 65536, compaction at 40960 and networking off.
  - `opencode-local`: type `opencode`, model `lmstudio/8-bit`, `extra-args: [--auto]`. A committed `opencode.json` defines:
    - the `lmstudio` provider at `http://127.0.0.1:1234/v1`
    - the 65536 context limit, with a local `small_model`
    - `share: disabled`
    - deny rules for `webfetch`, `task`, and `git commit*`/`git push*` in bash

  Both aliases use `timeout: 60m`. The disposable smoke fixture may use a smaller matching context budget verified against the current load; record it. Sanarte requires the declared 65536 budget for both, but this spec never reloads a model. The Sanarte pre-command is `pnpm install --frozen-lockfile --offline`, run only when `pnpm-lock.yaml` exists. Both worktrees need prepared writable caches; failure stops before inference. The standard-library fixture has no install step. The post-command runs a commit script copied from `docs/runbooks/local-variant-commit.sh`.
- **Prompt and guidance:**
  - The top-level `command` names the ledger explicitly, tells workers to retrieve the next phase with `rune next <ledger> --phase --format json`, and binds work to that selection. It supplies the absolute installed next-task skill path, uses sequential execution, and forbids commits. With one top-level task per phase this is model-retrieved phase execution, not Orbit runner selection.
  - The guidance file is a copy of `docs/runbooks/local-variant-guidance.yaml`. It is parsed in `cmd/orbit/run.go:532`, merged at 559-569, and rendered under "Guidance for this Implementation" (`variants.go:818-823`).
  - Its `global_guidance` carries bounded worker rules from `localPrompt` (`internal/orbit/local.go`): respect repository instructions, use Makefile checks, change only selected task statuses, read existing partial edits, and stop after the selection. The fixture uses one stream and requires no spec edits or overview changes beyond task statuses. If a skill or project rule needs a human or conflicts with the bounded task, state the blocker and stop. A variant does not consume local halt sentinels; its completion guard must detect a blocked session. Do not describe prompt rules as enforced sandbox limits.
- **Docs:** `docs/runbooks/local-variant-comparison.md` covers the procedure, independent checks and reset (`orbit cleanup simplify-for-square`, then remove `.orbit/` and `comparison-report/`). It also defines the smoke run: a disposable standard-library Python repository with a fixed approved spec and a rune-created ledger containing two dependent tasks in two phases, with phase-specific Makefile acceptance and a final `make test` target. First use `orbit local` without aliases through both phases, then reset and run each alias, then the parallel variant comparison from the same base. Preserve each result before resetting. The smoke run passes when:
  - Codex's `--debug` arguments omit the bypass flag;
  - the direct local run completes both phases with passing independent tests, and each alias/variant completes both with a host-side commit;
  - a report contains a successful judge result and a recommendation naming a completed variant; a partial/status-only HTML report does not pass this check;
  - a separate forced 2-minute per-attempt timeout probe with no competing inference leaves no measured request surviving the terminated Orbit invocation, and LM Studio idle within 60 seconds of that termination. Record retries so a fresh attempt is not mistaken for an orphan. If another user shares the load, record the check as blocked rather than attribute their activity to an orphan. OpenCode may use `--standalone`, supported by installed 2.2.0; recheck actual cleanup, not merely the argv.

  Add the guidance and commit-script templates beside the runbook. Link both from `docs/runbooks/local-next-task.md` and `docs/agent-notes/local-execution.md`. Results go in `specs/local-run-comparison/runs.md`. Coding acceptance is `make test-local`, `make test`, `make lint` and `make build`, plus script syntax and rune parsing checks; runtime evaluations are not prerequisites for merging a tested adapter fix.
- **Out of Scope:**
  - the remaining `run-termination` work: a blocked status, the sentinel, the iteration ceiling, single-run stall detection and classification
  - retry policy and the Codex classifier
  - judge diffs or a judge override
  - OpenCode live output and process groups
  - changing `orbit local` or the `next-task` skill
  - merging the ledger into `simplify-for-square`
  - running owner-gated tasks, copying `.env` into worktrees, and LM Studio reconfiguration

## Risks and Assumptions
- Risk: Retries differ by adapter, not harness quality. Codex's classifier scans transcript text, and a retried phase can run up to five attempts (`orbit.go:28`). | Mitigation: The timeout is 60 minutes. Record each retry and its cause per harness.
- Risk: OpenCode `run` uses a background service, so a killed session may keep inferring. | Mitigation: The smoke run forces a short timeout and checks that LM Studio becomes idle. If it doesn't, add `--standalone` and recheck.
- Risk: Once one variant finishes, the other runs without contention. | Mitigation: Record finish times.
- Risk: The OpenCode harness fetches models.dev metadata. | Mitigation: Prompts stay local through the local `small_model`. Record any outbound requests.
- Unverified assumption: Both harnesses follow `~/.agents/skills/next-task/SKILL.md` when given its absolute path, and `8-bit` (262144-token context, 4 slots) serves two 65536-token sessions. The smoke run verifies instruction loading and usable capacity, and checks Codex's arguments with `--debug`. No capacity or cleanup claim is accepted solely from configuration.
- Prerequisite: The LM Studio server is serving `8-bit`, the pnpm store holds sanarte's dependencies, and `orbit` (built from this branch), `rune`, `codex` and `opencode` are on PATH.

## Acceptance boundary

All coding phases are locally testable without model inference. Live comparison
requires the explicit evaluation gate. A completed variant, a commit, a judge
recommendation or a green type check alone does not establish feature acceptance.
The Sanarte owner ledger and full SDD authoring remain outside unattended work.

## Escalation Note
This change was scoped as a smolspec. If implementation reveals ambiguity only the user can resolve, an irreversible boundary (public API, persisted schema, auth path), or a contested architectural choice, stop and escalate to the full spec workflow rather than deciding it inline.
