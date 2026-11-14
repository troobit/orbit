# Design: Run Termination Guarantees

## Overview

Three independent defects combine to let a run consume an account's entire
credit balance while reporting nothing useful. They are fixed in three places
that do not overlap:

1. **`internal/orbit/variants.go` / `internal/orbit/single.go`** — the phase loop
   gains progress detection, an iteration ceiling, and a halt-sentinel check.
2. **`internal/agents/claudecode/`** — the error classifier learns credit
   exhaustion from structured fields, and stops discarding the agent's own
   diagnostic text.
3. **`internal/orbit/variants.go` (run level)** — variant mode gains the
   terminal-state finalization that only `runSingle` performs today.

Fix 1 is the one that stops the money burn; the other two make the resulting
stop legible. They are independently landable and independently valuable.

---

## 1. Phase Loop Termination

### Current structure

`internal/orbit/variants.go:615-719`:

```go
	for {
		select { case <-ctx.Done(): ...; return ctx.Err(); default: }

		nextPhase, err := variantRuneClient.GetNextPhase()
		if err != nil { ...StatusFailed...; return variantErr }

		if nextPhase.AllComplete { break }
		...
		phaseResult, err := o.runVariantPhaseWithRetry(...)
		if err != nil { ...StatusFailed...; return variantErr }
	}
```

Three exits: context cancellation, `rune next` error, `AllComplete`. A phase that
runs cleanly without completing a task reaches none of them.
`AllComplete` is derived purely from task `status`
(`internal/rune/client.go:202-219`), and a human-gated task is never
`completed`, so it can never become true.

### Progress measure

Before executing a phase, capture a **progress fingerprint**: the count of
incomplete tasks plus the identity of the next task, both already available from
the `GetNextPhase` result. After the phase returns successfully, recompute it. An
unchanged fingerprint means the iteration did nothing.

The identity component matters as well as the count — a phase that completes one
task and adds another would hold the count steady while genuinely progressing.
Comparing the next-task identity as well makes that case distinguishable.

### New terminal state

A stalled variant is **not** a failure. Reporting it as `StatusFailed` is what
led the consuming project to hand-annotate "that status is spurious" in its own
notes after an earlier incident. The design adds a distinct blocked state so
that `orbit status` and the variant metadata can say "phase X is waiting on a
human" rather than "phase X failed".

This propagates to `internal/variants/manager.go` (the status enum written to
`variants.json`) and to the status display.

### Iteration ceiling

A counter on the loop, defaulting high enough that no legitimate run approaches
it (the observed pathological runs reached 1223 and 1394; a legitimate run
executes one iteration per phase). It exists so that a stall whose shape the
fingerprint fails to capture is still bounded. Reaching it is recorded distinctly
from a fingerprint stall, because reaching it means the fingerprint missed
something and that is a defect worth seeing.

### Halt sentinel

Agents already reach for this convention unprompted: in the observed run the
agent wrote its halt reason to `<worktree>/.orbit/halt`, which orbit never reads.
The path split is the trap — orbit's run-state directory is
`filepath.Dir(tasksFile)/.orbit` (`cmd/orbit/run.go:136`) while the worktree is
`<spec>/.orbit/worktrees/<branch-dir>/` (`internal/variants/manager.go:57, 267`),
so a worktree-relative `.orbit/halt` lands somewhere orbit never enumerates.

Two candidate locations, resolved in the decision log: the variant's own log
directory (orbit-owned, per-variant, already enumerated) versus the worktree root
(where the agent's working directory actually is, and therefore where an agent
will naturally write). The sentinel must be checked after each phase and cleared
at variant start so a stale file cannot halt the next run.

---

## 2. Error Classification

### Current path

`Classifier.Classify` (`internal/agents/claudecode/errors.go:27-77`) matches
literal substrings against `stderr + stdout + errMsgs`. Credit exhaustion emits:

```
You're out of usage credits. Run /usage-credits to keep using Fable 5 or /model to switch models.
```

which matches none of the twenty substrings across the six pattern sets, so it
reaches `NewUnknownError` at `errors.go:76`. `ErrorClass.IsRetryable()` returns
false for unknown, so `RunWithRetry` (`internal/agents/retry.go:85-87`) returns
immediately.

Even a matching phrase would not help: the branch at `errors.go:39-48` requires
`parseUsageLimitReset` to return a positive duration, and credit exhaustion
carries no reset time.

### Structured classification

The agent's own JSON result already contains the answer:

```json
{
  "is_error": true,
  "terminal_reason": "api_error",
  "api_error_status": 429,
  "result": "You're out of usage credits. ..."
}
```

`api_error_status` and `terminal_reason` appear nowhere in orbit's Go source.
Parsing them gives a classification signal that does not change when Anthropic
rewrites the sentence — which it has now done three times
(`usage-limit-no-timezone`, `claude-session-limit-message`, and this one).

Note that `api_error_status: 429` alone is ambiguous: a genuine rate limit is
also 429. The discriminator is the pair — a rate limit is retryable and
transient, credit exhaustion is neither — so the message text is still needed to
separate them, but as a *refinement* of a structured signal rather than as the
sole detector. Literal matching remains as a fallback for agent versions that do
not emit these fields.

### A new error class

Credit exhaustion fits neither existing class:

| Class | Behaviour | Why it is wrong here |
|---|---|---|
| `ErrorClassRetryable` | ~5 attempts, 60s apart | Burns attempts on a condition that cannot clear on its own |
| `ErrorClassRateLimitWait` | Sleep until reset, resume | There is no reset time to parse |
| `ErrorClassUnknown` | Stop, generic message | Current behaviour; loses the cause |

The new class stops promptly like a fatal error, but carries an attributable,
actionable message ("out of credits — add credits or run with `--agent opus`").
Adding a class touches the shared `agents` package and therefore every agent
implementation's switch statements; this is called out as a cost in the decision
log, and matters because orbit is a fork with an upstream.

### Preserving diagnostic text

Independently of credits, two layers currently discard agent error text:

- `processExecResult` (`internal/agents/claudecode/agent.go:303-309`) returns the
  raw `*exec.ExitError` whenever `execResult.Err != nil`, overriding the
  `ClassifiedError` built immediately above it.
- `classifyFromAgent` (`internal/orbit/single.go:71-73`) synthesises
  `errMsgs = []string{"exit status 1"}`, and `BuildUnknownMessage`
  (`internal/agents/errors.go:237-249`) prefers non-empty `errMsgs` over
  `stderr`.

The net effect is that *any* classified failure that also exits non-zero loses
its message. Fixing the precedence benefits every agent and every failure mode,
not just this one.

---

## 3. Run-Level Finalization

`New()` (`internal/orbit/orbit.go:264-273`) builds the run-level `logs.Manager`
unconditionally, before the mode is known — variant mode is not decided until
`Run()` reaches `orbit.go:416`. Construction writes `Status: "running"`
immediately (`internal/logs/manager.go:220-253`).

The only mutators are `Manager.Complete()` (`manager.go:610-623`) and
`Manager.Fail(err)` (`manager.go:832-845`), reachable only via `o.complete()` /
`o.fail()`, every caller of which is in `single.go`. `runWithVariants`
(`variants.go:176-308`) contains no reference to `o.logManager` at all.

The fix gives `runWithVariants` a terminal path that finalizes the run-level
manager and aggregates per-variant totals. The aggregation rule needs stating
because a variant losing a comparison is a normal outcome, not a run failure:

- every variant failed → run failed
- at least one variant completed → run succeeded, with per-variant detail
- interrupted → recorded as such, distinct from both

The registry has the identical gap: `registerRun` (`registry_bridge.go:17-59`)
and `updateRunStatus` are single-run only, with variant mode using
`registerVariantRun` instead. Fixing only the summary would move the
inconsistency rather than remove it, so both are in scope.

A stale `running` summary is also actively harmful rather than merely untidy:
`loadExistingSummary` increments `RunNumber` and resets `Status` to `"running"`
on the next invocation (`manager.go:236-239`), so each run inflates the counter
while never closing out the previous one.

---

## 4. Observability

Within one invocation `RunNumber` is frozen, so `phaseFileName`
(`manager.go:506-513`) produced the same filename for all 1223 iterations and
each `SaveSession` overwrote the last. The only surviving evidence was the
`sessions[]` array in the per-variant summary — many entries sharing one `phase`
and `run_number`, each with a distinct `session_id` minted at `manager.go:325`.

Repeated execution should be visible without that reconstruction: a warning once
a phase exceeds a small number of executions in one invocation, and documentation
correcting the natural misreading that `-run-N` counts retries.

---

## Testing Strategy

The existing mock-agent framework (`specs/integration-test-framework`) covers
this without invoking real CLIs:

- A mock agent that exits 0 without completing any task drives the stall path;
  the test asserts the loop stops after a bounded number of iterations and
  records the blocked state rather than a failure.
- A mock agent emitting the credit-exhaustion JSON asserts the new class, the
  absence of retries, and that the recorded message contains the cause.
- Classifier table tests cover the three historical message variants alongside
  the structured-field path, guarding the regression this feature exists to stop.
- A variant run driven to each terminal route asserts the run-level
  `summary.json` is closed out with aggregated totals.
