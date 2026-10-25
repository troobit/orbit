# Bugfix Report: Local context and progress

**Date:** 2026-10-25
**Status:** Context controls verified; completion accounting fixed

## Description of the Issue

LM Studio reported removing 23110 tokens from a 33340-token request using
TruncateMiddle. The worker continued editing but dependency installation was
blocked, and shell pipelines printed success statuses after failed commands.
The run ultimately failed because no ledger task completed. Its summary
nevertheless counted the failed phase as completed.

**Reproduction steps:**

1. Use the local preset with an LM Studio model loaded at 32768 tokens.
2. Run a task requiring enough tool turns to grow the conversation beyond
   that window, or dependencies unavailable in the sandbox.
3. Observe server-side truncation and the failed/no-progress run summary.

**Impact:** Loss of intermediate context, wasted iterations retrying blocked
prerequisites, and misleading completed-phase accounting. Raw JSONL and stderr
were retained; terminal-only activity presentation is not part of this fix.

## Investigation Summary

- Read the real failed run's summary and retained artifacts without modifying
  the target repository. The error was `no selected tasks completed`; code
  edits alone were not treated as successful work.
- Checked `lms ps --json` and the model API: loaded context was 32768 despite
  the model advertising a maximum of 262144. Model capacity and loaded context
  are different values.
- The preset set neither context nor compaction limits. It used workspace-write
  with networking disabled and no writable dependency-cache guidance.
- `SaveSession` advanced completed phases unconditionally, including failures.
- Considered smaller tasks and alternate harnesses. Nested tasks are dispatched
  together with their parent, and the current OpenCode adapter expects one JSON
  document and does not pass live-output/cancellation options through.

## Discovered Root Cause

The preset inherited model-context assumptions without matching the loaded
runtime, while failure reporting conflated saving an attempt with completing
work. Missing dependencies were an independent blocker. TruncateMiddle removes
history; repeated metadata refresh errors are a separate model-catalog
compatibility issue. Aggregate input tokens over a turn are not the size of a
single prompt.

## Resolution for the Issue

**Changes made:** Local workers now receive explicit context and earlier
compaction limits (32768/20480 by default), a 2048-token tool-output history
budget and configurable tool networking via `--network`. Preflight rejects
invalid budgets. Worker prompts request bounded reads, correct validation exit
status and a halt for unavailable prerequisites. Failed sessions remain logged
but no longer advance completed-phase counts.

The runbooks distinguish context loss from catalog warnings, explain loaded
context versus model maximum, and compare smaller tasks, larger context,
OpenCode and a direct API harness. Extra terminal activity display was excluded
after the user clarified that this visual change was unnecessary.

**Alternatives considered:** Smaller tasks alone do not solve dependencies or
oversized reads. Larger context alone needs resources and a deliberate reload.
OpenCode remains a candidate but needs adapter verification; a direct API
integration requires an execution loop. No target ledger, model load or server
overflow policy was changed automatically.

## Regression Test

- `TestLocalCommandDispatch` verifies default context/compaction/tool-output
  settings on actual worker arguments.
- `TestLocalCommandContextAndNetwork` checks explicit budgets and opt-in network
  access; preflight cases reject invalid budgets without worker/log creation.
- `TestManager_FailedSessionDoesNotAdvanceProgress` verifies failed turns and
  process failures retain diagnostics without increasing completed phases.

Run with `make test-local` and `make test`.

## Affected Files

Local CLI and worker prompt, log manager and regression tests, local runbooks,
feature specification and agent notes.

## Verification

- [x] Regression tests fail before the fix and pass afterward.
- [x] Context and sandbox behavior checked with installed Codex and a loopback stub.
- [x] Build, full tests and lint pass.

Installed Codex 0.164.0 was supplied a 22000-token usage report against the
32768/20480 budgets. It issued a separate local summary request with no tools,
then resumed a normal tool-enabled request. Separate rebuilt-Orbit checks
completed two dependent rune tasks with real shell and Makefile calls, both
with default networking and with `--network`. The enabled-network case made
two actual HTTP requests from sandboxed shell tools to the loopback stub.

These checks use stub responses, not new live-model inference. They verify
configuration, compaction activation, tool execution and failure handling;
they do not establish LM Studio tokenizer agreement or summary quality under
a long real coding task. Model-catalog refresh errors remain a separate
compatibility limitation. Previous failed summaries were not rewritten.

Documentation links, rune parsing and `git diff --check` passed. The existing
installed launcher was updated through `make install-local-next-task`; it
matches the validated build and exposes all three new options in `--help`.

The longer user-driven run was pending when the initial context controls were
committed. Its first iteration subsequently passed: real compaction reduced
the server-side prompt count from 20243 to 10947 tokens, tools resumed, three
tests and Ruff checks passed, task 1 completed through rune, and Orbit accepted
the result before starting iteration 2. Duration was 1098.955 seconds. No
TruncateMiddle events occurred within that observed iteration. The overall run
continues; see the feature verification for the scope and remaining limitations.

### Restarted-summary follow-up

The next live run reported `status: running` with the previous run's error and
completion time still set. The existing-summary regression reproduced these
stale fields. New invocations now clear `Error` and `CompletedAt` when marking
the summary running, while preserving session history and incremented run
numbering. The active user's summary was not edited while the worker owned it.

## Prevention

Pin runtime budgets, distinguish model capacity from loaded context, preserve
validation exit codes, and test failure accounting. Record evidence before
changing harnesses or assuming smaller tasks solve environmental blockers.

## Related

- [Local design](../../local-compute-agent/design.md)
- [Local runner](../../../docs/runbooks/local-next-task.md)
