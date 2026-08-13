# Requirements: Run Termination Guarantees

## Introduction

Orbit currently has no guarantee that a run ends. On 2026-08-13 a single
`orbit run --variants 2 --parallel` invocation re-ran one phase **1223 times**
on variant 1 and **1394 times** on variant 2, because the phase's only open task
was human-gated and could never be completed by an agent. Each iteration started
a fresh agent session, re-executed the phase in full (including the repository's
test suite), correctly concluded the work was not agent-executable, exited
cleanly, and was started again. The run recorded **$1089.53**, of which roughly
**$830 bought no forward progress**, and ended only when the account's usage
credits were exhausted.

That exhaustion was then misreported. The Claude CLI's credit-exhaustion message
matches none of orbit's classifier patterns, so it surfaced as a bare
`exit status 1` — indistinguishable from a genuine phase failure. Meanwhile the
run-level `summary.json` was never finalized, leaving `"status": "running"` on
disk hours after every variant had died, so from the outside the dead run looked
live.

This feature makes run termination a guarantee rather than an emergent property:
every run reaches a terminal state, records it, and reports the reason
accurately — including when the correct outcome is "stop, a human is needed".

Three bugfix reports contain the full diagnosis and verified code references:
`specs/bugfixes/variant-phase-loop-no-progress-guard/report.md`,
`specs/bugfixes/claude-out-of-usage-credits/report.md`, and
`specs/bugfixes/variant-run-summary-never-finalized/report.md`.

---

## Definitions

- **Phase loop**: the `for {}` in `runVariant` (`internal/orbit/variants.go:615-719`)
  and its counterpart in `runSingle` (`internal/orbit/single.go:771-837`), which
  repeatedly asks `rune next` for a phase and executes it.
- **Stalled iteration**: a phase iteration that completes without error and
  without changing the set of incomplete tasks.
- **Human-gated task**: a task a coding agent cannot complete because it requires
  physical or human action (conventionally marked `STOP` in the consuming
  project's task files). Orbit does not currently model this concept.
- **Terminal state**: a recorded run status that is not `running` — currently
  `success` or `failed`, plus the new blocked state introduced here.
- **Run-level manager**: the `logs.Manager` constructed at
  `internal/orbit/orbit.go:264-273` that owns `<spec>/.orbit/summary.json`, as
  distinct from the per-variant managers created in `runVariant`.

---

## Requirements

### 1. The Phase Loop Terminates On Stall

**User Story:** As an orbit user, I want a run that cannot make progress to stop
and tell me why, so that it does not consume my budget indefinitely on work no
agent can do.

**Acceptance Criteria:**

1. <a name="1.1"></a>The system SHALL record a measure of outstanding work before executing a phase and compare it after the phase completes.
2. <a name="1.2"></a>WHEN a phase iteration completes without error and the measure of outstanding work is unchanged, THEN the system SHALL stop that variant rather than beginning another iteration.
3. <a name="1.3"></a>WHEN the system stops a variant for a stalled iteration, it SHALL record a terminal state distinct from a phase failure, so that "blocked, needs a human" is not reported as "the code failed".
4. <a name="1.4"></a>The recorded stop reason SHALL name the phase and the task or tasks that remained incomplete.
5. <a name="1.5"></a>The system SHALL apply the same stall detection to the single-run phase loop as to the variant phase loop.
6. <a name="1.6"></a>The stall check SHALL NOT trigger when a phase fails with an error, since that path already terminates the loop.

### 2. A Bounded Phase Loop

**User Story:** As an orbit user, I want a hard ceiling on phase iterations, so
that any stall the progress check fails to model still cannot run unbounded.

**Acceptance Criteria:**

1. <a name="2.1"></a>The phase loop SHALL enforce a maximum iteration count per variant.
2. <a name="2.2"></a>WHEN the iteration ceiling is reached, the system SHALL stop the variant and record that the ceiling was the reason.
3. <a name="2.3"></a>The ceiling SHALL be configurable, and its default SHALL be high enough that no legitimate multi-phase run reaches it.
4. <a name="2.4"></a>The ceiling SHALL be a backstop, not the primary guard — a stalled run SHALL normally stop via requirement 1 well before it.

### 3. An Agent-Visible Halt Signal

**User Story:** As an agent running inside an orbit phase, I want a way to tell
orbit that work is blocked on a human, so that orbit stops for the right reason
instead of inferring it.

**Acceptance Criteria:**

1. <a name="3.1"></a>The system SHALL define a halt sentinel file that an agent can write to signal that the run is blocked pending human action.
2. <a name="3.2"></a>The sentinel path SHALL be defined against a directory orbit itself owns and enumerates, and SHALL be documented for agent authors.
3. <a name="3.3"></a>WHEN the sentinel is present after a phase completes, the system SHALL stop that variant and record the blocked terminal state.
4. <a name="3.4"></a>The contents of the sentinel SHALL be read and recorded as the stop reason.
5. <a name="3.5"></a>The system SHALL clear or ignore a stale sentinel from a previous run so it cannot halt a subsequent one.
6. <a name="3.6"></a>The halt signal SHALL be optional — requirement 1 SHALL stop a stalled run whether or not the agent writes a sentinel.

### 4. Credit Exhaustion Is Classified And Reported

**User Story:** As an orbit user, I want a run that stopped because I ran out of
credits to say so, so that I do not investigate my code for a billing problem.

**Acceptance Criteria:**

1. <a name="4.1"></a>The system SHALL classify Claude Code credit exhaustion from the structured fields the agent already emits (`terminal_reason`, `api_error_status`, `is_error`) rather than solely from the human-readable message text.
2. <a name="4.2"></a>The system SHALL retain literal-message matching as a fallback for agent versions that do not emit the structured fields.
3. <a name="4.3"></a>Credit exhaustion SHALL be classified into a class that is neither retried nor waited on, because it has no reset time and clears only by human action.
4. <a name="4.4"></a>The recorded error SHALL state the cause and the available remedies, and SHALL NOT be reduced to `exit status 1`.
5. <a name="4.5"></a>WHEN an agent's diagnostic text is available, the system SHALL prefer it over a synthesised exit-status string when building the recorded error message.
6. <a name="4.6"></a>Classification SHALL remain correct for the existing session-limit messages, with and without a reset time and timezone.

### 5. Every Run Reaches A Recorded Terminal State

**User Story:** As an orbit user or a tool reading orbit's output, I want the
run-level summary to say whether the run finished, so that a dead run is not
indistinguishable from a live one.

**Acceptance Criteria:**

1. <a name="5.1"></a>WHEN a variant-mode run ends by any route, the system SHALL finalize the run-level `summary.json` with a terminal status and a completion timestamp.
2. <a name="5.2"></a>The finalized run-level summary SHALL aggregate per-variant results into its phase, cost, and duration totals rather than leaving them at zero.
3. <a name="5.3"></a>The aggregation rule for the run-level status SHALL be defined explicitly, including the case where every variant failed and the case where variants disagree.
4. <a name="5.4"></a>The system SHALL update the run-level registry entry to a terminal status in variant mode, as it already does in single-run mode.
5. <a name="5.5"></a>WHEN a run is interrupted by signal or context cancellation, the system SHALL still record a terminal state.
6. <a name="5.6"></a>A subsequent run against the same log directory SHALL NOT be able to increment `run_number` past a summary that was never closed out.

### 6. The Loop Is Observable

**User Story:** As an orbit user diagnosing a run, I want repeated phase
execution to be visible in the artifacts, so that I can see a stall without
reconstructing it from session arrays.

**Acceptance Criteria:**

1. <a name="6.1"></a>WHEN a phase is executed more than once within a single invocation, the system SHALL make the repetition visible in the log artifacts rather than overwriting the previous attempt's files.
2. <a name="6.2"></a>The system SHALL surface a warning when a phase has been executed more than a small number of times within one invocation.
3. <a name="6.3"></a>Documentation SHALL state that the `-run-N` filename suffix counts `orbit run` invocations, not loop iterations, since this has already caused a misreading.

---

## Out Of Scope

- Modelling task gating (`STOP` semantics) inside `rune` itself. Requirement 3
  gives agents a signal without requiring a task-schema change; recognising a
  gated task before launching a phase is a possible later optimisation.
- Cost or wall-clock budget caps. These were considered as the primary guard and
  rejected in the decision log — they stop the bleeding without diagnosing the
  cause.
- Any change to how variants are compared or consolidated.
