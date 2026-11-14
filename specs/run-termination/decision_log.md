# Decision Log: Run Termination Guarantees

## Decision 1: Stall detection is the primary termination guard

**Date**: 2026-08-13
**Status**: proposed

### Context

A single `orbit run` invocation re-ran one phase 1223 and 1394 times across two
variants, spending roughly $830 with no forward progress, because the phase's
only open task was human-gated. The phase loop
(`internal/orbit/variants.go:615-719`) has exactly three exits — context
cancellation, `rune next` erroring, and `AllComplete` — and a phase that executes
cleanly without completing a task reaches none of them.

Several mechanisms could bound the loop. The choice determines whether orbit
stops for a diagnosable reason or merely stops.

### Decision

Make a per-iteration progress comparison the primary guard: fingerprint the
outstanding work before a phase and after it, and stop the variant when a
successful iteration leaves the fingerprint unchanged. Cost, wall-clock, and
iteration ceilings are backstops layered on top, not the primary mechanism.

### Rationale

The progress check is the only candidate that identifies the actual condition —
"this phase cannot advance" — at the moment it becomes true, on the first wasted
iteration rather than the thousandth. It is also cheap: the fingerprint is
computed from the `GetNextPhase` result the loop already fetches, so it adds no
agent invocations and no I/O.

Crucially it produces a *diagnosis* rather than a symptom. A budget cap would
have stopped this run too, but it would have reported "budget exceeded", leaving
the user to work out why 1223 sessions ran. The progress check can name the phase
and the blocking task.

### Alternatives Considered

- **Cost or wall-clock budget cap as the primary guard**: Bound total spend or
  runtime per run - Rejected: stops the bleeding without diagnosing the cause,
  and any threshold is either loose enough to waste substantial money or tight
  enough to kill legitimate long phases. Retained as an optional backstop.
- **Iteration ceiling alone**: A simple counter on the loop - Rejected as the
  primary guard: to be safe for legitimate multi-phase runs the ceiling must sit
  far above normal usage, so a stall still burns a large amount before tripping
  it. Retained as a backstop for stalls the fingerprint fails to model.
- **Require agents to exit non-zero when blocked**: Let the agent signal failure
  - Rejected: depends on a convention across five supported agent CLIs that
  orbit does not define, and conflates "blocked" with "failed", which is exactly
  the misreporting this feature exists to remove.

### Consequences

**Positive:**
- A stalled run stops after roughly one wasted iteration instead of thousands.
- The stop carries an actionable reason naming the phase and blocking task.
- No additional agent invocations; the fingerprint reuses data already fetched.

**Negative:**
- The fingerprint must be chosen carefully: a phase that completes one task while
  adding another progresses genuinely but could hold a naive count steady, which
  is why task identity is compared alongside the count.
- A phase that legitimately does agent-visible work without changing task state
  (for example, a pure investigation phase) would now stop. No such phase exists
  in current usage, but the behaviour is a change in contract.

---

## Decision 2: A blocked run is a distinct terminal state, not a failure

**Date**: 2026-08-13
**Status**: proposed

### Context

When the phase loop stops because work is human-gated, the natural implementation
is to reuse the existing `StatusFailed` path that every other loop exit uses. But
orbit already has a track record of this being wrong: after an earlier run, the
consuming project recorded in its own notes that "Orbit recorded both variants as
`failed` — that status is spurious", and had to annotate the real cause by hand.

The same conflation appears in the credit-exhaustion path, where a billing
condition surfaces as a generic phase failure.

### Decision

Introduce a terminal state distinct from both success and failure, meaning
"stopped, and the remaining work needs a human". Use it for stall detection and
for the halt sentinel, and propagate it through the variant status enum, the
`variants.json` metadata, and the status display.

### Rationale

The distinction is operational, not cosmetic. A failure means "something is
broken, investigate the code". Blocked means "the machine did its part, now do
yours". Presenting the second as the first sends the user to debug code that
works, which is precisely what happened and cost hours of diagnosis.

It also protects the fix in Decision 1 from being unusable: if stopping on a
stall reported `failed`, users would reasonably treat the new guard as a source
of false alarms.

### Alternatives Considered

- **Reuse `StatusFailed` with a distinguishing error message**: Smallest change -
  Rejected: status is what tooling and humans read first; burying the distinction
  in message text repeats the mistake that required the hand-annotation.
- **Reuse `StatusCompleted`**: A blocked run did finish everything it could -
  Rejected: it would report success while tasks remain incomplete, and would make
  a genuinely finished run indistinguishable from a stalled one.

### Consequences

**Positive:**
- "Needs a human" is visible at a glance in `orbit status` and `variants.json`.
- Removes the recurring need to annotate spurious `failed` statuses by hand.

**Negative:**
- A new enum value touches the status display, the variant manager, and any
  consumer that switches exhaustively on status — including, potentially,
  downstream tooling.
- Adds a third outcome that documentation and users must learn.

---

## Decision 3: Classify credit exhaustion from structured fields, with message text as refinement

**Date**: 2026-08-13
**Status**: proposed

### Context

`Classifier.Classify` (`internal/agents/claudecode/errors.go:27-77`) detects
usage exhaustion by `strings.Contains` against literal phrases. That approach has
now failed three times as Anthropic reworded the message: `usage-limit-no-timezone`
(2026-03-05), `claude-session-limit-message` (2026-07-22), and the present
`You're out of usage credits...` variant. Each previous fix added one more
literal.

The agent's JSON result already carries `terminal_reason: "api_error"`,
`api_error_status: 429`, and `is_error: true`. Neither `api_error_status` nor
`terminal_reason` appears anywhere in orbit's Go source.

### Decision

Classify from the structured fields as the primary signal, retaining literal
matching as a fallback for agent versions that do not emit them. Where the
structured signal is ambiguous, use the message text to refine it rather than to
detect it.

### Rationale

The structured fields are a machine interface and change far less often than
user-facing prose; keying off them is what stops this defect recurring on the
next rewording. Prose matching remains necessary as a fallback because orbit
supports agent versions that predate these fields, and removing it would trade a
recurring bug for an immediate regression.

Refinement is needed because `api_error_status: 429` alone cannot separate a
genuine transient rate limit from credit exhaustion — both are 429, but one is
retryable and one is not. The pair of signals answers what neither does alone.

### Alternatives Considered

- **Add `"usage credits"` to `commonRateLimitPatterns`**: One-line fix -
  Rejected: yields `ErrorClassRetryable` with a 60s delay, wasting five attempts
  on a condition that cannot clear on its own, and perpetuates prose matching.
- **Add a fourth literal to the existing usage-limit guard**: Consistent with the
  two prior fixes - Rejected: the guard also requires a parseable reset time
  (`errors.go:39-48`), which this message does not have, so the branch still
  would not fire — and it guarantees a fourth recurrence.
- **Match on `api_error_status` alone**: Purely structural - Rejected: cannot
  distinguish credit exhaustion from a transient 429, and would make genuine rate
  limits non-retryable.

### Consequences

**Positive:**
- Ends a defect that has recurred three times on message rewording.
- Uses data orbit already captures and writes to disk.

**Negative:**
- Two detection paths must be maintained and kept consistent.
- The structured fields are themselves an undocumented interface of the Claude
  CLI and could change, though less readily than displayed prose.

---

## Decision 4: Credit exhaustion gets a new error class rather than reusing an existing one

**Date**: 2026-08-13
**Status**: proposed

### Context

Once classified, credit exhaustion must map to an `ErrorClass`. The existing
values are `ErrorClassRetryable`, `ErrorClassRateLimitWait`, `ErrorClassFatal`,
and `ErrorClassUnknown`. Orbit is a fork with an upstream
(`ArjenSchwarz/orbit`), so widening a shared enum has costs beyond this repo.

### Decision

Add a class for "stopped, human action required to continue" — not retried, not
waited on, but carrying an attributable, actionable message rather than a generic
fatal error.

### Rationale

Neither retry semantics fit. `ErrorClassRetryable` would spend five attempts over
about thirty seconds against a condition only a human can clear.
`ErrorClassRateLimitWait` requires a reset time to sleep until, and credit
exhaustion has none — the existing branch would silently fall through even if the
phrase matched.

`ErrorClassFatal` is behaviourally correct but semantically misleading in the
same way Decision 2 describes: it reports a billing state as a broken run. Since
Decision 2 already introduces a blocked outcome at the run level, aligning the
error class with it keeps one vocabulary rather than two.

### Alternatives Considered

- **Reuse `ErrorClassFatal`**: No enum change, correct stop behaviour - Rejected:
  reports "out of credits" as a fatal error, so the user still cannot tell a
  billing stop from a broken run without reading the message. Remains the
  fallback if the upstream-divergence cost is judged too high.
- **Reuse `ErrorClassRateLimitWait` with a synthetic wait**: Reuses the existing
  wait-and-resume machinery - Rejected: there is nothing to wait for; the run
  would sleep and then fail again, wasting time and obscuring the cause.

### Consequences

**Positive:**
- The run stops promptly with a message naming the cause and the remedies.
- One vocabulary shared with the run-level blocked state from Decision 2.

**Negative:**
- Widens a shared enum in `internal/agents`, touching every agent's switch
  statements and increasing divergence from upstream — the main cost of this
  decision, and the reason `ErrorClassFatal` is retained as a fallback.

---

## Decision 5: The halt sentinel lives in an orbit-owned directory, not the worktree

**Date**: 2026-08-13
**Status**: proposed

### Context

In the observed run the agent wrote its halt reason to `<worktree>/.orbit/halt`,
plainly expecting orbit to read it. Orbit has no halt convention at all — a
repo-wide search finds no sentinel handling — so the file was inert. The path it
chose was also unreachable: orbit's run-state directory is
`filepath.Dir(tasksFile)/.orbit` (`cmd/orbit/run.go:136`), whereas the worktree
sits at `<spec>/.orbit/worktrees/<branch-dir>/`
(`internal/variants/manager.go:57, 267`), so a worktree-relative `.orbit/halt`
lands in a directory orbit never enumerates.

### Decision

Define the sentinel against a directory orbit owns and already enumerates — the
variant's own log directory — and document the path explicitly for agent authors.
Check it after each phase, read its contents as the stop reason, and clear it at
variant start so a stale file cannot halt a later run.

### Rationale

Orbit must be able to find the file without guessing, and the log directory is
already per-variant, already created by orbit, and already enumerated. Placing it
there also keeps run state out of the worktree, so the sentinel cannot be
accidentally committed to the variant branch or confuse a consolidation diff.

The convention needs documenting precisely because an agent already invented it
unprompted: agents will reach for a halt file whether or not one exists, so the
choice is between a defined path and an undefined one.

### Alternatives Considered

- **Worktree root (`<worktree>/.orbit/halt`)**: Where the agent's working
  directory actually is, so it is the path an agent reaches for naturally -
  Rejected: puts run state inside the git worktree, where it can be committed to
  the variant branch or pollute a consolidation diff. Worth supporting as an
  additional accepted location if agents keep writing there in practice.
- **No sentinel; rely solely on stall detection**: Simplest - Rejected: stall
  detection infers the reason, while the sentinel lets the agent state it. The
  agent knows why it stopped; discarding that produces a worse message. Note the
  sentinel remains optional — requirement 1 stops a stalled run regardless.

### Consequences

**Positive:**
- Gives agents a defined way to say "blocked on a human" with a reason in their
  own words.
- Keeps run state out of the worktree and therefore out of variant diffs.

**Negative:**
- The natural path for an agent is the worktree, so the documented path must be
  communicated well or agents will keep writing an inert file.
- Requires lifecycle care: a stale sentinel must be cleared at variant start or
  it will halt an unrelated later run.

---
