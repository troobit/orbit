---
references:
    - specs/apsis-shared-json/requirements.md
    - specs/apsis-shared-json/design.md
    - specs/apsis-shared-json/decision_log.md
---
# Apsis shared JSON, listing and Codex exec streams

## Validation baseline

- [-] 1. Assert follower shutdown errors so baseline lint is clean <!-- id:9kwhro4 -->
  - internal/transcript/follow_test.go: three unchecked channel errors after cancellation. Assert the expected return without changing production behavior; run make lint and project tests.
  - Stream: 1
  - Owner: codex-gpt-6
  - Requirements: [5.1](requirements.md#5.1)

## Shared JSON and provider

- [ ] 2. Write failing shared session JSON contract and CLI tests <!-- id:9kwhro5 -->
  - Synthetic examples for all five existing sources; optional metadata, first-known/model changes, tool links/errors, cost/unit, long content, empty sessions and partial warnings. Validate against the published schema using a real test-only validator. Check -f shared-json help and follow rejection, preserve raw output.
  - Blocked-by: 9kwhro4 (Assert follower shutdown errors so baseline lint is clean)
  - Stream: 1
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.3](requirements.md#1.3), [1.4](requirements.md#1.4), [1.5](requirements.md#1.5), [1.6](requirements.md#1.6), [5.1](requirements.md#5.1), [5.3](requirements.md#5.3)

- [ ] 3. Implement shared session JSON and schema to pass contract tests <!-- id:9kwhro6 -->
  - internal/transcript/document.go/json.go; docs/schemas/apsis-v1.schema.json; parsed source metadata and cmd/apsis conversion/help. Encode empty sessions rather than taking the human-output early return. Preserve common content and warnings; no cross-project listing or exec parser in this task.
  - Blocked-by: 9kwhro5 (Write failing shared session JSON contract and CLI tests)
  - Stream: 1
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.3](requirements.md#1.3), [1.4](requirements.md#1.4), [1.5](requirements.md#1.5), [1.6](requirements.md#1.6), [5.1](requirements.md#5.1), [5.3](requirements.md#5.3)

- [ ] 4. Write failing Codex provider propagation and renderer tests <!-- id:9kwhro7 -->
  - Synthetic model_provider cases across shared JSON, Markdown, standalone HTML and web fragments including grouped/deferred tool paths, omission and first-known summaries.
  - Blocked-by: 9kwhro6 (Implement shared session JSON and schema to pass contract tests)
  - Stream: 1
  - Requirements: [3.1](requirements.md#3.1), [5.1](requirements.md#5.1)

- [ ] 5. Preserve and render recorded Codex providers to pass tests <!-- id:9kwhro8 -->
  - Codex metadata/parser state, optional common Entry fields, renderer metadata and group/tool-result propagation. Preserve legacy raw JSON and absent-provider output. Validate the integrated slice with make test, make lint and make build-apsis.
  - Blocked-by: 9kwhro7 (Write failing Codex provider propagation and renderer tests)
  - Stream: 1
  - Requirements: [3.1](requirements.md#3.1), [5.1](requirements.md#5.1)

## Cross-project catalogue

- [ ] 6. Write failing global catalogue, list-open and refresh tests <!-- id:9kwhro9 -->
  - Every existing source must round-trip CLI/web from unrelated directories; preserve Kiro directory/workspace/execution-detail locators. Check source collisions, shared listing schema, missing time/fallback and latest ties, changed/removed companions, zero unchanged body reads and concurrent snapshots. Add generated cold/traversal benchmarks before dependent implementation.
  - Blocked-by: 9kwhro8 (Preserve and render recorded Codex providers to pass tests)
  - Stream: 1
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [4.4](requirements.md#4.4), [5.8](requirements.md#5.8), [5.9](requirements.md#5.9)

- [ ] 7. Implement global catalogue, source selection and JSON listings <!-- id:9kwhroa -->
  - sessions.Catalog/private locators; CLI --all-projects/--source; serve catalogue reuse; preserve bare IDs and first-known recorded metadata. Watch companion files and SQLite WAL/journal; warn on failed reads rather than stale success. Pass preceding tests/benchmarks and project checks.
  - Blocked-by: 9kwhro9 (Write failing global catalogue, list-open and refresh tests)
  - Stream: 1
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [4.4](requirements.md#4.4), [5.8](requirements.md#5.8), [5.9](requirements.md#5.9)

## Codex exec streams

- [ ] 8. Write failing exec lifecycle, content and renderer tests <!-- id:9kwhrob -->
  - Pinned synthetic shapes: BOM/malformed prefix, forced pre-thread failure, multi-turn interleaved tools, reused item IDs, updates/failure/EOF, unknown payloads and usage fields. Rapid lifecycle/order/payload properties; JSON/Markdown/both HTML renderers; no prompt files.
  - Blocked-by: 9kwhroa (Implement global catalogue, source selection and JSON listings)
  - Stream: 1
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.9](requirements.md#2.9), [2.10](requirements.md#2.10), [2.11](requirements.md#2.11), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.6](requirements.md#5.6), [5.7](requirements.md#5.7)

- [ ] 9. Implement exec translation and lifecycle fidelity <!-- id:9kwhroc -->
  - FormatCodexExec, parser dispatch, common content and renderers. Preserve source timestamps as absent, scoped tool links, raw JSON events, every usage snapshot field as reported total without summing; reject follow mode after indirect source resolution.
  - Blocked-by: 9kwhrob (Write failing exec lifecycle, content and renderer tests)
  - Stream: 1
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.9](requirements.md#2.9), [2.10](requirements.md#2.10), [2.11](requirements.md#2.11), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.6](requirements.md#5.6), [5.7](requirements.md#5.7)

## Codex exec discovery

- [ ] 10. Write failing containment, identity and rollout-precedence tests <!-- id:9kwhrod -->
  - Empty/configured/nested/overlapping roots and CLI/environment precedence; approved skips and explicitly skipped-name roots; escapes/cycles, nearest worktree root, copies/appends/resume IDs. Verify global resolvable-rollout suppression before project filters, direct conversion outside roots, source opening, latest/follow and unchanged refresh.
  - Blocked-by: 9kwhroc (Implement exec translation and lifecycle fidelity)
  - Stream: 1
  - Requirements: [2.5](requirements.md#2.5), [2.6](requirements.md#2.6), [2.7](requirements.md#2.7), [2.8](requirements.md#2.8), [2.10](requirements.md#2.10), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4), [5.5](requirements.md#5.5), [5.8](requirements.md#5.8), [5.9](requirements.md#5.9)

- [ ] 11. Integrate safe stream discovery and stable opening <!-- id:9kwhroe -->
  - Canonical roots/files, bounded artifact IDs and global rollout index; --codex-exec-dir / APSIS_CODEX_EXEC_DIRS in CLI/serve; source badges/filters and warning surfaces. Pass all synthetic round trips; run make test, make lint and make build.
  - Blocked-by: 9kwhrod (Write failing containment, identity and rollout-precedence tests)
  - Stream: 1
  - Requirements: [2.5](requirements.md#2.5), [2.6](requirements.md#2.6), [2.7](requirements.md#2.7), [2.8](requirements.md#2.8), [2.10](requirements.md#2.10), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4), [5.5](requirements.md#5.5), [5.8](requirements.md#5.8), [5.9](requirements.md#5.9)
