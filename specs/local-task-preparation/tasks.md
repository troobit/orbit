---
references:
    - specs/local-task-preparation/requirements.md
    - specs/local-task-preparation/design.md
    - specs/local-task-preparation/decision_log.md
---
# Local task preparation

## Phase 1: Structured work contract

- [ ] 1. Validate source obligations and local worker plans <!-- id:5xo7plv -->
  - Scope and contracts: specs/local-task-preparation/design.md. Use adjacent RED/GREEN children; --leaf gives separate sessions. No live model or owner acceptance belongs in this implementation ledger.
  - Stream: 1
  - Requirements: [1.1](specs/local-task-preparation/requirements.md#1.1), [1.2](specs/local-task-preparation/requirements.md#1.2), [1.3](specs/local-task-preparation/requirements.md#1.3), [2.1](specs/local-task-preparation/requirements.md#2.1), [2.2](specs/local-task-preparation/requirements.md#2.2), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.1](specs/local-task-preparation/requirements.md#3.1)
  - [ ] 1.1. RED: Validate source obligations and local worker plans <!-- id:5xo7plw -->
    - Create behaviour tests for internal/workplan with source rune snapshots, immutable host obligation keys, completed/in-progress source history, dependency closure and deferred blocks_items. Include unknown/duplicate IDs, missing coverage, altered obligation text, invalid references/anchors, cycles, unsafe/symlink paths and the three-file target limit.
    - Use rapid properties for ID/order/dependency mutations and source preservation. Test RED expectations require named failures/assertions and a reviewed verifier, not arbitrary nonzero output. Acceptance: make test-local must include and fail these new assertions against missing behaviour, without inference.
    - Add internal/workplan to make test-local in this RED assignment so the new behaviour assertions actually run; keep the harness buildable with only necessary API/type scaffolding.
    - Stream: 1
    - Requirements: [1.1](specs/local-task-preparation/requirements.md#1.1), [1.2](specs/local-task-preparation/requirements.md#1.2), [1.3](specs/local-task-preparation/requirements.md#1.3), [2.1](specs/local-task-preparation/requirements.md#2.1), [2.2](specs/local-task-preparation/requirements.md#2.2), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.1](specs/local-task-preparation/requirements.md#3.1)
  - [ ] 1.2. GREEN: Validate source obligations and local worker plans <!-- id:5xo7plx -->
    - Implement the versioned plan/profile/packet schema and deterministic validator under internal/workplan. Derive obligation keys before planning, join numeric rune identities with canonical stable markers, and use revision-scoped identities only for legacy markerless sources.
    - Enforce exact path scopes, source-dependency closure, inherited constraints and explicit deferrals. Do not mark original source acceptance complete or build a general profile framework. Retain the RED assignment package wiring in make test-local. Acceptance: unchanged red cases and make test-local pass.
    - Blocked-by: 5xo7plw (RED: Validate source obligations and local worker plans)
    - Stream: 1
    - Requirements: [1.1](specs/local-task-preparation/requirements.md#1.1), [1.2](specs/local-task-preparation/requirements.md#1.2), [1.3](specs/local-task-preparation/requirements.md#1.3), [2.1](specs/local-task-preparation/requirements.md#2.1), [2.2](specs/local-task-preparation/requirements.md#2.2), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.1](specs/local-task-preparation/requirements.md#3.1)

## Phase 2: Rune candidate and approval

- [ ] 2. Materialise and approve portable rune candidates <!-- id:5xo7ply -->
  - Scope and contracts: specs/local-task-preparation/design.md. Use adjacent RED/GREEN children; --leaf gives separate sessions. No live model or owner acceptance belongs in this implementation ledger.
  - Blocked-by: 5xo7plv (Validate source obligations and local worker plans)
  - Stream: 1
  - Requirements: [1.1](specs/local-task-preparation/requirements.md#1.1), [1.2](specs/local-task-preparation/requirements.md#1.2), [1.3](specs/local-task-preparation/requirements.md#1.3), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.1](specs/local-task-preparation/requirements.md#3.1), [3.2](specs/local-task-preparation/requirements.md#3.2), [3.3](specs/local-task-preparation/requirements.md#3.3)
  - [ ] 2.1. RED: Materialise and approve portable rune candidates <!-- id:5xo7plz -->
    - Test cmd/orbit prepare --from-plan using temporary repositories with spaces and real rune. Assert atomic output publication, one assignment per phase, source immutability, new candidate stable IDs, complete lineage, resolving references, candidate .gitignore and refusal to overwrite output.
    - Test JSON success:false with exit zero, readback mismatches, unsupported manifests, source-status drift, candidate/packet/verifier mutation and approval invalidation. Status-only host-backed progress remains compatible. No model invocation. Acceptance: new assertions fail under make test-local.
    - Blocked-by: 5xo7plv (Validate source obligations and local worker plans)
    - Stream: 1
    - Requirements: [1.1](specs/local-task-preparation/requirements.md#1.1), [1.2](specs/local-task-preparation/requirements.md#1.2), [1.3](specs/local-task-preparation/requirements.md#1.3), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.1](specs/local-task-preparation/requirements.md#3.1), [3.2](specs/local-task-preparation/requirements.md#3.2), [3.3](specs/local-task-preparation/requirements.md#3.3)
  - [ ] 2.2. GREEN: Materialise and approve portable rune candidates <!-- id:5xo7pm0 -->
    - Implement cmd/orbit/prepare.go import and approve paths, register command routing, and add internal/workplan rune generation/manifest/review/approval helpers. Stage in a sibling temporary directory and rename only after validated readback; preserve failed evidence outside unpublished output.
    - Add the orbit-prepared frontmatter marker via rune and use repository-relative hashes/paths that rebase cleanly into forks/worktrees. Approval pins source statuses, candidate definition, packet and verifier bytes. Do not introduce an execution task that asks a worker for approval. Acceptance: unchanged tests, rune parsing and make test-local pass.
    - Blocked-by: 5xo7plz (RED: Materialise and approve portable rune candidates)
    - Stream: 1
    - Requirements: [1.1](specs/local-task-preparation/requirements.md#1.1), [1.2](specs/local-task-preparation/requirements.md#1.2), [1.3](specs/local-task-preparation/requirements.md#1.3), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.1](specs/local-task-preparation/requirements.md#3.1), [3.2](specs/local-task-preparation/requirements.md#3.2), [3.3](specs/local-task-preparation/requirements.md#3.3)

## Phase 3: Host-controlled workers

- [ ] 3. Execute prepared local and single-run packets with host-owned completion <!-- id:5xo7pm1 -->
  - Scope and contracts: specs/local-task-preparation/design.md. Use adjacent RED/GREEN children; --leaf gives separate sessions. No live model or owner acceptance belongs in this implementation ledger.
  - Blocked-by: 5xo7ply (Materialise and approve portable rune candidates)
  - Stream: 1
  - Requirements: [2.1](specs/local-task-preparation/requirements.md#2.1), [2.2](specs/local-task-preparation/requirements.md#2.2), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.3](specs/local-task-preparation/requirements.md#3.3), [4.1](specs/local-task-preparation/requirements.md#4.1), [4.2](specs/local-task-preparation/requirements.md#4.2), [4.4](specs/local-task-preparation/requirements.md#4.4), [5.1](specs/local-task-preparation/requirements.md#5.1), [5.2](specs/local-task-preparation/requirements.md#5.2), [5.3](specs/local-task-preparation/requirements.md#5.3), [5.4](specs/local-task-preparation/requirements.md#5.4)
  - [ ] 3.1. RED: Execute prepared local and single-run packets with host-owned completion <!-- id:5xo7pm2 -->
    - Use real-rune ledgers and fake Codex subprocesses to test two dependent packets, host-only progress/complete, passing retained code without new edits, no-progress failure, model status tampering, graph mutation, reopened tasks and declared/undeclared changes to dirty and untracked files.
    - Cover host passing/RED checks, exact failed-test verifier evidence, environment/unrelated failures, timeouts, missing evidence storage and crashes at each journal stage. Assert statuses cannot skip unaccepted work on restart, and partial code is retained. Test conflicting --phase/model selection before logs and ordinary-ledger compatibility. Acceptance: meaningful new cases fail with make test-local.
    - Test current/stale readiness receipts, natural exits, timeout/receipt races, deliberate handoff cancellation and separate raw-result/host-acceptance accounting. Retained passing work must complete without new inference.
    - Blocked-by: 5xo7ply (Materialise and approve portable rune candidates)
    - Stream: 1
    - Requirements: [2.1](specs/local-task-preparation/requirements.md#2.1), [2.2](specs/local-task-preparation/requirements.md#2.2), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.3](specs/local-task-preparation/requirements.md#3.3), [4.1](specs/local-task-preparation/requirements.md#4.1), [4.2](specs/local-task-preparation/requirements.md#4.2), [4.4](specs/local-task-preparation/requirements.md#4.4), [5.1](specs/local-task-preparation/requirements.md#5.1), [5.2](specs/local-task-preparation/requirements.md#5.2), [5.3](specs/local-task-preparation/requirements.md#5.3), [5.4](specs/local-task-preparation/requirements.md#5.4)
  - [ ] 3.2. GREEN: Execute prepared local and single-run packets with host-owned completion <!-- id:5xo7pm3 -->
    - Extract per-worktree selection/prompt/graph operations into internal/orbit/task_worker.go, with internal/workplan packet loading/check execution/journals. Route approved prepared ledgers from localCommand and runSingle; keep ordinary local prompts/status ownership and hosted defaults intact.
    - Host runs prerequisites, marks progress, invokes Agent.Run once, checks path/graph integrity and acceptance, persists evidence, then completes with rune. Recover unfinished journals and reconcile every status against initial/accepted state before selection. No Resume, automatic retry, model-owned rune commands or rollback of source edits.
    - Acquire the shared Git-worktree OS lock at implementation entry points, including ordinary local/single paths. Bound workers/checks, keep required logs fatal and registry best effort. Acceptance: unchanged tests plus make test-local, make test and make lint pass.
    - Implement/use the prepared readiness receipt protocol and pre-inference acceptance check from design.md Decision 4. Stop the worker group before host validation; retain raw exits and accept only independently verified work.
    - Blocked-by: 5xo7pm2 (RED: Execute prepared local and single-run packets with host-owned completion)
    - Stream: 1
    - Requirements: [2.1](specs/local-task-preparation/requirements.md#2.1), [2.2](specs/local-task-preparation/requirements.md#2.2), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.3](specs/local-task-preparation/requirements.md#3.3), [4.1](specs/local-task-preparation/requirements.md#4.1), [4.2](specs/local-task-preparation/requirements.md#4.2), [4.4](specs/local-task-preparation/requirements.md#4.4), [5.1](specs/local-task-preparation/requirements.md#5.1), [5.2](specs/local-task-preparation/requirements.md#5.2), [5.3](specs/local-task-preparation/requirements.md#5.3), [5.4](specs/local-task-preparation/requirements.md#5.4)

## Phase 4: Variant parity

- [ ] 4. Run prepared variants through isolated workers and supported adapters <!-- id:5xo7pm4 -->
  - Scope and contracts: specs/local-task-preparation/design.md. Use adjacent RED/GREEN children; --leaf gives separate sessions. No live model or owner acceptance belongs in this implementation ledger.
  - Blocked-by: 5xo7pm1 (Execute prepared local and single-run packets with host-owned completion)
  - Stream: 1
  - Requirements: [3.3](specs/local-task-preparation/requirements.md#3.3), [4.1](specs/local-task-preparation/requirements.md#4.1), [4.2](specs/local-task-preparation/requirements.md#4.2), [4.3](specs/local-task-preparation/requirements.md#4.3), [4.4](specs/local-task-preparation/requirements.md#4.4), [5.1](specs/local-task-preparation/requirements.md#5.1), [5.2](specs/local-task-preparation/requirements.md#5.2), [5.3](specs/local-task-preparation/requirements.md#5.3), [6.1](specs/local-task-preparation/requirements.md#6.1), [5.4](specs/local-task-preparation/requirements.md#5.4)
  - [ ] 4.1. RED: Run prepared variants through isolated workers and supported adapters <!-- id:5xo7pm5 -->
    - Test prepared metadata validation before worktrees/inference and path rebasing inside two parallel variants. One fake worker fails/stalls while its sibling completes; assert independent snapshots, counters, prompts, task states, journals and existing terminal statuses.
    - Add real subprocess tests for Claude live-writer forwarding, FinalOutput halt detection and Unix process-group cancellation using shared executor fixtures. Reject unsupported adapters, unavailable logs, unapproved plans, custom AI prompts/commands and unexplained copied completion. Preserve ordinary hosted retry/prompt behaviour. Acceptance: new cases fail in make test-local.
    - Add internal/agents/claudecode to make test-local in this RED assignment so the new adapter/cancellation assertions actually run.
    - Test current/stale readiness receipts, natural exits, timeout/receipt races, deliberate handoff cancellation and separate raw-result/host-acceptance accounting. Retained passing work must complete without new inference.
    - Blocked-by: 5xo7pm1 (Execute prepared local and single-run packets with host-owned completion)
    - Stream: 1
    - Requirements: [3.3](specs/local-task-preparation/requirements.md#3.3), [4.1](specs/local-task-preparation/requirements.md#4.1), [4.2](specs/local-task-preparation/requirements.md#4.2), [4.3](specs/local-task-preparation/requirements.md#4.3), [4.4](specs/local-task-preparation/requirements.md#4.4), [5.1](specs/local-task-preparation/requirements.md#5.1), [5.2](specs/local-task-preparation/requirements.md#5.2), [5.3](specs/local-task-preparation/requirements.md#5.3), [6.1](specs/local-task-preparation/requirements.md#6.1), [5.4](specs/local-task-preparation/requirements.md#5.4)
  - [ ] 4.2. GREEN: Run prepared variants through isolated workers and supported adapters <!-- id:5xo7pm6 -->
    - Route prepared runVariant through a worker instance per worktree and rebase all candidate/source paths locally. Apply common worktree locks and host acceptance before phase success accounting; skip default commit/AI prompts and retain explicit host shell hooks.
    - Forward stdout/stderr/ProcessGroup through internal/agents/claudecode/agent.go and populate FinalOutput from its parsed assistant result. Initial prepared adapters are Codex/Claude; fail others before dispatch. Reuse bounded worker checks rather than adding a second generic variant stall guard. Acceptance: unchanged tests, make test-local, make test and make lint pass.
    - Implement/use the prepared readiness receipt protocol and pre-inference acceptance check from design.md Decision 4. Stop the worker group before host validation; retain raw exits and accept only independently verified work.
    - Blocked-by: 5xo7pm5 (RED: Run prepared variants through isolated workers and supported adapters)
    - Stream: 1
    - Requirements: [3.3](specs/local-task-preparation/requirements.md#3.3), [4.1](specs/local-task-preparation/requirements.md#4.1), [4.2](specs/local-task-preparation/requirements.md#4.2), [4.3](specs/local-task-preparation/requirements.md#4.3), [4.4](specs/local-task-preparation/requirements.md#4.4), [5.1](specs/local-task-preparation/requirements.md#5.1), [5.2](specs/local-task-preparation/requirements.md#5.2), [5.3](specs/local-task-preparation/requirements.md#5.3), [6.1](specs/local-task-preparation/requirements.md#6.1), [5.4](specs/local-task-preparation/requirements.md#5.4)

## Phase 5: Planner and retained feedback

- [ ] 5. Generate candidate plans with a read-only Codex planner <!-- id:5xo7pm7 -->
  - Scope and contracts: specs/local-task-preparation/design.md. Use adjacent RED/GREEN children; --leaf gives separate sessions. No live model or owner acceptance belongs in this implementation ledger.
  - Blocked-by: 5xo7pm4 (Run prepared variants through isolated workers and supported adapters)
  - Stream: 1
  - Requirements: [1.1](specs/local-task-preparation/requirements.md#1.1), [1.2](specs/local-task-preparation/requirements.md#1.2), [1.3](specs/local-task-preparation/requirements.md#1.3), [2.1](specs/local-task-preparation/requirements.md#2.1), [2.2](specs/local-task-preparation/requirements.md#2.2), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.1](specs/local-task-preparation/requirements.md#3.1), [3.3](specs/local-task-preparation/requirements.md#3.3), [5.1](specs/local-task-preparation/requirements.md#5.1), [5.2](specs/local-task-preparation/requirements.md#5.2)
  - [ ] 5.1. RED: Generate candidate plans with a read-only Codex planner <!-- id:5xo7pm8 -->
    - Test planner configuration and effective argv with fake Codex: allowlisted provider/budget options only, enforced read-only sandbox, disabled hooks/apps/search/delegation, unsupported alias/adapter rejection, source-change detection and one bounded call per selected source task.
    - Test malformed/truncated plans, false/missing coverage, separate planner output and dispatch ledgers, timeout diagnostics, failed output retention, and replanning into a fresh directory with old attempts/partial changes as context. The import path must need no model. Acceptance: new cases fail with make test-local.
    - Blocked-by: 5xo7pm4 (Run prepared variants through isolated workers and supported adapters)
    - Stream: 1
    - Requirements: [1.1](specs/local-task-preparation/requirements.md#1.1), [1.2](specs/local-task-preparation/requirements.md#1.2), [1.3](specs/local-task-preparation/requirements.md#1.3), [2.1](specs/local-task-preparation/requirements.md#2.1), [2.2](specs/local-task-preparation/requirements.md#2.2), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.1](specs/local-task-preparation/requirements.md#3.1), [3.3](specs/local-task-preparation/requirements.md#3.3), [5.1](specs/local-task-preparation/requirements.md#5.1), [5.2](specs/local-task-preparation/requirements.md#5.2)
  - [ ] 5.2. GREEN: Generate candidate plans with a read-only Codex planner <!-- id:5xo7pm9 -->
    - Add --agent planning to prepare using existing agents.Agent, one source-task prompt at a time, with host-owned obligation data, bounded relevant context and prior failure summaries. Sanitize effective planner config rather than appending arbitrary alias extra-args or inheriting AutoApprove.
    - Validate all returned data through the phase-1 contract and phase-2 publication path; no direct model source/ledger writes, inferred approval, automatic repair loop or native subagents. Keep positive planner limits and failed artifacts. Acceptance: unchanged tests, make test-local, make test and make lint pass; document real sandbox verification as a live prerequisite.
    - Blocked-by: 5xo7pm8 (RED: Generate candidate plans with a read-only Codex planner)
    - Stream: 1
    - Requirements: [1.1](specs/local-task-preparation/requirements.md#1.1), [1.2](specs/local-task-preparation/requirements.md#1.2), [1.3](specs/local-task-preparation/requirements.md#1.3), [2.1](specs/local-task-preparation/requirements.md#2.1), [2.2](specs/local-task-preparation/requirements.md#2.2), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.1](specs/local-task-preparation/requirements.md#3.1), [3.3](specs/local-task-preparation/requirements.md#3.3), [5.1](specs/local-task-preparation/requirements.md#5.1), [5.2](specs/local-task-preparation/requirements.md#5.2)

## Phase 6: Reproducible acceptance

- [ ] 6. Provide an offline fixture and exercise the full preparation handoff <!-- id:5xo7pma -->
  - Scope and contracts: specs/local-task-preparation/design.md. Use adjacent RED/GREEN children; --leaf gives separate sessions. No live model or owner acceptance belongs in this implementation ledger.
  - Blocked-by: 5xo7pm7 (Generate candidate plans with a read-only Codex planner)
  - Stream: 1
  - Requirements: [1.1](specs/local-task-preparation/requirements.md#1.1), [1.3](specs/local-task-preparation/requirements.md#1.3), [2.1](specs/local-task-preparation/requirements.md#2.1), [2.2](specs/local-task-preparation/requirements.md#2.2), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.1](specs/local-task-preparation/requirements.md#3.1), [3.2](specs/local-task-preparation/requirements.md#3.2), [3.3](specs/local-task-preparation/requirements.md#3.3), [4.1](specs/local-task-preparation/requirements.md#4.1), [4.2](specs/local-task-preparation/requirements.md#4.2), [4.3](specs/local-task-preparation/requirements.md#4.3), [4.4](specs/local-task-preparation/requirements.md#4.4), [5.1](specs/local-task-preparation/requirements.md#5.1), [5.2](specs/local-task-preparation/requirements.md#5.2), [5.3](specs/local-task-preparation/requirements.md#5.3), [6.1](specs/local-task-preparation/requirements.md#6.1), [6.2](specs/local-task-preparation/requirements.md#6.2), [5.4](specs/local-task-preparation/requirements.md#5.4)
  - [ ] 6.1. RED: Provide an offline fixture and exercise the full preparation handoff <!-- id:5xo7pmb -->
    - Add an automated CLI integration fixture with a small approved source spec/ledger, two dependent behaviours, a trusted structured RED verifier and fake worker executables. Exercise import, review, approval, local/single/parallel execution, failure/resume and relocation without model inference.
    - Assert actual failing baseline/RED results, exact coverage/deferred acceptance, readonly source, host-owned transitions, accepted final behaviour, cancellation cleanup and artifact isolation. A complete status or generated report alone cannot pass. Acceptance: new end-to-end assertions fail under make test-local.
    - Test current/stale readiness receipts, natural exits, timeout/receipt races, deliberate handoff cancellation and separate raw-result/host-acceptance accounting. Retained passing work must complete without new inference.
    - Blocked-by: 5xo7pm7 (Generate candidate plans with a read-only Codex planner)
    - Stream: 1
    - Requirements: [1.1](specs/local-task-preparation/requirements.md#1.1), [1.3](specs/local-task-preparation/requirements.md#1.3), [2.1](specs/local-task-preparation/requirements.md#2.1), [2.2](specs/local-task-preparation/requirements.md#2.2), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.1](specs/local-task-preparation/requirements.md#3.1), [3.2](specs/local-task-preparation/requirements.md#3.2), [3.3](specs/local-task-preparation/requirements.md#3.3), [4.1](specs/local-task-preparation/requirements.md#4.1), [4.2](specs/local-task-preparation/requirements.md#4.2), [4.3](specs/local-task-preparation/requirements.md#4.3), [4.4](specs/local-task-preparation/requirements.md#4.4), [5.1](specs/local-task-preparation/requirements.md#5.1), [5.2](specs/local-task-preparation/requirements.md#5.2), [5.3](specs/local-task-preparation/requirements.md#5.3), [6.1](specs/local-task-preparation/requirements.md#6.1), [6.2](specs/local-task-preparation/requirements.md#6.2), [5.4](specs/local-task-preparation/requirements.md#5.4)
  - [ ] 6.2. GREEN: Provide an offline fixture and exercise the full preparation handoff <!-- id:5xo7pmc -->
    - Implement the reusable fixture/configuration generation and CLI test support, including a reviewed verifier emitting the documented failure schema. Cover both a supplied regression template and an implementation packet so scaffolding is recorded rather than confused with independent planning.
    - Wire all checks into existing Makefile targets; add a concise runbook and update relevant agent notes. Include exact hosted codex/opus and forked local-alias commands, required result commits/judge, fresh-reset/evidence handling, a real read-only planner sandbox probe and the authorised Sanarte trial procedure. Do not run models or claim live success from this coding task.
    - Acceptance: unchanged integration tests, rune parsing, script syntax where relevant, make test-local, make test, make lint and make build pass. Live outcomes are recorded separately in evaluation.md.
    - Implement/use the prepared readiness receipt protocol and pre-inference acceptance check from design.md Decision 4. Stop the worker group before host validation; retain raw exits and accept only independently verified work.
    - Blocked-by: 5xo7pmb (RED: Provide an offline fixture and exercise the full preparation handoff)
    - Stream: 1
    - Requirements: [1.1](specs/local-task-preparation/requirements.md#1.1), [1.3](specs/local-task-preparation/requirements.md#1.3), [2.1](specs/local-task-preparation/requirements.md#2.1), [2.2](specs/local-task-preparation/requirements.md#2.2), [2.3](specs/local-task-preparation/requirements.md#2.3), [3.1](specs/local-task-preparation/requirements.md#3.1), [3.2](specs/local-task-preparation/requirements.md#3.2), [3.3](specs/local-task-preparation/requirements.md#3.3), [4.1](specs/local-task-preparation/requirements.md#4.1), [4.2](specs/local-task-preparation/requirements.md#4.2), [4.3](specs/local-task-preparation/requirements.md#4.3), [4.4](specs/local-task-preparation/requirements.md#4.4), [5.1](specs/local-task-preparation/requirements.md#5.1), [5.2](specs/local-task-preparation/requirements.md#5.2), [5.3](specs/local-task-preparation/requirements.md#5.3), [6.1](specs/local-task-preparation/requirements.md#6.1), [6.2](specs/local-task-preparation/requirements.md#6.2), [5.4](specs/local-task-preparation/requirements.md#5.4)
