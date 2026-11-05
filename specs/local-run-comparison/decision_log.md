# Decision Log: Local Run Comparison

## Quick Decisions

| ID | Date | Decision | Rationale |
|----|------|----------|-----------|
| Q1 | 2026-10-28 | Scope as a smolspec in `specs/local-run-comparison/` | superseded by Q20 |
| Q2 | 2026-10-28 | The two variants differ by harness only: Codex preset flags versus OpenCode `lmstudio/8-bit` | User choice; tests the overlap between `orbit local` and OpenCode as local harnesses |
| Q3 | 2026-10-28 | Stage the ledger, configuration and run in `~/repos/sanarte/.worktrees/wt-0`; keep `wt-1` at base | User choice; keeps the main sanarte checkout and `simplify-for-square` plan of record unchanged |
| Q4 | 2026-10-28 | Tasks cover the process (ledger, configuration, runbook, runs) plus the OpenCode adapter fix; the spec-writing session itself makes no code edits | superseded by Q20 |
| Q5 | 2026-10-28 | Owner-gated sanarte tasks move to `tasks-owner-gated.md` on the staging branch | `rune next --phase` returns every incomplete task, including in-progress tasks and gate placeholders, so they cannot stay in a ledger that variant mode dispatches |
| Q6 | 2026-10-28 | One top-level task per phase | Variant mode dispatches by phase; one task per phase makes the dispatch unit the same for both harnesses and comparable to `orbit local`'s one-task default |
| Q7 | 2026-10-28 | Declared context budget 65536 (Codex compaction at 40960) for both harnesses | A quarter of the 262144-token load across 4 parallel slots; record the actual LM Studio load per run |
| Q8 | 2026-10-28 | Disable the post-prompt for comparison runs | A review session doubles local inference time and is not part of the harness under test |
| Q9 | 2026-10-28 | Commit variant results in an Orbit post-command, not in the worker | `orbit compare` diffs `base..HEAD`; a host-side commit avoids sandboxed `.git` writes and keeps both harnesses identical |
| Q10 | 2026-10-28 | Use the staging repository's default `claude-code` agent as the comparison judge | The judge is not under test; in-run comparison follows `--agent`, which defaults to that alias |
| Q11 | 2026-10-28 | Fix the OpenCode adapter's JSON handling in this spec | User instruction to roll the bugfix in. A live OpenCode 2.2.0 probe emitted two JSON events per successful run; `json.Valid` over the whole output marks it as an error, and the classifier then scans transcript text, so the variant fails or reruns completed work |
| Q12 | 2026-10-28 | Per-session timeout 60m for both aliases | superseded by Q24 |
| Q13 | 2026-10-28 | Treat sandbox and network differences as part of the harness under test and record them | Codex can be sandboxed; OpenCode has no OS sandbox. Matching them would require changing one harness, which defeats the comparison |
| Q14 | 2026-10-28 | Label single-run conclusions indicative; keep `wt-1` for a repeatability run | One run per harness cannot separate harness effects from model sampling variance |
| Q15 | 2026-10-28 | Keep the variant-mode halt and no-progress guard out of scope | superseded by Decision 2 |
| Q16 | 2026-10-28 | Run with `--skip-permissions=false` | superseded by Q21 |
| Q17 | 2026-10-28 | Commit the guidance file on wt-0 as a copy of the Orbit template | Variant setup rejects a checkout with untracked files, and a copy keeps the run reproducible from the staging commit |
| Q18 | 2026-10-28 | Keep original stable IDs on the task that keeps each original outcome; dependencies across files become references | Rune treats a missing blocker as permanently unready, and keeping IDs lets the plan of record be reconciled later |
| Q19 | 2026-10-28 | Workers choose their own task through the next-task skill | Variant mode sends a fixed prompt without a task; this matches `orbit local --selection model`, not its default where Orbit selects the task |
| Q20 | 2026-10-28 | Scope as a smolspec covering the run process plus two Orbit defect fixes: OpenCode event-stream parsing and variant stall detection | User directed that the spec carry the code changes it needs; both are revertable fixes to reported defects with no public surface or schema change |
| Q21 | 2026-10-28 | Set `auto-approve: false` on `codex-local` and run with `--skip-permissions=false` | Alias auto-approve defaults to true (`internal/config/config.go:545`) and `--skip-permissions` forces it on (`internal/orbit/variants.go:459`); either one adds Codex's sandbox bypass (Codex peer review) |
| Q22 | 2026-10-28 | Ignore `.orbit/` at any depth on the staging branch | Orbit writes `summary.json` under the spec's `.orbit/` before variant setup rejects untracked files (`internal/orbit/orbit.go:272`, `internal/variants/manager.go:186`); ignoring it also keeps runtime files out of variant commits |
| Q23 | 2026-10-28 | Completing any task in the dispatched phase, including a leaf, counts as progress | Matches `orbit local`, where completing a selected task or subtask satisfies progress; counting only top-level tasks would stop legitimate partial progress under one-task-per-phase ledgers |
| Q24 | 2026-10-28 | Per-session timeout 60m; a timeout fails the variant and is recorded rather than retried — superseded by Decision 4 | The rationale already allowed classifier-dependent retries; the decision overstated the guarantee |
| Q25 | 2026-10-28 | A run where fewer than two variants complete is a recorded result, without a comparison report — superseded by Decision 4 | Judging is skipped, but Orbit can still generate a partial/status report; a failed harness remains evidence |
| Q26 | 2026-10-28 | OpenCode non-JSON stdout lines are ignored; success needs exit 0, at least one valid event and no error event | Matches the Codex JSONL parser (`internal/agents/codex/agent.go:261`) and avoids failing a completed session over a stray line |

---

## Decision 1: Compare Local Harnesses Through Orbit Variant Mode

**Date**: 2026-10-28
**Status**: superseded by Decision 3

### Context

Orbit has two routes to the local `8-bit` model. `orbit local` drives Codex with Orbit-side task selection and progress guards, but excludes variants and comparison. An OpenCode alias with `lmstudio/8-bit` runs through the standard `orbit run` loop, which supports variants and `orbit compare` but has no local guards. The goal is to compare the two on the same ledger and have `orbit compare` report on the result. The user directed that this use the same harness as `orbit run --parallel` comparisons, re-wiring existing agent instructions and process rather than changing code.

### Decision

Run both harnesses as variant agents (`codex-local`, `opencode-local`) in one `orbit run --variants 2 --parallel` invocation from the wt-0 staging checkout. Express the `orbit local` restrictions as alias `extra-args` and its worker rules as `global_guidance`.

### Rationale

Variant mode already creates comparable worktrees from one base commit, records per-variant metadata and runs the comparison. Agent aliases, guidance files, the `command` prompt and pre/post-commands are existing configuration surfaces. Together they reproduce the parts of the local preset that matter for a harness comparison. The only Go change needed is the OpenCode adapter fix (Q11), which is a defect rather than a new capability.

### Alternatives Considered

- **Run `orbit local` in wt-0 and OpenCode in wt-1, then hand-write `variants.json`**: Use the existing worktrees and register them manually - Rejected because `orbit local` leaves edits uncommitted, the hand-written metadata must mimic Orbit's internal schema, and the two runs would use different loops, prompts and dispatch units, so a difference could not be attributed to the harness.
- **Add `orbit local` as a variant agent type**: Make the local preset selectable per variant - Rejected because it adds a new orchestration surface before there is evidence that the preset's guards matter; this comparison supplies that evidence.
- **Compare by hand without `orbit compare`**: Diff the two worktrees manually - Rejected because the user asked for runs that `orbit compare` can consume.

### Consequences

**Positive:**
- No new orchestration code; the comparison report, status and registry work as for hosted variants
- Both harnesses share the loop, prompt, guidance and dispatch unit, so differences are attributable to the harness
- The process is reusable for other ledgers through the runbook

**Negative:**
- Variant mode lacks `orbit local`'s halt, no-progress and graph checks, so a stuck worker is re-dispatched until timeout or cancellation
- The Codex variant runs under the hosted retry policy rather than the local run-once policy
- The OpenCode variant has no OS sandbox; only its permission configuration limits it

### Impact

`internal/agents/opencode/` (adapter fix), sanarte staging branch `wt-0` (`.orbit.yaml`, `opencode.json`, guidance file, ledgers), Orbit runbooks under `docs/runbooks/`, and `docs/agent-notes/local-execution.md`.

---

## Decision 2: Stop Stalled Variants Through the Existing Failure Path

**Date**: 2026-10-28
**Status**: superseded by Decision 4

### Context

Variant mode loops on `GetNextPhase` until rune reports all tasks complete (`internal/orbit/variants.go:616-719`). A session that returns successfully without completing a task is dispatched again with no limit. `specs/bugfixes/variant-phase-loop-no-progress-guard/report.md` records a run that repeated one human-gated phase 2,617 times across two variants, for about $830. Local models make this more likely, and the comparison runs last hours unattended. Ctrl+C cancels the whole process, which stops both variants.

`specs/run-termination` (pending, 25 tasks) plans a progress fingerprint, a new `blocked` variant status (its Decision 2), an iteration ceiling and a `.orbit/halt` sentinel. The sentinel and the `ORBIT-HALT:` output line are conventions specific to this fork. Upstream Orbit (`ArjenSchwarz/orbit`, main at 7259a6c) and its `next-task` skill have neither, and no upstream branch or PR fixes the loop. In this fork only `orbit local` reads the sentinel. Its single live halt evidence is the August incident, where Claude wrote the file and Orbit never read it. No live 8-bit run has recorded a halt. The user directed that upstream Orbit's process is the standard to build on, and that a stuck variant should be removed the same way existing variants are.

### Decision

After each successful variant session, compare the progress fingerprint from `specs/run-termination/design.md` (incomplete count plus next task ID) with its value before the session. If it is unchanged, end that variant through the existing failure branch: status `failed`, a registry failure and the log manager's failure record, with an error naming the phase, the next task and "stalled". Other variants continue. Do not read `.orbit/halt` and do not add a status.

### Rationale

Progress detection stops every stall, whether or not the agent signals it. `run-termination`'s own analysis makes it the primary guard and the sentinel optional (its requirement 3.6). A halted session also completes no task, so reading the sentinel would add only a reason string. Reusing the failure path keeps upstream's variant lifecycle: failed variants are already excluded from comparison and are removed by `orbit cleanup` or recreated by "New run". This fixes a reported defect rather than adding a parallel mechanism.

### Alternatives Considered

- **Implement `run-termination` phases 1–2 as written**: Add a distinct `blocked` status and read the sentinel - Rejected at the user's direction to build on upstream's existing variant states; it changes the `variants.json` status enum and the status display before there is evidence a separate state is needed.
- **Process only**: Document killing the stuck variant's agent process so that variant fails through the existing path - Rejected because the runs are long and unattended, and a fast-halting worker produces a new process every few seconds.
- **Host-side `cli-path` wrapper**: Refuse dispatch when `.orbit/halt` exists and cap sessions per phase - Rejected because it copies logic that `orbit local` already has into a third place.

### Consequences

**Positive:**
- No variant can loop indefinitely on a phase it cannot complete
- One stuck variant no longer forces the operator to cancel the other
- Uses existing variant states, comparison filtering and cleanup

**Negative:**
- Overrides `run-termination` Decision 2: a stalled variant reports `failed`, which that spec found was misread as a code failure; the "stalled" error text is the only distinction
- A legitimate session that makes progress without completing a task, or that only changes task status outside the selected phase, is stopped
- Single-run mode keeps the defect until `run-termination` task 5 is done

### Impact

`internal/orbit/variants.go`, `internal/testutil` scenarios, and `specs/run-termination/tasks.md` (tasks 2–3 are delivered here; tasks 1 and 8–10 need reconciling with this decision).

---

## Decision 3: Compare Local Harnesses Through Orbit Variant Mode With Two Defect Fixes

**Date**: 2026-10-28
**Status**: accepted

**Clarification (2026-10-29):** Decision 4 defines delivery order, task-completion
semantics and ledger ownership. The alias configuration does not reproduce
`LocalConfig` runner behavior. This decision retains the variant comparison
route and two defect fixes, not a claim of equivalent local guards.

### Context

Orbit has two routes to the local `8-bit` model. `orbit local` drives Codex with Orbit-side task selection and progress guards, but excludes variants and comparison. An OpenCode alias with `lmstudio/8-bit` runs through the standard `orbit run` loop, which supports variants and `orbit compare` but has no local guards. The user directed that the comparison use the same harness as `orbit run --parallel` comparisons, re-wiring existing agent instructions and process, and that the spec carry the code changes it needs.

Research and review found that variant mode can carry the comparison only with two defects fixed. Orbit's OpenCode adapter marks every OpenCode 2.2.0 session as an error. The variant phase loop re-dispatches a phase forever when a session completes nothing (Decision 2).

### Decision

Run both harnesses as variant agents (`codex-local`, `opencode-local`) in one `orbit run --variants 2 --parallel` invocation from the wt-0 staging checkout. Express the `orbit local` restrictions as alias `extra-args` and its worker rules as `global_guidance`. Fix the OpenCode event-stream parsing and add the variant stall guard in this spec. Add no other orchestration code.

### Rationale

Variant mode already creates comparable worktrees from one base commit, records per-variant metadata and runs the comparison. Agent aliases, guidance files, the `command` prompt and pre/post-commands are existing configuration surfaces, and together they reproduce the parts of the local preset that matter for a harness comparison. The two fixes repair reported defects in upstream behaviour rather than add a parallel mechanism.

### Alternatives Considered

- **Run `orbit local` in wt-0 and OpenCode in wt-1, then hand-write `variants.json`**: Use the existing worktrees and register them manually - Rejected because `orbit local` leaves edits uncommitted, the hand-written metadata must mimic Orbit's internal schema, and the two runs would use different loops, prompts and dispatch units, so a difference could not be attributed to the harness.
- **Add `orbit local` as a variant agent type**: Make the local preset selectable per variant - Rejected because it adds a new orchestration surface before there is evidence that the preset's guards matter; this comparison supplies that evidence.
- **Compare by hand without `orbit compare`**: Diff the two worktrees manually - Rejected because the user asked for runs that `orbit compare` can consume.

### Consequences

**Positive:**
- No new orchestration surface; the comparison report, status and registry work as for hosted variants
- Both harnesses share the loop, prompt, guidance and dispatch unit, so most differences are attributable to the harness
- The OpenCode fix also repairs hosted `orbit run --agent opencode`
- The process is reusable for other ledgers through the runbook

**Negative:**
- Variant mode keeps hosted retry behaviour, which differs between adapters (Q24)
- Workers choose their own task through the next-task skill (Q19), unlike `orbit local`'s default where Orbit selects it
- The OpenCode variant has no OS sandbox; only its permission configuration limits it
- The judge sees summaries rather than diffs, so the report is one input alongside the recorded checks

### Impact

`internal/agents/opencode/`, `internal/orbit/variants.go`, sanarte staging branch `wt-0` (`.gitignore`, `.orbit.yaml`, `opencode.json`, guidance file, commit script, ledgers), Orbit runbooks under `docs/runbooks/`, and `docs/agent-notes/local-execution.md`.

---

## Decision 4: Prove Small Execution Units Before the Sanarte Trial

**ID**: 4
**Date**: 2026-10-29
**Status**: accepted

### Context

The coding ledger was absent. The procedure combined two Orbit repairs with
external setup, a long model comparison and owner-gated feature outcomes.
Decision 2's count/next-ID fingerprint could change without any completion,
and Q24 promised no timeout retry despite standard classifier-dependent retries.
`run-termination` still planned the same guard with a different status. Local
execution already has stronger selection/graph/cancellation checks, but flags
copied into a variant alias cannot enable those checks.

### Decision

Keep four coding phases in `tasks.md` and opt-in runtime work in
`tasks-evaluation.md`. Build a dependency-free two-phase fixture first. Verify
direct `orbit local` execution separately from individual aliases and the
parallel comparison, including independent code checks and cancellation.
Only a passing smoke stage unlocks the Sanarte trial.

For variant progress, require a before-incomplete selected ID/title/ancestor identity to be completed
after the session, including descendants. Read the full ledger after execution
instead of fingerprinting a different next phase. No completion or a read error
uses the existing failed-state path, before success accounting. This spec owns
the implementation; `run-termination` must reuse it and replan remaining work.
Keep retry policy unchanged and record actual attempts; 60m is per attempt.
Rune JSON exposes numeric IDs, not stable IDs. Guard identity checks include
numeric ID, title and ancestors; the commit helper validates statuses with JSON
and maps each revision's numbers to stable markers in rune's canonical Markdown.
This portability assumption is verified before implementation, not a request to
change rune's output schema.

A successful comparison smoke needs an actual judge result and recommendation,
not merely generated HTML. Fewer than two successful variants skip judging but
can still produce a partial/status report; record that distinction. Judge failure
can also leave HTML without a recommendation even when the Orbit command exits 0.

Keep original Sanarte IDs with their complete owner acceptance. Narrow code-only
tasks receive new IDs and references to those originals. A code trial cannot
complete browser/widget-policy acceptance. Current work executes approved
ledgers; local task authoring and wider SDD need a separate bounded spec.

### Rationale

The fixture separates orchestration failures from application prerequisites.
Selected-identity completion is directly observable and cannot be satisfied merely
by changing a count or the next task. Accurate task identity preserves the
plan of record. Separate evaluation gates keep coding tasks completable with
normal repository tooling and stop external setup from being silently assumed.

### Alternatives Considered

- **Run Sanarte first and repair blockers as they appear:** Rejected because
  missing caches, browser acceptance and Square capability obscure whether
  the local execution path works at all.
- **Extend `orbit local` into a planner/variant controller now:** Rejected
  because its immutable execution ledger and current evidence do not define
  planning output, review transitions or support for another harness.
- **Retain count-plus-next-ID and a new blocked status:** Rejected for this
  guard because count/identity churn is not completion and a persisted state
  change is unnecessary to stop a reported stall. Broader termination design
  remains separate work, with an explicit reconciliation gate.

### Consequences

**Positive:**
- Small coding phases have deterministic acceptance without running models.
- Direct local execution and alias comparison have separate evidence.
- One owner for the variant guard, with no silent task acceptance downgrade.

**Negative:**
- Fixtures/configuration and an evaluation gate add work before a real trial.
- A session that edits useful code but completes no task stops as stalled.
- Standard variants still lack local graph invariants and process-group cleanup;
  timeout cleanup must be measured before relying on long unattended runs.

---
