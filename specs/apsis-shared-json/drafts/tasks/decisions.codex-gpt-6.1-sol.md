# Task Draft Decisions: Apsis Shared JSON

This accompanies [the rune draft](draft.codex-gpt-6.1-sol.md), derived from the current [requirements](../../requirements.md), [design](../../design.md), [decision log](../../decision_log.md) through Q62, [questions](../design/questions.md) and [handoff](../../handoff.md). No other task draft or canonical task contents were read. The coordinator supplied the baseline fact that the canonical file contains 11 tasks and its task 1 remains in progress under `codex-gpt-6`, with two unchecked receives at `follow_test.go:1056` and `:1120`.

The draft is planning only. All 13 tasks are pending and unowned; none may be executed or used to claim implementation while the source/canonical-task freeze persists. Draft-local numbering, generated stable IDs and pending statuses do not replace canonical IDs, status or ownership. Later adoption must reconcile the plan with the protected canonical file rather than copy over it, preserving `codex-gpt-6`'s existing task 1 claim and every recorded status. No implementation, dependency change, canonical task mutation, claim, release or completion is part of this draft. The frontmatter records the freeze for readers; rune does not enforce it as an execution lock.

## Proposal T1: Use Thirteen Incremental Tasks in One Stream

**ID**: T1
**Date**: 2026-11-05
**Status**: proposed; draft planning only

### Context

The design spans provider extraction, a public schema, platform cache capability, list/open behavior, exec parsing and discovery. Most changes touch the same parser/session/CLI/web files. The baseline lint repair is unfinished and already owned elsewhere.

### Decision

Use one test-only baseline task followed by six adjacent red/green pairs: provider and first-known summaries (2–3), shared JSON/schema (4–5), platform fingerprints (6–7), existing-source listing/opening/cache (8–9), exec detection/parser/rendering (10–11), and exec discovery/identity/integration (12–13). Each task after the first declares the previous task as a blocker. Shared data types enter task 3 before projection tests need them; type additions are the TDD exemption, not separate setup tasks.

Task 11 implements working direct-file identity/project helpers with the parser slice, before task 13 reuses them for discovery. Task 10 tests web warning rendering from synthetic parsed values; exec source/ID route tests wait until task 12, avoiding a hidden dependency on discovery.

### Rationale

Each pair delivers recognizable behavior, keeps failing tests next to the code that passes them and integrates into an existing path. Schema tests use synthetic new-content ParseResults and SessionInfo values until the exec parser exists, so earlier schema work does not depend on later extraction or discovery. The plan retains `Lister`/`Resolver` and the long-lived cache rather than reintroducing a catalogue.

### Alternatives Considered

- **Independent parser, web and cache streams:** rejected because they edit shared representation, CLI, resolver and rendering code; parallel execution would create avoidable overlap.
- **Separate tasks for each field, fixture, registry entry and wiring call:** rejected because those edits form coherent behavior and would inflate the plan beyond the required test/implementation split.
- **One large implementation task per PR slice:** rejected because tests would not provide adjacent red/green checkpoints and early capability risks would be buried.

### Consequences

**Positive:** dependency order is explicit, all pairs are adjacent, and schema work can proceed without an exec parser.

**Negative:** the deliberately serial plan delays parser work behind the native capability gate. A later reviewed plan may separate genuinely independent work; this draft does not imply parallel execution is already safe.

---

## Proposal T2: Keep Capability and Policy Gates Visible

**ID**: T2
**Date**: 2026-11-05
**Status**: proposed; draft planning only

### Context

The design's platform cache contract needs native Windows evidence, and U1–U3 remain unanswered. The current authorization permits a draft and its review, while source and canonical tasks remain frozen. Existing canonical ownership must survive planning.

### Decision

Task 7 cannot complete until injected-adapter and native Windows replacement/rewrite tests establish the required identity/change fingerprint behavior. Task 8 is blocked by task 7, and task 9 explicitly retains the same admission dependency. Cross-compilation and mocks do not establish native capability. No native Windows runtime verification was performed during this draft.

Plan U1-A (omit Claude model extraction), U2-A (omit Kiro IDE provider extraction) and U3-A (test-only maintained schema validator) as recommendations, preserving their unanswered status. U3's dependency cannot be added during the current freeze. A different human answer requires revising affected tasks before implementation.

Preserve accepted policies: separate LM Studio spec; provider only with no locality flag; no prompt-file reads; global rollout precedence before project filtering; labelled filesystem ordering fallback; approved directory skips with explicitly configured roots still scanned. Q62 limits global candidate selection to suppression, opening and all-projects deduplication; legacy project rollout rows retain per-file cwd filtering.

### Rationale

Neither a plan nor API availability proves storage behavior, user approval or authority to change owned work. Explicit gates keep later code from relying on evidence that does not exist.

### Alternatives Considered

- **Use size/mtime when Windows identity metadata is unavailable:** rejected because it weakens 5.9 and misses equal-size preserved-mtime changes.
- **Treat U1–U3 recommendations as accepted decisions:** rejected because the question file explicitly preserves human choices.
- **Replace or claim the canonical baseline task while drafting:** rejected because it is frozen and already owned by another agent.

### Consequences

**Positive:** review can assess scope and evidence without accidental implementation admission or ownership transfer.

**Negative:** native Windows access and any material policy changes remain real future dependencies; the draft cannot remove them by marking tasks complete.

---

## Coverage and Risk Audit

Criterion links below map to the test/implementation pairs in the rune draft. The baseline task also protects [5.1](../../requirements.md#5.1).

| Acceptance criterion | Tasks |
|---|---|
| [1.1](../../requirements.md#1.1) | 2–5, 8–13 |
| [1.2](../../requirements.md#1.2) | 4–5, 8–13 |
| [1.3](../../requirements.md#1.3) | 4–5 |
| [1.4](../../requirements.md#1.4) | 4–5, 8–13 |
| [1.5](../../requirements.md#1.5) | 4–5, 10–11 |
| [1.6](../../requirements.md#1.6) | 2–5, 8–11 |
| [2.1](../../requirements.md#2.1) | 10–13 |
| [2.2](../../requirements.md#2.2) | 10–11 |
| [2.3](../../requirements.md#2.3) | 10–11 |
| [2.4](../../requirements.md#2.4) | 10–11 |
| [2.5](../../requirements.md#2.5) | 12–13 |
| [2.6](../../requirements.md#2.6) | 10–13 |
| [2.7](../../requirements.md#2.7) | 12–13 |
| [2.8](../../requirements.md#2.8) | 10–13 |
| [2.9](../../requirements.md#2.9) | 10–11 |
| [2.10](../../requirements.md#2.10) | 10–13 |
| [2.11](../../requirements.md#2.11) | 10–13 |
| [3.1](../../requirements.md#3.1) | 2–3 |
| [4.1](../../requirements.md#4.1) | 8–9, 12–13 |
| [4.2](../../requirements.md#4.2) | 8–9, 12–13 |
| [4.3](../../requirements.md#4.3) | 4–5, 8–9, 12–13 |
| [4.4](../../requirements.md#4.4) | 8–9, 12–13 |
| [5.1](../../requirements.md#5.1) | 1–13 |
| [5.2](../../requirements.md#5.2) | 10–13 |
| [5.3](../../requirements.md#5.3) | 4–5, 10–13 |
| [5.4](../../requirements.md#5.4) | 12–13 |
| [5.5](../../requirements.md#5.5) | 12–13 |
| [5.6](../../requirements.md#5.6) | 10–13 |
| [5.7](../../requirements.md#5.7) | 8–13 |
| [5.8](../../requirements.md#5.8) | 8–9, 12–13 |
| [5.9](../../requirements.md#5.9) | 6–9, 12–13 |

| Design risk or assumption | Verification and dependent work |
|---|---|
| Large configured roots | Task 12 writes the generated 10,000-file benchmark before task 13's integration; evaluate the stated directory-entry caching fallback only if needed. |
| Kiro WAL writes | Task 8 tests two writes on one WAL connection before task 9 relies on cached DB discovery. A failed stat assumption prevents a claimed 5.9 pass. |
| Windows identity/change metadata | Tasks 6–7 establish adapter and native capability before cache work in 8–9 or discovery in 12–13. Native execution remains unverified. |
| Cold Kiro IDE reads | Task 8 adds a generated cold multi-source benchmark; task 9 evaluates whole-file extraction versus the design's narrow streaming decoder. |
| Duplicate web_search id keys | Task 10 tests a pinned-definition fixture before task 11 relies on last-key decoding for lifecycle identity. |

The plan explicitly includes complete-line probing across false Kiro prefixes, duplicate selected-rollout winners, Q62 project-row compatibility, legacy timestamp/tie behavior, missing-ID generic fallback, header-only metadata, both HTML passes, raw exec BOM handling and open/structurally closed schema checks. Property tests cover lifecycle, ordering and stable identifiers. All fixtures are synthetic; no private transcript or server-log content was read for this draft.

Scope excludes LM Studio parsing, prompt files, locality inference, OpenCode transcripts, exec/shared-JSON follow support, new non-exec token extraction, orbit invocation changes and JSON HTTP endpoints. Makefile checks are planned implementation validations; no builds, implementation tests, native Windows runs or schema-output conformance runs were performed while creating these planning documents.
