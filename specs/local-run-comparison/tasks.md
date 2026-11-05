---
references:
    - specs/local-run-comparison/smolspec.md
    - specs/local-run-comparison/decision_log.md
---
# Local harness comparison implementation

## Phase 1: OpenCode event streams

- [ ] 1. Accept typed OpenCode event streams without classifying reply text <!-- id:i5ei0te -->
  - Scope: internal/agents/opencode/agent.go and errors.go, with behavior tests in that package. Preserve hosted provider independence and existing plaintext error diagnostics.
  - Use --leaf for separate red and green sessions; default task/phase dispatch completes both children. No model CLI or network is needed.
  - Stream: 1
  - Requirements: [1.1](specs/local-run-comparison/smolspec.md#1.1)
  - [ ] 1.1. RED: Reproduce OpenCode event-stream and classification failures <!-- id:i5ei0tf -->
    - Test two-event and single-event success, valid events mixed with noise, error events at exits 0 and 1, and an error without a message. Assert IsError, exit handling, session ID and classification source.
    - Reject empty/plaintext-only output, arrays/scalars/null and untyped objects as events. Preserve plaintext auth/model/rate-limit diagnostics. Valid reply/tool text mentioning timeout or api key must not cause an error.
    - Acceptance: make test-local fails on these new behavior assertions against current code; record expected failing cases and leave production code unchanged.
    - Stream: 1
    - Requirements: [1.1](specs/local-run-comparison/smolspec.md#1.1)
  - [ ] 1.2. GREEN: Parse typed events and classify only diagnostics <!-- id:i5ei0tg -->
    - Share the event scan between adapter and classifier; retain raw output, collect top-level errors and prefer an event sessionID. A single typed event is valid; an untyped JSON document is not.
    - Use error-event messages, adapter errors and stderr for classification; plaintext stdout is a fallback only when no valid event exists. Do not change other agents or add output streaming/process-group work.
    - Acceptance: unchanged red tests and existing OpenCode cases pass with make test-local; make test and make lint pass.
    - Blocked-by: i5ei0tf (RED: Reproduce OpenCode event-stream and classification failures)
    - Stream: 1
    - Requirements: [1.1](specs/local-run-comparison/smolspec.md#1.1)

## Phase 2: Variant completion guard

- [ ] 2. Stop successful variant sessions that complete no selected work <!-- id:i5ei0th -->
  - Scope: runVariant in internal/orbit/variants.go and behavior tests under internal/orbit. Use testutil agents and temporary real-rune ledgers, without inference.
  - This spec owns the shared variant guard. Do not add the competing run-termination blocked enum or sentinel. Follow the portability note: rune JSON IDs are task numbers.
  - Blocked-by: i5ei0te (Accept typed OpenCode event streams without classifying reply text)
  - Stream: 1
  - Requirements: [2.1](specs/local-run-comparison/smolspec.md#2.1)
  - [ ] 2.1. RED: Reproduce variant stalls and false progress <!-- id:i5ei0ti -->
    - Assert a successful no-op dispatch fails its variant, registry and log summary after one session, without retry, while a sibling can complete. Require phase, next task and stalled in the error. Use a bounded test context to avoid hanging the suite.
    - Cover leaf completion under an incomplete parent, final-phase completion and title-cased statuses. In-progress-only changes, deletion/replacement/renumbering, additions and unrelated completion must not pass.
    - Inject a post-session ledger read error; assert its reason and retained diagnostics. A stalled session must not increase completed phases. Acceptance: make test-local fails on these new assertions with production code unchanged.
    - Blocked-by: i5ei0te (Accept typed OpenCode event streams without classifying reply text)
    - Stream: 1
    - Requirements: [2.1](specs/local-run-comparison/smolspec.md#2.1)
  - [ ] 2.2. GREEN: Require selected task identity to transition to completed <!-- id:i5ei0tj -->
    - Capture incomplete numeric IDs, titles, ancestor paths and statuses recursively from the dispatched phase. Re-read ListAll after success and require a matching selected identity to become completed. Do not compare different next phases or count/ID churn.
    - Attach a stall/read error before SaveSession/CompletePhase and use the existing variant failed-state, registry and log path. Preserve diagnostics, skip success accounting, do not feed a stalled success back into retries, and let siblings continue.
    - Leave full variant graph validation, single-run guards and retry policy out of scope. Acceptance: unchanged red tests pass; make test-local, make test and make lint pass.
    - Blocked-by: i5ei0ti (RED: Reproduce variant stalls and false progress)
    - Stream: 1
    - Requirements: [2.1](specs/local-run-comparison/smolspec.md#2.1)

## Phase 3: Comparable result commits

- [ ] 3. Commit variant results with attributable stable task IDs <!-- id:i5ei0tk -->
  - Scope: planned docs/runbooks/local-variant-commit.sh and Go integration tests under internal/orbit, included by make test-local. This is a host post-command; workers do not commit.
  - Use temporary Git repositories, rune and Python 3. Isolate Git config/signing. No credentials, model or Sanarte checkout is needed.
  - Blocked-by: i5ei0th (Stop successful variant sessions that complete no selected work)
  - Stream: 1
  - Requirements: [8.1](specs/local-run-comparison/smolspec.md#8.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)
  - [ ] 3.1. RED: Specify commit-helper behavior in disposable repositories <!-- id:i5ei0tl -->
    - Test explicit base/ledger paths with spaces, nested completion, renumbering, ORBIT_AGENT alias naming and a repeated no-change invocation. Assert resulting commits/messages and exact staged-path allowlisting, not just script existence.
    - Validate statuses with rune JSON and map each revision to stable markers in rune canonical Markdown before comparison. Messages must name newly completed stable IDs, excluding already-completed base tasks.
    - Assert no commit for malformed/missing ledgers, missing/duplicate stable IDs and nonexistent/non-ancestor bases. Logs, secrets and owner ledgers stay unstaged. Acceptance: make test-local fails on these cases before helper implementation.
    - Blocked-by: i5ei0th (Stop successful variant sessions that complete no selected work)
    - Stream: 1
    - Requirements: [8.1](specs/local-run-comparison/smolspec.md#8.1), [5.1](specs/local-run-comparison/smolspec.md#5.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)
  - [ ] 3.2. GREEN: Implement the reusable host-side commit helper <!-- id:i5ei0tm -->
    - Implement the shell helper with explicit ledger/base/staged-path inputs. Use rune validation, JSON statuses and canonical Markdown stable-ID mapping; document Python 3. Fail without committing on invalid input and no-op on no changes.
    - Stage only declared source/test/ledger paths, never unconditional git add -A. Commit the alias and completed stable IDs since base; do not push.
    - Acceptance: unchanged red tests pass; sh -n docs/runbooks/local-variant-commit.sh, make test-local, make test and make lint pass.
    - Blocked-by: i5ei0tl (RED: Specify commit-helper behavior in disposable repositories)
    - Stream: 1
    - Requirements: [8.1](specs/local-run-comparison/smolspec.md#8.1), [5.1](specs/local-run-comparison/smolspec.md#5.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)

## Phase 4: Reusable two-phase fixture

- [ ] 4. Provide a fixture and runbook for approved-ledger execution <!-- id:i5ei0tn -->
  - Scope: planned docs/runbooks/local-variant-fixture.sh, local-variant-guidance.yaml, alias templates and local-variant-comparison.md. Integrate automated fixture/template checks into existing Makefile test targets.
  - Generate the fixture without Sanarte or inference. This is execution of approved inputs; local SDD authoring remains a later spec.
  - Blocked-by: i5ei0tk (Commit variant results with attributable stable task IDs)
  - Stream: 1
  - Requirements: [3.1](specs/local-run-comparison/smolspec.md#3.1), [4.1](specs/local-run-comparison/smolspec.md#4.1), [6.1](specs/local-run-comparison/smolspec.md#6.1), [7.1](specs/local-run-comparison/smolspec.md#7.1), [9.1](specs/local-run-comparison/smolspec.md#9.1), [10.1](specs/local-run-comparison/smolspec.md#10.1), [11.1](specs/local-run-comparison/smolspec.md#11.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)
  - [ ] 4.1. RED: Specify generated fixture and template behavior without inference <!-- id:i5ei0to -->
    - Invoke the generator in an empty temporary destination. Assert a standalone Git root, fixed approved spec, standard-library Python stub/tests, make test, .orbit ignore rule, and a rune-created two-task/two-phase dependent ledger with stable IDs and resolving references.
    - Assert initial tests fail for intended missing behavior and reruns refuse to overwrite work. Parse templates and inspect fake-agent argv for codex auto-approve false, skip-permissions false, required local flags/model/budgets and no bypass; verify OpenCode local helper/deny rules.
    - Check explicit ledger/guidance, host helper base/path wiring and disabled pre/post prompts/consolidation. No real model, hosted judge or Sanarte. Acceptance: make test-local fails on new behavior assertions before implementation.
    - Blocked-by: i5ei0tk (Commit variant results with attributable stable task IDs)
    - Stream: 1
    - Requirements: [3.1](specs/local-run-comparison/smolspec.md#3.1), [4.1](specs/local-run-comparison/smolspec.md#4.1), [6.1](specs/local-run-comparison/smolspec.md#6.1), [7.1](specs/local-run-comparison/smolspec.md#7.1), [8.1](specs/local-run-comparison/smolspec.md#8.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)
  - [ ] 4.2. GREEN: Generate the fixture and wire acceptance and reset instructions <!-- id:i5ei0tp -->
    - Create generator/templates with rune for the exact disposable fixture contract: normalize_label then unique_labels, accepted by make test-normalize then make test. No downloads, credentials or manual acceptance belong in the fixture. Check rune batch JSON success and read back counts/phases/blockers; an exit code of zero alone is insufficient.
    - Document direct orbit local through both phases, separate aliases, uncontended timeout probes and parallel variants from one base. Cover fresh resets/artifact retention, tee exit propagation, actual retries, summary-only judge limits and warning-only post-command checks.
    - Link docs/runbooks/local-next-task.md and update docs/agent-notes/local-execution.md. Include later Sanarte ID/owner split and runtime prerequisites; do not stage Sanarte or run inference. Acceptance: unchanged red tests, script syntax, links and rune parsing pass; make test-local, make test, make lint and make build pass.
    - Blocked-by: i5ei0to (RED: Specify generated fixture and template behavior without inference)
    - Stream: 1
    - Requirements: [3.1](specs/local-run-comparison/smolspec.md#3.1), [4.1](specs/local-run-comparison/smolspec.md#4.1), [5.1](specs/local-run-comparison/smolspec.md#5.1), [6.1](specs/local-run-comparison/smolspec.md#6.1), [7.1](specs/local-run-comparison/smolspec.md#7.1), [8.1](specs/local-run-comparison/smolspec.md#8.1), [9.1](specs/local-run-comparison/smolspec.md#9.1), [10.1](specs/local-run-comparison/smolspec.md#10.1), [11.1](specs/local-run-comparison/smolspec.md#11.1), [12.1](specs/local-run-comparison/smolspec.md#12.1)
