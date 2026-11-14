# Bugfix Report: Claude Out-Of-Usage-Credits Is Unclassified

**Date:** 2026-08-13
**Status:** Reported (diagnosed, not yet fixed)

## Description of the Issue

When the Claude Code CLI exhausts prepaid usage credits it prints

```
You're out of usage credits. Run /usage-credits to keep using Fable 5 or /model to switch models.
```

and exits non-zero. Orbit does not recognise this as a usage-limit condition. The
error falls through to `ErrorClassUnknown`, which is not retryable, so the variant
is marked `failed` immediately and the recorded error is the bare
`phase Phase 4: Backbone Swap Spike (gated half + comparison): exit status 1`.
The CLI's own diagnostic text never reaches the persisted error.

This is the third wording variant of the same defect. See
`claude-session-limit-message` (2026-07-22) and `usage-limit-no-timezone`
(2026-03-05), both fixed by adding another literal phrase.

**Reproduction steps:**
1. Exhaust the usage credits on a Claude Code account.
2. Run `orbit run --variants 2 --parallel --variant-agents fable,opus`.
3. Observe the fable variant recorded as `"status": "failed"` with
   `"error": "phase <name>: exit status 1"` in `.orbit/variants.json`.
4. Observe that no retry and no wait were attempted, despite `maxRetries = 5`.

**Impact:** An in-progress orchestration is lost on a condition that is
diagnosable and reportable. Because the recorded error is a generic
`exit status 1`, the cause is indistinguishable from a genuine phase failure —
so a run that quit on billing looks identical to a run that quit on broken code.
Observed on three separate specs: `estimation/segmenter-foundation` and
`ui/glucose-lock-widget` (both 2026-08-13), and previously on
`estimation/support-plane-reference` (2026-08-05/06), where the consuming
project had to annotate the spurious `failed` status by hand in its own notes.

## Investigation Summary

- **Symptoms examined:** Variant marked `failed` with a generic exit-status
  message; no retry attempted; the CLI's credit-exhaustion text present in the
  saved session JSON but absent from the recorded error.
- **Code inspected:** `internal/agents/claudecode/errors.go` (`Classify`,
  `parseUsageLimitReset`), `internal/agents/errors.go` (the shared pattern sets
  and `BuildUnknownMessage`), `internal/agents/claudecode/agent.go`
  (`processExecResult`), `internal/agents/retry.go` (`RunWithRetry`),
  `internal/orbit/single.go` (`classifyFromAgent`), `internal/orbit/variants.go`
  (phase-loop error wrapping).
- **Hypotheses tested:** Confirmed the message matches none of the twenty
  literal substrings across all six pattern sets. Confirmed that even if the
  usage-limit branch were entered, it would still not fire, because the credits
  message carries no `resets <time>` clause and `parseUsageLimitReset` returns 0.

## Discovered Root Cause

Three independent layers compound.

**Layer 1 — the phrase guard.** `Classifier.Classify`
(`internal/agents/claudecode/errors.go:37-38`) gates usage-limit detection on
`strings.Contains` of `"hit your limit"` or `"hit your session limit"`. The
credits message contains neither. It equally fails `commonRateLimitPatterns`
(`"rate limit"`, `"rate_limit"`, `"429"`, `"too many requests"`), the auth,
session, connection, and overload sets — so it reaches the
`NewUnknownError` fallthrough at `errors.go:76`.

**Layer 2 — the reset-time requirement.** Even with a matching phrase, the
branch at `errors.go:39-48` only returns `ErrorClassRateLimitWait` when
`parseUsageLimitReset` yields a positive duration. Credit exhaustion has no
reset time — it is cleared by human action (buying credits or switching model),
not by the clock — so this branch is structurally unable to handle it.

**Layer 3 — the message is discarded.** `processExecResult`
(`internal/agents/claudecode/agent.go:303-309`) returns the raw
`*exec.ExitError` whenever `execResult.Err != nil`, which is always true here,
overriding the `ClassifiedError` built just above it. `classifyFromAgent`
(`internal/orbit/single.go:71-73`) then synthesises
`errMsgs = []string{"exit status 1"}`, and `BuildUnknownMessage`
(`internal/agents/errors.go:237-249`) prefers non-empty `errMsgs` over `stderr`
— so the real text, which *was* correctly passed into `Classify` for matching,
is never used for the recorded message. `internal/orbit/variants.go:687` wraps
that into the final `phase %s: %w` string.

**Defect type:** Missing external-message variant handling, compounded by an
error-precedence bug that discards diagnostic text.

**Why it occurred:** Classification is keyed on human-readable prose emitted by
an external tool. That prose is not a stable interface — it has now changed
three times. Each previous fix added one more literal string, which addressed
the instance but not the fragility.

**Contributing factors:** The saved session JSON already contains structured,
machine-stable fields that identify this unambiguously:

```json
{
  "is_error": true,
  "terminal_reason": "api_error",
  "api_error_status": 429,
  "subtype": "success",
  "result": "You're out of usage credits. Run /usage-credits to keep using Fable 5 or /model to switch models."
}
```

`api_error_status` and `terminal_reason` appear nowhere in orbit's Go source, so
these signals are currently discarded. Note also `subtype: "success"` alongside
`is_error: true` — related to `claude-non-json-output-success`.

## Proposed Resolution

**Recommended approach — classify on the structured fields, not the prose.**
Parse `terminal_reason` and `api_error_status` from the agent's JSON result and
classify from those, keeping the literal-phrase matching only as a fallback for
older CLI versions. This is the change that stops the recurrence, rather than
adding a fourth magic string that will break on the fifth wording.

**A new error class is needed.** Credit exhaustion is neither
`ErrorClassRetryable` nor `ErrorClassRateLimitWait`:

- `ErrorClassRetryable` would burn all five attempts over ~30s against a
  condition that cannot clear on its own.
- `ErrorClassRateLimitWait` requires a reset time that does not exist here.

It wants a terminal-but-attributable class that stops the run promptly and
reports the cause accurately — the operational difference being that the user
must act (buy credits, or `--agent opus`), and should be told so rather than
being handed `exit status 1`.

**Secondary fix (independently valuable).** Make `processExecResult` prefer a
matched `ClassifiedError` over the raw `*exec.ExitError`, or at minimum stop
`classifyFromAgent` from synthesising `"exit status 1"` into `errMsgs` when real
`stderr`/`result` text exists. Today *any* classified failure that also exits
non-zero loses its message this way, so this is not specific to credits.

**Alternatives considered:**
- **Add `"usage credits"` to `commonRateLimitPatterns`** — Rejected: yields
  `ErrorClassRetryable` with a 60s `DefaultRateLimitRetryAfter`, wasting five
  attempts on an unclearable condition, and perpetuates prose matching.
- **Add a fourth literal phrase to the existing usage-limit guard** — Rejected
  on its own: the reset-time requirement at `errors.go:39-48` means the branch
  still would not fire, and it repeats the pattern that has now failed
  three times.

## Affected Files

- `internal/agents/claudecode/errors.go:27-77` — `Classify` phrase guards
- `internal/agents/claudecode/errors.go:89` — `parseUsageLimitReset` regex
- `internal/agents/claudecode/agent.go:303-309` — `execResult.Err` precedence
- `internal/agents/errors.go:109-115, 237-249` — pattern sets, `BuildUnknownMessage`
- `internal/orbit/single.go:51-85` — `classifyFromAgent`
- `internal/agents/retry.go:85-87` — non-retryable early return

## Related

- `claude-session-limit-message` — second wording variant (2026-07-22)
- `usage-limit-no-timezone` — reset-time parsing variant (2026-03-05)
- `gmt-usage-limit-dst-offset` — reset-time timezone handling
- `claude-non-json-output-success` — `subtype: "success"` with `is_error: true`
- `variant-run-summary-never-finalized` — why this failure also leaves the
  run-level summary stuck at `"running"`
- `variant-phase-loop-no-progress-guard` — the sibling defect that keeps a
  surviving variant looping after the other one dies here
