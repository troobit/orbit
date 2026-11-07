# Canonical Task Reconciliation

The [pre-merge canonical file](tasks.canonical-before-merge-r1.md) is byte-exact: SHA-256 `98447a2e556a81ee825dc550826ce1afadc769a863ce6d7d64d1d38b5edfa72c`. Both original drafts and every review remain unchanged. The merged [canonical plan](../../tasks.md) uses rune updates/additions, preserving all original numbered/stable IDs 1–11 and their statuses/owners. Task 1 remains in progress under `codex-gpt-6`; the merge does not claim or complete it. New parents 12–16 and children have rune-generated stable IDs.

## Original Scope Mapping

| Original task / stable ID | Original scope retained in merged task(s) |
|---|---|
| 1 / 9kwhro4 | 1: follower shutdown assertions and baseline checks; count corrected from three remaining receives to two; separately coordinated repair does not transfer ownership. |
| 2 / 9kwhro5 | 2.1 first-known metadata; 3.1 session/schema contract; 4.1 CLI/help/follow/raw-JSON/empty cases. |
| 3 / 9kwhro6 | 2.2 parsed metadata; 3.2 projection/schema; 4.2 CLI input context/dispatch. Placement corrected to internal/sharedjson and the exact published schema path. |
| 4 / 9kwhro7 | 2.1 provider/renderer tests; 3.1 JSON provider tests. Grouped/deferred headers are tested as unchanged, resolving the earlier scope against Decision 16. |
| 5 / 9kwhro8 | 2.2 provider state/modelLabel; 3.2 JSON projection; 4.2 slice checks. The proposed group/provider change is replaced by the reviewed existing-model-header rule. |
| 6 / 9kwhro9 | 5.1 ordering/metadata; 6.1 listing schema; 7.1 source resolution; 8.1 CLI/web round trips; 13.1 traversal benchmark; 15.1 WAL/platform risk; 16.1 zero-body reads/changes/concurrency/cold benchmark. |
| 7 / 9kwhroa | 5.2 row metadata/options; 6.2 JSON listing; 7.2 all-project resolver; 8.2 flags/web; 16.2 cache. The rejected Catalog/private-locator architecture is replaced by the reviewed Lister/Resolver design with companion invalidation, warnings and fresh slices. |
| 8 / 9kwhrob | 9.1 synthetic lifecycle/provenance/usage/generic payload tests; 10.1 all rendering/projection tests; 11.1 complete-line detection; 12.1 forced/pre-thread/raw/follow tests. |
| 9 / 9kwhroc | 9.2 exec types/parser/dispatch; 10.2 renderers/projection; 11.2 detection; 12.2 direct/raw/follow; 14.2 indirect opening/follow. |
| 10 / 9kwhrod | 12.1 identity/project/direct-file tests; 13.1 settings/containment/suppression/conflict/benchmark tests; 14.1 latest/follow/round trips; 16.1 unchanged refresh and nested-repository changes. |
| 11 / 9kwhroe | 12.2 identity/repository helpers; 13.2 discovery/settings/suppression; 14.2 source opening/badges/warnings; 16.2 cache/integration checks. |

No original behaviour/testing scope is dropped. Corrections to previously proposed mechanisms are explicit above; human-approved policies remain unchanged.

## Draft Choices and Ownership

Keep Claude's four PR boundaries and smaller paired units. Keep Codex's full-contract synthetic schema tests, injected digest-conflict seam and source-specific regressions. Replace staged schema enums with Decision 25, and admit the isolated fingerprint package in Decision 26. Streams follow the Claude ownership split, with task 15.2's explicit task-3.2 dependency fixing shared module-file concurrency. Task 15.1 may write isolated adapter tests early; unavailable native Windows evidence blocks 15.2 completion and task 16, never PRs 1–3.

Every task detail/reference/requirement/dependency mutation was made through rune batch. The installed rune has no command to rename or move existing phase markers, so only H2 phase metadata was migrated separately to four PR boundaries; no task line was rewritten for that migration. The final file is validated through rune JSON.

Design risks map to 13.1 (10,000-file traversal), 15.1/15.2 (WAL and native platform metadata), 16.1/16.2 (cold Kiro IDE cost/fallback), and 9.1/9.2 (duplicate web-search id keys). Native Windows evidence remains absent. This mapping records planning provenance, not implementation verification or human approval.

## Reviewed Revision

The exact merged inputs, including task 1's authorised subsequent completion, are retained as `*revised-r1.md`. The current revision preserves every task ID/status/owner. Peer major 1 adds parse-option forwarding/forced-format and ID/latest/web parity; the checked warning note adds execution-action positions; Decision 28 uses a shared-json-only option to retain adjacent model changes while protecting default provider-less grouping. No source implementation or human approval is part of this revision.
