# Decisions: Local task preparation

## Decision 1: Prepare separate ledgers and structured packets

**ID**: 1  
**Date**: 2026-10-29  
**Status**: accepted

### Context

A Sanarte `orbit local` run spent 19m18.893s on one broad task, repeatedly
reading context and making no source edits before an authorised interruption.
The source task combines implementation, browser acceptance and provider
capability decisions. The user wants host reasoning to enable smaller workers,
with programmatic handoffs and retained improvements and failures.

### Decision

Prepare a separate rune ledger from an approved source revision, using
structured packets and a visible source-outcome map. Host code validates and
materialises the planner's JSON. Prepared assignments get new IDs; original
scope, completed history and deferred owner acceptance remain in the source.

### Rationale

Decomposition is a planning operation. Making a local implementation model
perform it during every session spends context before any code changes and
makes scope loss difficult to detect. A separate artifact can be reviewed once
and consumed repeatedly by different workers.

### Alternatives Considered

- Rewrite the active source ledger: rejected because it changes live graph
  identity, risks erasing acceptance and makes retained partial work hard to trace.
- Increase timeout/context alone: rejected because it does not narrow the
  reasoning burden or resolve human/provider prerequisites.
- Enable model-native subagents: rejected because the observed local provider
  does not support that namespace, and delegation does not define a work contract.

### Consequences

Positive: explicit source lineage, smaller assignments and reusable failure
evidence; no new tool-calling harness.

Negative: a separate candidate requires semantic review; structural coverage
and file limits cannot establish that a smaller model can finish the work.

---

## Decision 2: Pin approval and verify acceptance in the host

**ID**: 2  
**Date**: 2026-10-29  
**Status**: accepted

### Context

The current local runner proves selected task completion and graph integrity,
but a worker controls both its validation prose and rune status updates. Red
checkpoints deliberately have failing tests. Useful partial edits must survive
failed attempts without being mistaken for accepted work.

### Decision

Require explicit approval of source/candidate definitions and commands. Run
packet acceptance independently in the host, including declared red-failure
evidence. The host alone changes execution task statuses and completes them
after acceptance. Persist attempts before accounting and preserve code on
failure. Interrupted host checks must be recovered before any next assignment.

### Rationale

The coordinator can enforce command results and revision identity without
depending on model honesty or attention. Review remains responsible for
semantic scope and the suitability of expected-red assertions.

### Alternatives Considered

- Trust a completed rune task: rejected because status is a claim rather than
  independent evidence, and can skip work after interrupted validation.
- Require every checkpoint to pass all tests: rejected because it collapses
  the explicit red/green handoff into one oversized assignment.
- Revert all failed edits: rejected because it discards useful implementation
  work and the evidence the user explicitly wants retained.

### Consequences

Positive: failed attempts remain useful and cannot advance accepted work;
approvals remain valid across status-only progress.

Negative: host checks add execution time and recovery state. Declared evidence
can be inadequate, so human review remains necessary.

---

## Decision 3: Share prepared dispatch across existing runners

**ID**: 3  
**Date**: 2026-10-29  
**Status**: accepted

### Context

Local aliases in standard variants reproduce CLI flags but omit `orbit local`'s
selection, graph checks, run-once policy and cancellation. The user intends a
hosted Codex/Opus implementation comparison followed by local validation from
a fork. Provider choice must not silently change the prepared work contract.

### Decision

Extract the existing bounded worker operations into per-worktree runtime state.
Prepared ledgers activate that runtime in local, single and variant execution.
Start with Codex and Claude adapters; keep ordinary ledgers' existing policies.
Parallelism uses existing isolated variant worktrees, with sequential fresh
workers inside each variant.

### Rationale

The existing agent interface and variant manager already provide execution and
isolation. Sharing the task contract closes the policy gap without adding a
new inference loop or a scheduler for concurrent edits to one checkout.

### Alternatives Considered

- Only emit more subtasks: rejected because default parent/phase dispatch
  bundles them, and alias flags still omit the runner controls.
- Nest `orbit local` subprocesses under variants: rejected because it duplicates
  lifecycle/accounting and obscures ownership of cancellation and failures.
- Replace all hosted phase execution: rejected because existing custom prompts,
  commit flows and retry policies have different contracts.

### Consequences

Positive: one prepared-work contract across models, retained registry/logging
and isolated variant failures.

Negative: extraction must avoid sharing mutable iteration state; Claude needs
output/cancellation forwarding, and other adapters initially reject prepared work.

---

## Decision 4: Use a readiness receipt for bounded handoff

**ID**: 4  
**Date**: 2026-10-29  
**Status**: accepted

### Context

The supplied Sanarte RED packet produced the exact test. Independent host
verification found 14 passing existing tests and the one intended assertion
failure. The worker still reached its six-minute timeout before finalising its
session. Useful code and model session completion are different events.

### Decision

Give prepared workers an exact command to submit an attempt-bound readiness
receipt. Stop that worker on valid receipt, then independently validate and
host-complete its assignment. Also support normal successful exits and recheck
retained work before spending another inference session.

### Rationale

A deterministic handoff lets the supervisor finish a concrete work unit without
requiring the model to produce more prose. The receipt requests validation; it
does not establish acceptance. Raw cancellations/errors remain retained evidence.

### Alternatives Considered

- Wait for natural completion or increase timeouts: rejected because the observed
  correct edit can already be checked while finalisation continues consuming compute.
- Treat a file change or completed model status as success: rejected because
  neither proves the approved acceptance predicates.
- Kill a worker on a timer and accept any passing checks: rejected because a
  deadline is a failure, while an explicit current-attempt handoff has a defined owner.

### Consequences

Positive: bounded machine-readable completion, independently accepted retained
work and less dependence on model-written task status or narration.

Negative: the supervisor must handle receipt/process/deadline races and keep
raw process termination separate from host acceptance accounting.
