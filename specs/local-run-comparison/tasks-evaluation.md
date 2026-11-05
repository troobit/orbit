---
references:
    - specs/local-run-comparison/smolspec.md
    - specs/local-run-comparison/decision_log.md
    - specs/local-run-comparison/prerequisites.md
---
# Local harness comparison opt-in evaluation

## Phase 1: Evaluation approval

- [ ] 1. STOP: Confirm prerequisites and owner approval for live evaluation <!-- id:slnoxx0 -->
  - Gate only: do not dispatch this ledger unattended. Complete manually after the owner requests live fixture runs and prerequisites.md Before releasing the evaluation gate is satisfied; coding completion is not approval.
  - Record built Orbit commit, installed CLI versions, rune commit, existing model/load/context, helper routing, hosted-judge authorization, artifact locations and an uncontended timeout window. Do not reconfigure LM Studio.
  - Stream: 1
  - Requirements: [6.1](specs/local-run-comparison/smolspec.md#6.1), [7.1](specs/local-run-comparison/smolspec.md#7.1), [9.1](specs/local-run-comparison/smolspec.md#9.1), [10.1](specs/local-run-comparison/smolspec.md#10.1), [11.1](specs/local-run-comparison/smolspec.md#11.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)

## Phase 2: Disposable smoke acceptance

- [ ] 2. Verify disposable two-phase execution paths <!-- id:slnoxx1 -->
  - Use the coding-phase generator/runbook. Keep results separate for direct local and alias/variant execution; preserve each result before reset.
  - Complete only when all children pass. Missing judge, contended timeout probes, failed code acceptance or cleanup failure leave this stage incomplete and block Sanarte.
  - Blocked-by: slnoxx0 (STOP: Confirm prerequisites and owner approval for live evaluation)
  - Stream: 1
  - Requirements: [3.1](specs/local-run-comparison/smolspec.md#3.1), [4.1](specs/local-run-comparison/smolspec.md#4.1), [6.1](specs/local-run-comparison/smolspec.md#6.1), [7.1](specs/local-run-comparison/smolspec.md#7.1), [8.1](specs/local-run-comparison/smolspec.md#8.1), [9.1](specs/local-run-comparison/smolspec.md#9.1), [10.1](specs/local-run-comparison/smolspec.md#10.1), [11.1](specs/local-run-comparison/smolspec.md#11.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)
  - [ ] 2.1. Run orbit local through both approved fixture phases <!-- id:slnoxx2 -->
    - Use a fresh fixture, --phase with runner selection, recorded loaded context and a two-iteration limit. Preview, then execute both phases with a bounded timeout. No aliases or hosted judge are needed for this check.
    - Acceptance: both identities completed, independent make test passes, Orbit terminates successfully and prompts/logs/settings are retained. Record failures in runs.md; prose or task status alone cannot complete this smoke.
    - Blocked-by: slnoxx0 (STOP: Confirm prerequisites and owner approval for live evaluation)
    - Stream: 1
    - Requirements: [3.1](specs/local-run-comparison/smolspec.md#3.1), [4.1](specs/local-run-comparison/smolspec.md#4.1), [11.1](specs/local-run-comparison/smolspec.md#11.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)
  - [ ] 2.2. Verify alias timeout cleanup without competing inference <!-- id:slnoxx3 -->
    - Probe each alias separately with a forced two-minute per-attempt timeout. After the Orbit invocation terminates, confirm no measured request survives and LM Studio is idle within 60 seconds. Record every retry so a fresh attempt is not mistaken for an orphan.
    - Try installed OpenCode --standalone and recheck if necessary. Surviving requests or competing inference block acceptance and longer runs; record and scope a repair. Do not kill unrelated model users.
    - Blocked-by: slnoxx2 (Run orbit local through both approved fixture phases)
    - Stream: 1
    - Requirements: [7.1](specs/local-run-comparison/smolspec.md#7.1), [10.1](specs/local-run-comparison/smolspec.md#10.1), [11.1](specs/local-run-comparison/smolspec.md#11.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)
  - [ ] 2.3. Run each alias separately on the same fixture base <!-- id:slnoxx4 -->
    - Create/reset clean fixtures at the same approved base. Verify actual instruction loading, flags, local provider/helper routing and matching budgets. Run codex-local/opencode-local separately without pre/post prompts or consolidation.
    - Acceptance: each completes both phases with independent make test and a host commit beyond base. No worker commit, log/secret staging or task graph rewrite. Retain durations, retries and restrictions.
    - Blocked-by: slnoxx3 (Verify alias timeout cleanup without competing inference)
    - Stream: 1
    - Requirements: [6.1](specs/local-run-comparison/smolspec.md#6.1), [7.1](specs/local-run-comparison/smolspec.md#7.1), [8.1](specs/local-run-comparison/smolspec.md#8.1), [11.1](specs/local-run-comparison/smolspec.md#11.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)
  - [ ] 2.4. Run the parallel fixture comparison and retain its report <!-- id:slnoxx5 -->
    - Reset to the same approved base and run standard variants with explicit aliases/ledger/guidance and authorized claude-code judge. Preserve console exit status through tee.
    - Acceptance: two completed variants and attributable commits, independent make test for each, unchanged base, no hand-edited variants.json and a report with a successful judge result and recommendation naming a completed variant; status-only HTML does not pass. Record recommendation as indicative and verify cleanup/reset.
    - Blocked-by: slnoxx4 (Run each alias separately on the same fixture base)
    - Stream: 1
    - Requirements: [6.1](specs/local-run-comparison/smolspec.md#6.1), [8.1](specs/local-run-comparison/smolspec.md#8.1), [9.1](specs/local-run-comparison/smolspec.md#9.1), [11.1](specs/local-run-comparison/smolspec.md#11.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)

## Phase 3: Sanarte scope approval

- [ ] 3. STOP: Review permission and the narrowed Sanarte trial scope <!-- id:slnoxx6 -->
  - Complete only after owner authorization for Sanarte staging/trial and prerequisites.md Before the Sanarte task is satisfied. This is separate from permission for disposable fixture inference.
  - Confirm clean wt-0/base, baseline checks, offline dependencies, writable caches, 65536-context capacity and code-only outcomes. Keep original acceptance/IDs owner-gated and leave main checkout, simplify-for-square and wt-1 intact.
  - Blocked-by: slnoxx1 (Verify disposable two-phase execution paths)
  - Stream: 1
  - Requirements: [3.1](specs/local-run-comparison/smolspec.md#3.1), [5.1](specs/local-run-comparison/smolspec.md#5.1), [6.1](specs/local-run-comparison/smolspec.md#6.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)

## Phase 4: Sanarte staging

- [ ] 4. Stage the reviewed code-only Sanarte ledger and local configuration <!-- id:slnoxx7 -->
  - Read target spec/ledger first. Use rune to move full original owner outcomes and stable IDs to tasks-owner-gated.md; create narrower code tasks with new IDs, one per phase, and preserve completed history.
  - Keep deferred acceptance via references and owner gates; resolve every within-file blocker and add required behavior tests to Sanarte make test. Exclude browser/widget policy and genuine Square acceptance from unattended work.
  - Commit reviewed templates/ledger and .orbit ignore rule on wt-0, record exact base for helper, and verify rune list/next --phase and baseline Makefile checks. No inference in this task.
  - Blocked-by: slnoxx6 (STOP: Review permission and the narrowed Sanarte trial scope)
  - Stream: 1
  - Requirements: [3.1](specs/local-run-comparison/smolspec.md#3.1), [4.1](specs/local-run-comparison/smolspec.md#4.1), [5.1](specs/local-run-comparison/smolspec.md#5.1), [6.1](specs/local-run-comparison/smolspec.md#6.1), [7.1](specs/local-run-comparison/smolspec.md#7.1), [8.1](specs/local-run-comparison/smolspec.md#8.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)

## Phase 5: Sanarte experiment

- [ ] 5. Run the gated Sanarte comparison and record independent results <!-- id:slnoxx8 -->
  - Use approved staging base/aliases/context/timeouts/guidance; disable pre/post prompts and consolidation. Retain console output with pipeline exit propagation, every observed attempt/timeout and finish times.
  - Inspect rune states/commits and independently run make check, make test and make build in each final worktree. Preserve diffs/artifacts and report or explicit failure/no-report reason before cleanup removes only experiment-created variants/metadata.
  - Acceptance is a truthful runs.md result, including failed variants, not feature completion. Compare main checkout, simplify-for-square and wt-1 before/after. Never run owner tasks/real bookings or relax gates for a recommendation.
  - Blocked-by: slnoxx7 (Stage the reviewed code-only Sanarte ledger and local configuration)
  - Stream: 1
  - Requirements: [6.1](specs/local-run-comparison/smolspec.md#6.1), [7.1](specs/local-run-comparison/smolspec.md#7.1), [8.1](specs/local-run-comparison/smolspec.md#8.1), [9.1](specs/local-run-comparison/smolspec.md#9.1), [10.1](specs/local-run-comparison/smolspec.md#10.1), [11.1](specs/local-run-comparison/smolspec.md#11.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)
