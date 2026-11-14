# Bugfix Report: Phase Loop Spins Forever On A Human-Gated Task

**Date:** 2026-08-13
**Status:** Reported (diagnosed, not yet fixed)

## Description of the Issue

The variant phase loop re-runs a phase indefinitely when the phase's only
remaining task is human-gated (a `STOP` task that no agent can complete). Each
iteration starts a **fresh agent session**, which re-enters the phase, concludes
correctly that the work is not agent-executable, exits cleanly, and is then
started again. Nothing detects that no progress was made.

Measured on a real run (`ui/glucose-lock-widget`, one `orbit run` invocation):

| Variant | Phase | Sessions | Mean per session | Mean cost | Total |
|---|---|---|---|---|---|
| 1 (fable) | 4 | **1223** | 32.7 s | $0.220 | $269.45 |
| 2 (opus) | 4 | **1394** | 28.3 s | $0.359 | $499.78 |
| 1 (fable) | 5 | 7 | 1373.7 s | $11.356 | $79.49 |
| 2 (opus) | 5 | 12 | 1453.9 s | $10.568 | $126.82 |

Phase 4's sole open task was `14. Human-gated on-device verification (STOP)`.
Combined across both variants and all phases the run recorded **$1089.53**, of
which roughly **$830 was spent looping on `STOP` tasks that no agent could ever
complete**. This is the direct cause of the account hitting its usage limit,
which in turn triggered `claude-out-of-usage-credits`.

**The loop has two distinct cost profiles, and both matter.** In phase 4 the
agent recognised the gate quickly and exited in about 30 seconds, so each
iteration was individually cheap — the damage is purely volumetric, 2617
iterations at $0.22–$0.36. In phase 5 each iteration ran the repository's full
verification, and the measured `make test` for that repo is 19 m 51 s wall
(3.4 CPU-hours), which accounts for essentially all of the 23–24 minute mean
session time. So a stalled phase costs either a lot of cheap iterations or a few
very expensive ones depending on what the phase does, and an iteration ceiling
tuned for one profile is wrong for the other. This is the argument for making
progress detection the primary guard rather than a count or a budget.

**Reproduction steps:**
1. Author a tasks file whose next phase contains only a task that an agent
   cannot complete (a human-gated `STOP`, or any task the agent will decline).
2. Run `orbit run --tasks-file <spec>/tasks.md --variants 2 --parallel`.
3. Observe the same phase restarting with a new session ID, indefinitely, each
   iteration re-running the phase's verification commands.

**Impact:** Unbounded spend and unbounded wall-clock time with zero forward
progress, ending only when the account's usage limit is exhausted. On a repo
whose `make test` takes tens of minutes, each wasted iteration is also tens of
minutes of saturated CPU.

**This defeats the known spec-side workaround.** The consuming project had
already met this loop once (a non-`STOP` task gated on data only real-world use
could produce: 45 re-entries, 13 hours, ~$191, 2026-08-12) and mitigated it by
rewriting such tasks as human-gated `STOP` tasks, on the stated contract that
"`STOP` tasks are the only ones no agent can execute". The measurements above
show that contract does not hold on orbit's side: phase 4's open task *was* a
`STOP`, and it looped 1223 and 1394 times. Marking a task human-gated changes
nothing, because nothing in the phase loop reads task gating — see
`internal/rune/client.go:202-219`, which considers only `status`. Any fix must
therefore live in orbit; no amount of task authoring can avoid this.

## Investigation Summary

- **Symptoms examined:** Same phase restarting for hours; a surviving variant
  never terminating; `.orbit/logs/variant-N/summary.json` `sessions[]` holding
  1223 and 1394 entries all tagged with the same phase and `run_number`.
- **Code inspected:** `internal/orbit/variants.go:615-719` (the phase loop),
  `internal/orbit/single.go:771-837` (the single-run loop),
  `internal/rune/client.go:202-219` (`AllComplete` determination),
  `internal/logs/manager.go:325, 365-368, 506-513, 563-564` (session minting and
  file naming), and a repo-wide search for any halt/stop-file convention.
- **Hypotheses tested:** Confirmed the loop has exactly three exits and none of
  them can be reached by a phase that runs cleanly without completing a task.
  Confirmed no halt sentinel is read anywhere in the codebase.

## Discovered Root Cause

**The loop has no progress check and no iteration cap.**
`internal/orbit/variants.go:615-719`:

```go
	// === Step 3: Phase Loop ===
	for {
		select { case <-ctx.Done(): ...; return ctx.Err(); default: }

		nextPhase, err := variantRuneClient.GetNextPhase()      // line 629
		if err != nil { ...StatusFailed...; return variantErr }

		if nextPhase.AllComplete {                              // line 638
			break
		}
		...
		phaseResult, err := o.runVariantPhaseWithRetry(...)     // line 681
		if err != nil { ...StatusFailed...; return variantErr } // line 682-700
	}
```

The only exits are context cancellation, `rune next` erroring, and
`AllComplete`. A phase that runs successfully but completes no task hits none of
them. There is no counter bounding the outer `for {}` — nothing analogous to the
`maxRetries = 5` that guards the inner retry — and no comparison of the pending
task count before versus after an iteration.

**`AllComplete` cannot become true.** `internal/rune/client.go:202-219` derives
it purely from task status:

```go
		for _, task := range result.Tasks {
			if task.Status != "completed" && task.Status != "done" {
				allDone = false
				break
			}
		}
```

A human-gated `STOP` task is never `completed`, so `rune next` returns the same
phase forever.

**A clean exit is indistinguishable from progress.** The agent's only channel
back to orbit is its exit code and JSON. An agent that correctly determines
"this task needs a human" and exits 0 is read as a successful phase, and the
loop simply asks for the next phase — which is the same phase.

**No halt convention exists.** A repo-wide case-insensitive search for `halt`
across `*.go`, `*.md`, `docs/`, `specs/`, and `samples/` returns only an
unrelated spinner comment (`internal/display/spinner.go:173`) and a substring
match in `internal/registry/git.go:19`. There is no `os.Stat`/`ReadFile` of any
sentinel. In the observed run the agent did write its reason to a file, at
`<worktree>/.orbit/halt` — but orbit's run-state directory is
`filepath.Dir(tasksFile)/.orbit` (`cmd/orbit/run.go:136`), whereas the worktree
is `<spec>/.orbit/worktrees/<branch-dir>/` (`internal/variants/manager.go:57, 267`).
The file therefore landed at
`<spec>/.orbit/worktrees/<branch-dir>/.orbit/halt`, a path orbit never
enumerates. The agent was trying to say "stop, a human is needed" and had no
mechanism that could carry it.

**Defect type:** Missing loop-termination guard (no progress detection, no cap).

**Why it occurred:** The loop's termination condition assumes that a
successfully-executed phase always advances task state. That holds whenever
every task is agent-executable, which is the case the design anticipated. Tasks
that deliberately require human action violate the assumption, and the task
schema has no way to express "not agent-executable" that the loop consults.

**Contributing factors:** The per-iteration cost is invisible from the log
filenames, so the runaway is easy to miss — see below.

### The `-run-N` suffix is not the retry counter

`phaseFileName` (`internal/logs/manager.go:506-513`) suffixes `-run-N` from
`m.summary.RunNumber`, which is incremented once per `logs.Manager` construction
(`manager.go:230-239`) — i.e. **once per `orbit run` invocation**, not per loop
iteration. Within one process the variant manager is built once
(`variants.go:519`) and `RunNumber` is frozen, so all 1223 iterations overwrote
the *same* `phase-4-session.json` via `SaveSession` (`manager.go:585-603`).
Likewise `m.summary.PhasesCompleted = phase` (`manager.go:564`) is assigned, not
incremented, so it stays pinned.

The only durable fingerprint of the loop is the `sessions[]` array in
`.orbit/logs/variant-N/summary.json`: many entries sharing one `phase` and
`run_number` but each with a distinct `session_id` (a fresh
`uuid.NewString()` minted at `manager.go:325` on every `StartPhase`, because
`CompletePhase()` cleared `CurrentPhase` on the prior iteration at
`manager.go:365-368`).

## Proposed Resolution

Three complementary guards; the first alone stops the bleeding.

1. **Stall detection.** Capture the pending-task count (or the next-task
   identity) before running a phase and compare after. If an iteration completes
   with no change, stop the variant and report why — the phase is not
   agent-executable. This is the smallest change that converts an infinite spend
   into a clear terminal state.

2. **A hard iteration cap** on the outer `for {}`, mirroring `maxRetries`, as a
   backstop for any stall the progress check does not model.

3. **An agent-visible halt sentinel that orbit actually stats.** Define it
   against orbit's `LogDir` (or the variant's own log directory), not the
   worktree, and document the path — the observed near-miss shows agents will
   reach for this convention whether or not it exists. Read the file's contents
   into the recorded stop reason so the human gate is reported rather than
   inferred.

Consider also surfacing task-level gating in `internal/rune/client.go` so a
`STOP` task can be recognised as not-agent-executable before a phase is
launched, rather than after an expensive round trip.

**The single-run loop needs the same fix.** `internal/orbit/single.go:771-837`
has the identical shape. Its extra `ListPending()` guard at `:782-792` does not
help, because it also never compares counts across iterations.

**Alternatives considered:**
- **Rely on the agent to exit non-zero when it cannot proceed** — Rejected:
  requires every agent across five supported CLIs to adopt a convention orbit
  does not define, and conflates "cannot proceed" with "failed", which would
  mark legitimately-blocked runs as failures.
- **Cap by wall-clock or cost instead of iterations** — Rejected as the primary
  guard: it stops the bleeding without diagnosing the cause, and the right
  outcome here is a clear "human gate reached" verdict, not a timeout. Useful as
  an additional backstop.

## Affected Files

- `internal/orbit/variants.go:615-719` — the unbounded variant phase loop
- `internal/orbit/single.go:771-837` — same shape in single-run mode
- `internal/rune/client.go:202-219` — `AllComplete` derivation
- `internal/logs/manager.go:325, 365-368` — fresh session ID per iteration
- `internal/logs/manager.go:506-513` — `-run-N` naming (invocation, not iteration)
- `internal/logs/manager.go:563-564` — `sessions[]` append; `PhasesCompleted` assign
- `cmd/orbit/run.go:136` vs `internal/variants/manager.go:57, 267` — the
  run-state / worktree path split that made the agent's halt file invisible

## Related

- `claude-out-of-usage-credits` — the usage exhaustion this loop caused
- `variant-run-summary-never-finalized` — why the looping run also never
  recorded a terminal state
- `variant-phase-retry-nil-runresult` — prior defect in the same retry path
