# Requirements: Local task preparation

Orbit prepares approved work for models that need smaller assignments and
less context. A preparation run produces a reviewable rune ledger and explicit
worker packets, while preserving the original outcomes and useful partial work.
Prepared work executes through Orbit's existing agent adapters.

## Non-goals

- Authoring or changing product requirements and design decisions.
- Model-native delegation, a new inference/tool loop or concurrent edits to one worktree.
- Automatically approving generated plans or declaring original owner acceptance complete.
- Model downloads, server lifecycle management or a claim that size limits guarantee model success.

## 1. Prepare without losing work

**User Story:** As an operator, I want a separate local execution plan, so that decomposition cannot erase approved scope or partial implementation.

1. <a name="1.1"></a>WHEN given a source rune ledger and approved references, the system SHALL create a separate candidate ledger, work packets and provenance record without modifying source tasks, source documents or code. Existing output SHALL be refused.  
2. <a name="1.2"></a>WHEN selecting source work, the system SHALL retain pending and in-progress outcomes, record completed work as context, and associate each candidate task with its source task identity.  
3. <a name="1.3"></a>The system SHALL derive immutable source-obligation identities before planning and account for each through executable steps or explicit deferred acceptance with its reason; original owner acceptance SHALL remain pending until separately verified.  

## 2. Describe work for the target

**User Story:** As a local worker, I want a concrete assignment, so that I can implement one behaviour without planning the whole feature.

1. <a name="2.1"></a>WHERE the target is `local`, each executable task SHALL name one behaviour, source obligations, bounded read context, allowed change paths, prerequisites and executable acceptance commands with expected outcomes.  
2. <a name="2.2"></a>The system SHALL reject candidate tasks exceeding the selected profile's declared file limit, omitting required fields or containing unmet manual, credential, browser or provider-decision gates as executable work.  
3. <a name="2.3"></a>WHERE tests-first checkpoints are used, the plan SHALL distinguish expected behavioural test failure from passing implementation acceptance, with explicit dependencies and one assignment per fresh session.  

## 3. Validate and approve the candidate

**User Story:** As a reviewer, I want machine checks and a visible coverage map, so that parseable but incomplete plans cannot silently enter execution.

1. <a name="3.1"></a>The system SHALL reject invalid planning output, unknown source identities, missing or duplicate task identities, unresolved dependencies/references, cycles, altered source obligations and incomplete structural coverage before publishing a candidate. Deferred acceptance SHALL NOT release an unmet prerequisite for executable work.  
2. <a name="3.2"></a>WHEN publishing a candidate, the system SHALL verify rune's reported success and read back its tasks, phases, references and dependencies; a zero subprocess exit alone SHALL NOT establish success.  
3. <a name="3.3"></a>The system SHALL require explicit approval of the candidate definition, packets, deferred acceptance and source revision/status snapshot before dispatch; changes to these inputs SHALL invalidate approval, while candidate progress backed by host attempt evidence SHALL preserve it. Unexplained status changes SHALL be rejected before selection. Semantic scope coverage SHALL be presented for review and SHALL NOT be claimed from structural checks alone.  

## 4. Dispatch discrete workers consistently

**User Story:** As an operator, I want the same work contract in local and variant runs, so that changing the model does not remove orchestration controls.

1. <a name="4.1"></a>WHEN executing an approved prepared ledger, both `orbit local` and `orbit run` single/variant paths SHALL select one ready assignment in the host, supply its packet and start a fresh worker if edits are required; they SHALL NOT delegate task selection or invoke the default commit prompt.  
2. <a name="4.2"></a>Each worker SHALL have a positive session timeout and finite invocation limit, run once without automatic redispatch after failure, and stop on halt, unverified/no-progress work, graph mutation, worker-written status changes or changes outside its declared paths relative to the attempt's dirty/untracked starting state. The host alone SHALL advance task statuses after acceptance, including automatic ancestor completion.  
3. <a name="4.3"></a>Parallel variants SHALL have independent worktrees, task state, prompts, counters and logs; failure in one variant SHALL NOT stop another. Workers within a variant SHALL execute sequentially under exclusive worktree execution ownership. Approved references SHALL resolve within a relocated checkout, and unsupported adapters SHALL be rejected before inference.  
4. <a name="4.4"></a>Cancellation SHALL terminate the worker process group on supported Unix platforms, retain diagnostics and record the failure without incrementing successful work. Plain ledgers SHALL retain their dispatch, prompt and retry policies while participating in the same worktree execution lock.  

## 5. Learn from attempts

**User Story:** As an operator, I want failures to become useful inputs, so that a restart builds on evidence rather than repeating blind work.

1. <a name="5.1"></a>Each attempt SHALL retain the exact assignment, source/candidate revision, model settings, stdout/stderr, before/after task state, changed paths, validation results and stop reason, including interrupted attempts. Unavailable required evidence storage SHALL prevent dispatch or completion.  
2. <a name="5.2"></a>WHEN resuming an unchanged approved ledger, the system SHALL preserve partial changes and completed assignments, then select unfinished work in a fresh session. WHEN requesting a revised plan, it SHALL preserve the failed plan and evidence as inputs and create a new candidate requiring approval.  
3. <a name="5.3"></a>Host validation SHALL enforce declared acceptance outcomes before completing tasks. RED acceptance SHALL identify the intended failing tests/assertions against recorded baseline results and reject environment, syntax/import or unrelated failures; nonzero exit alone SHALL NOT establish success. Validation timeout, failure or interruption SHALL preserve edits and unfinished acceptance for recovery before the next assignment.  
4. <a name="5.4"></a>WHEN a worker signals readiness for the current assignment, the host SHALL stop that worker and validate its work without waiting for a final prose response. Readiness SHALL NOT bypass acceptance or accept stale attempts; timeout, interruption or failure before a valid handoff SHALL remain failure.  

## 6. Prove execution separately from planning

**User Story:** As an operator, I want independent results, so that a generated ledger or comparison report is not mistaken for working code.

1. <a name="6.1"></a>Automated acceptance SHALL cover preparation, approval, two dependent workers, failed/stalled/canceled workers, preserved partial work and variant isolation without model inference.  
2. <a name="6.2"></a>Live evaluation SHALL record the hosted implementation comparison and local trials from a forked checkout, including source revision, actual models, retained improvements/failures and independent Makefile results. A failed or deferred original acceptance item SHALL remain visibly incomplete.  
