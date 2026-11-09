# Bugfix Report: Run-Level summary.json Never Finalized In Variant Mode

**Date:** 2026-08-13
**Status:** Reported (diagnosed, not yet fixed)

## Description of the Issue

In variant mode (`--variants N`), the run-level `.orbit/summary.json` is written
once at construction with `"status": "running"` and is never updated again. Long
after every variant has terminated and the `orbit run` process has exited, the
file still reads:

```json
{
  "started_at": "2026-08-01T17:27:58.710282+10:00",
  "status": "running",
  "phases_completed": 0,
  "total_cost_usd": 0,
  "total_duration_ms": 0,
  "sessions": [],
  "run_number": 5,
  "branch_name": "research"
}
```

Per-variant summaries under `.orbit/logs/variant-N/` *are* finalized correctly.
Only the parent is orphaned.

**Reproduction steps:**
1. Run `orbit run --tasks-file <spec>/tasks.md --variants 2 --parallel`.
2. Let it finish or fail, by any route.
3. Read `<spec>/.orbit/summary.json` — `status` is still `running`,
   `phases_completed` still `0`, `sessions` still `[]`, and `completed_at` absent.

**Impact:** Any consumer that reads run state — a human, a status command, or
another agent inspecting the directory — cannot distinguish a finished run from
a live one. This was the signal that made two dead orchestrations look like they
were still working, delaying diagnosis by hours.

It is also not purely cosmetic. Because the run-level manager is in flat mode by
default (`DateSubdirs` false), the *next* invocation calls
`loadExistingSummary()`, increments `RunNumber`, and resets `Status` to
`"running"` (`internal/logs/manager.go:236-239`). So each invocation inflates
`run_number` while never closing out the previous one — the `run_number: 5`
above is a count of invocations, none of which were ever recorded as ending.

## Investigation Summary

- **Symptoms examined:** Parent `summary.json` pinned at `running` with zeroed
  counters, while sibling `logs/variant-N/summary.json` files carry correct
  terminal states.
- **Code inspected:** `cmd/orbit/run.go` (log-dir resolution), `internal/orbit/orbit.go`
  (`New`, `Run`, `fail`), `internal/orbit/single.go` (`runSingle`, `complete`),
  `internal/orbit/variants.go` (`runWithVariants`, `runVariant`),
  `internal/logs/manager.go` (`NewManagerWithOptions`, `Complete`, `Fail`,
  `loadExistingSummary`), `internal/orbit/registry_bridge.go`.
- **Hypotheses tested:** Confirmed by call-site enumeration that every caller of
  `o.complete()` and `o.fail()` lives in `single.go`, and that `runWithVariants`
  contains no reference to `o.logManager` at all.

## Discovered Root Cause

`New()` (`internal/orbit/orbit.go:264-273`) constructs the run-level
`logs.Manager` unconditionally, for both modes — variant mode is not decided
until `Run()` reaches `orbit.go:416`. Construction immediately writes the file
with `Status: "running"` (`internal/logs/manager.go:220-253`).

Only two functions ever mutate that status:

- `Manager.Complete()` — `internal/logs/manager.go:610-623`
- `Manager.Fail(err)` — `internal/logs/manager.go:832-845`

Their sole call sites are `internal/orbit/single.go:866` (inside `o.complete()`)
and `internal/orbit/orbit.go:502` (inside `o.fail()`). Every caller of
`o.complete()`/`o.fail()` is in `single.go` (lines 750, 755, 776, 785, 791, 799,
805, 833, 847).

`Run()` forks before any of that is reachable:

```go
// internal/orbit/orbit.go:416-425
	if o.variantManager != nil {
		...
		return o.runWithVariants(o.shutdownCtx)
	}
	// Single-run mode (existing behavior)
	return o.runSingle()
```

`runWithVariants` (`internal/orbit/variants.go:176-308`) returns via
`generateReport(...)` or a bare `return nil`, and never touches `o.logManager`.
The run-level manager is therefore written once and orphaned for the life of the
process.

Per-variant state is correct because `runVariant` builds its own manager
(`variants.go:517-526`) and finalizes it at `variants.go:797` (`Complete`) or
`variants.go:574, 601, 693, 742` (`Fail`).

**Defect type:** Missing lifecycle completion on a parallel code path.

**Why it occurred:** Variant mode was added alongside the existing single-run
flow rather than through it. The run-level bookkeeping — summary finalization
and registry status — was left in `runSingle`, and the variant path grew its own
per-variant equivalent without a parent-level counterpart.

**Contributing factors:** The run-level manager is created before the mode is
known, so nothing at construction signals that variant mode will not use it.

## Proposed Resolution

Give `runWithVariants` a terminal-state equivalent: finalize `o.logManager` with
`Complete()` on success and `Fail(err)` on failure, aggregating per-variant
results into `phases_completed`, `total_cost_usd`, and `total_duration_ms` so the
parent summary is a true roll-up rather than zeroes.

Consider also that a run where every variant failed should not report parent
`success`; the aggregation rule needs stating explicitly (all-failed → failed,
mixed → success with per-variant detail, since a variant losing a comparison is
a normal outcome).

**The same gap exists for the registry.** `registerRun()`
(`internal/orbit/registry_bridge.go:17-59`) is likewise only called from
`runSingle` (`single.go:764`); variant mode calls `registerVariantRun` instead.
So the run-level registry entry is never created, and
`updateRunStatus(registry.StatusCompleted/Failed)` is single-run only
(`single.go:863`, `orbit.go:499`). Fix both together or the inconsistency simply
moves.

**Alternatives considered:**
- **Skip creating the run-level manager in variant mode** — Rejected: the file
  is genuinely useful as a roll-up, and its absence would break consumers that
  expect `.orbit/summary.json` to exist. It also cannot be done at the current
  construction point, where the mode is not yet known.
- **Have each variant finalize the parent** — Rejected: concurrent writers to
  one summary under `--parallel`, with an obvious race and no clear owner.

## Affected Files

- `internal/orbit/orbit.go:264-273` — unconditional run-level manager construction
- `internal/orbit/orbit.go:416-425` — the mode fork
- `internal/orbit/orbit.go:499-502` — `o.fail()`, single-run only in practice
- `internal/orbit/single.go:863-866` — `o.complete()`, single-run only
- `internal/orbit/variants.go:176-308` — `runWithVariants`, no finalization
- `internal/logs/manager.go:220-253` — initial write with `Status: "running"`
- `internal/logs/manager.go:236-239` — `RunNumber++` / status reset on reload
- `internal/orbit/registry_bridge.go:17-59` — `registerRun`, single-run only

## Related

- `claude-out-of-usage-credits` — the failure that left two runs in this state
- `variant-phase-loop-no-progress-guard` — the other reason a variant run does
  not reach a terminal state
- `status-display-and-auto-consolidate-race`, `status-ignores-tasks-file` —
  prior defects in run-status reporting
