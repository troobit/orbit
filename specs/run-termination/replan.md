# Replan: Run Termination Guarantees

**Date**: 2026-11-06
**Status**: proposed (awaiting approval; this is the content of tasks.md task 1, the STOP gate)

Reconciles this ledger with [local-run-comparison Decision 4](../local-run-comparison/decision_log.md#decision-4-prove-small-execution-units-before-the-sanarte-trial) and this spec's Decision 6.

## What the code looks like today (branch `local-compute`, 7477fac)

| Area | State |
|------|-------|
| Variant no-progress guard | Not landed. Owned by local-run-comparison task 2 (RED 2.1, GREEN 2.2), which is blocked by its task 1 (OpenCode typed streams). |
| Single-run no-progress guard | Absent (`internal/orbit/single.go` phase loop compares nothing across iterations). |
| Iteration ceiling | Absent in both loops. |
| Halt sentinel | `orbit local` reads `<WorkingDir>/.orbit/halt` (`internal/orbit/local.go:37`) and `ORBIT-HALT:` output lines. Variant and single-run loops do not. Requirements 3.x and Decision 5 (orbit-owned directory) were written when nothing read it. |
| Error classification | Codex usage limits are classified and waited on (merged from `fix/codex-usage-limit-wait`; `ErrorClassRateLimitWait`, `internal/agents/codex/errors.go`). Claude credit exhaustion is not: `terminal_reason` and `api_error_status` are not parsed (`is_error` only), and no human-action class exists. |
| Diagnostic text | Unchanged: `processExecResult` still returns the raw `ExitError`. |
| Run-level finalization | Unchanged: variant mode never finalizes `summary.json` or the run registry entry. |

## Proposed changes

| Tasks | Disposition | Reason |
|-------|-------------|--------|
| 1 | Completes when this replan is approved. | Gate. |
| 2, 3, 4, 7 | **Delegated, not built here.** Mark each "delivered by local-run-comparison task 2" once it lands, with a one-line verification that the guard's error names the phase, next task and "stalled". | Decision 6. |
| 5 | Keep, rewritten: reuse the selected-identity helper from local-run-comparison 2.2 in the `single.go` loop; stalled run ends through the existing failure path. | Single-run mode keeps the defect after the variant guard lands. |
| 6 | Keep. Ceiling counts iterations per phase per invocation; exhaustion recorded as "ceiling reached" (distinct from "stalled"). | Backstop when identity tracking misses something. |
| 8, 9, 10 (halt sentinel) | **Defer, drop from this ledger.** | local-run-comparison Decision 2 chose not to read the sentinel in variants because progress detection already stops halted sessions; adding a reader only adds a reason string. `orbit local` keeps its own check. Revisit if a stalled-reason string proves insufficient. |
| 11, 12, 13, 14, 15, 16 | Keep. Task 13 narrowed to Claude credit exhaustion; task 15's regression table includes the existing codex wordings as already-passing cases. Task 12 keeps the `ErrorClassFatal` fallback (Decision 4). | Credit exhaustion is the second cause of the August $1089 run. |
| 17–22 (finalization) | Keep unchanged. | Nothing has landed. |
| 23, 24, 25 | Keep unchanged. | Observability, independent of the above. |

No `blocked` status is added (Decision 2 superseded in effect by Decision 6; a stalled run reports `failed` with "stalled" in the error). A Decision 7 recording the sentinel deferral and the blocked-state retirement is added on approval.

## Sequencing (the part that changes lane B)

The compute-saving guard is task 2 of local-run-comparison, which today waits behind its task 1 (OpenCode event streams, a large unrelated adapter). Its tasks use testutil agents and temporary real-rune ledgers, so the dependency looks like ordering, not a technical need.

Proposed order for lane B:

1. Remove `Blocked-by` on local-run-comparison task 2 (task 2 does not need typed OpenCode streams). Run `orbit run --tasks-file specs/local-run-comparison/tasks.md` for task 2 only (`--phase` stops after it).
2. Run this ledger: 5, 6, 11–25 (tasks 2–4, 7 marked delivered).
3. Continue the plan: rebase onto apsis, rescope local-task-preparation and local-run-comparison.

If task 1 (OpenCode streams) is needed first for a reason I did not see, say so and step 1 is dropped; the remaining order stands, but unattended orbit runs stay unprotected against the stall loop until local-run-comparison task 2 lands.

## What I need from you

1. **Approve or amend** the dispositions above (especially deferring the sentinel, tasks 8–10).
2. **Sequencing**: lift the `Blocked-by` on local-run-comparison task 2 and run it first? (recommended)
3. **Interim safety**: until the guard lands, run orbit only on ledgers with no `STOP:` task (the screw and apsis ledgers qualify; run-termination itself does not until task 1 is completed).

On approval I mark task 1 complete, mark tasks 2–4 and 7 delegated, remove 8–10, write Decision 7, and start step 1.
