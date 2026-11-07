# Tasks Critique: Merged Shared JSON Plan

SAME-MODEL fresh-context, non-author evidence. This is neither independent-model review nor human approval. Reviewed the 22:27:26Z merge, with task 1's separately logged 22:28:08Z completion. Current task-file SHA-256: `15fc67c939a902a572da07fc65afacd85f54da2660a38924d082146547012735`. No peer review or other current review artifact was read.

## Ranked Findings

**Critical:** none found. **Major:** none found. **Minor:** none found in the reviewed planning scope. This is a feasibility review, not evidence that the proposed implementation passes its tests.

## Checks and Boundaries

- [Canonical tasks](../../tasks.md) parse through `rune list --format json`: 16 parents, 46 total items, four PR phases, 15 adjacent red/green pairs. All 58 explicit dependency edges resolve, and the graph remains acyclic when parent completion also depends on children. Red tasks specify observable failures; green tasks implement those behaviours. Task 1 is a justified test-only lint repair.
- [The preserved canonical snapshot](tasks.canonical-before-merge-r1.md) matches its recorded SHA-256. Numbered/stable identities 1–11 survive. Tasks 2–11 remain pending and unowned; task 1 retains owner `codex-gpt-6`, with its completed status supported by the coordinator's later verification record. [The scope map](merge-map.codex-gpt-6.1-sol.md) explicitly relocates the original scopes rather than silently dropping them.
- All 31 [acceptance criteria](../../requirements.md) have concrete verification work. Tasks 2–4 cover parsed metadata, provider parity, schema and CLI output; 5–8 cover scoped listings, ordering and list/open round trips; 9–14 cover lifecycle, generic payloads, every renderer, detection, direct conversion, discovery and source opening; 15–16 cover fingerprints and unchanged-refresh reads.
- All five [design risks](../../design.md#risks-and-assumptions) have verification: traversal dimensions/measurements in 13.1; WAL changes in 15.1; native Windows metadata in 15.2; cold Kiro IDE measurements in 16.1; duplicate web-search keys in 9.1. Timing is investigative evidence, while zero unchanged transcript-body bytes remains mandatory.
- Cross-stream file ownership is feasible. Sessions/CLI work follows shared-session conversion; renderer/projection work waits for listing projection; direct exec conversion waits for scope wiring and transcript work. Stream 3 owns isolated `internal/fileidentity` files, and 15.2 waits for 3.2 before changing module files. Full-suite checks still need coordination while another stream intentionally holds failing red tests.
- The schema admission is technically explicit: 3.1/3.2 publish all known v1 branches immediately, exclude known discriminators from the unknown-type extension branch, and retain the exact initial schema for later producers. Open arbitrary JSON and structurally closed test validation are distinguished. U3 remains an unanswered human question; the test-only validator is a reversible agent decision.
- Human-approved rollout precedence, filesystem-time labelling and recursive skips remain intact. U1/U2 keep their current omissions; no human design/tasks approval is inferred. Header-only Codex success is restricted to shared JSON, preserving the current parser's default error.
- [Native Windows prerequisites](../../prerequisites.md) explicitly prevent completing 15.2 or starting 16 without evidence tied to the implementation under review. The graph does not impose that gate on PRs 1–3. Cross-compilation, mocks and a CI change cannot substitute for the native run.

## Actionable Unresolved Questions

No additional product or design answer is required before admitting PR 1 after both current reviews are reconciled.

1. Before completing 15.2 or starting 16, which native Windows host/filesystem will supply the required run, and which implementation revision or file digests will its recorded results cover? If none is available, keep those tasks incomplete as specified.
2. If the WAL test fails, can the measured fallback discover queries preserve zero conversation-body reads under 5.9? Record that audit before admitting cache consumers; otherwise obtain a requirements decision rather than weakening the assertion.
