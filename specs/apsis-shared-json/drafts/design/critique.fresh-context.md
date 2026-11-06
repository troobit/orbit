# Fresh-context design critique

Reviewed `draft.codex-gpt-6.md` against approved requirements, decisions and current code. This is a same-harness/model fresh-context review, not independent-model evidence. No private transcripts were inspected.

## Critical

None identified. The first shared JSON/provider slice is viable after major findings 1–3 are resolved in the admitted design/tasks. Findings 4–7 block their later slices, not that first slice.

## Major

1. **Define metadata extraction and precedence before implementing the projection.** Requirements 1.1, 1.6, 3.1, 4.3. “Source-recorded session metadata” does not specify each source's mapping. `ParseResultMetadata` currently contains only cost; `processSessionMeta` discards Codex cwd/time, Copilot conversion ignores session-start metadata, and Kiro conversion omits its conversation ID. `Message`/`Entry.UnmarshalJSON` do not extract Claude's nested model. `SessionInfo.CreatedAt` also conflates recorded time with mtime. Fix: specify one per-source mapping and precedence table, distinguish recorded creation from sorting fallback, and make direct stdin/file conversion work without the catalogue. Choose whether to preserve previously discarded model fields now; do not accidentally promise them through an entry-only projection. Test metadata-only sessions, absent metadata, conflicting filename/content IDs and invalid timestamp strings against the schema.

2. **Make the first slice's public shapes and dispatch concrete.** Requirements 1.1–1.5, 5.1. The examples have no entries; content discriminator names, result representation, warning codes and optional-state encoding remain unspecified. Existing `ContentItem.Content` is a display string, not structured results, and custom unmarshalling requires deliberate changes when fields are added. `convert` currently returns without writing anything when entries are empty. Fix: define the exact v1 entry/content/warning structs and schema together, identify which parsed legacy fields map where, and dispatch shared JSON before the empty-rendering return. Validate representative nonempty documents plus empty/warnings-only output; keep legacy raw JSON on its existing path. Catalogue construction must not become a prerequisite for ordinary file/stdin shared conversion.

3. **Resolve the grouping compatibility contradiction.** Requirements 3.1, 5.1. The draft both prohibits grouping across model boundaries and promises unchanged legacy output. `preprocessEntries` currently crosses such boundaries, while Read/Edit headers omit model and `BuildToolMeta` drops metadata. Applying the proposed rule to existing Copilot/Kiro entries changes output without any new option. Fix: explicitly scope new turn/state grouping rules to exec entries, preserve existing no-provider output, and specify provider-bearing Codex group/deferred-result headers. Cover provider-only entries and tool-only groups in Markdown, standalone HTML and fragments. Finalise pending Codex entries before changing model/provider state so later content is not labelled with an earlier state.

4. **Kiro locator retention needs lower-level API work.** Requirements 4.2, 5.8. `logs.DiscoverAll` already deduplicates conversation IDs by newest update, losing conflicting keys before catalogue selection. `GetSession` normalises its directory argument, so passing the “exact key” does not guarantee exact-row lookup. Kiro IDE selection also prefers the richest chat, not an arbitrary canonical-path representative. Fix: specify a catalogue-facing discovery/exact-row read API or explicitly retain current selection policies with deterministic ties. Keep selected workspace/chat/execution-detail dependencies together and test duplicate IDs across keys/workspaces plus noncanonical stored keys.

5. **Cache invalidation is incomplete for non-content changes.** Requirements 2.6, 5.5, 5.9. File identity/size/mtime does not detect permission changes or every same-size rewrite; caching only the nearest existing git marker misses a newly created nearer repository. Companion files may initially be absent. Fix: name dependency fingerprints for existing and missing companions, ancestor marker discovery, symlink targets and permissions; define platform limits for metadata-preserving rewrites. Verify fingerprints before/after extraction and retry changed reads. Test WAL-only commits, companion creation/removal, root permission changes and nested repository creation while preserving zero unchanged-body reads.

6. **Immutable snapshot publication needs an ownership contract.** Requirements 4.4, 5.9. Current web listing calls `slices.Reverse` on returned sessions. Reusing that pattern against a cached snapshot mutates shared ordering. Fix: return defensive copies or render reverse iteration; define whether a refresh caller waits while other requests use the previous snapshot, and separate refresh locking from snapshot access. Test concurrent list/open and web newest-first rendering without changing catalogue order.

7. **New HTML dispatch must update visibility checks too.** Requirements 2.2–2.4, 5.7. `formatAssistantMessageHTML` first computes `hasContent` using only existing variants and can return before its rendering switch. Adding cases only to common content dispatch hides error/plan/event/usage-only entries. Fix both passes, preserve incomplete tool activity without completed results, and assert visible content/state independently for every new variant in both HTML entry paths.

## Minor

8. **Raw exec BOM handling needs an explicit test.** Requirements 2.1, 5.3. Detection strips the BOM, but `convertToJSON` scans the original bytes and skips that first event as invalid. Strip only the initial BOM before raw-event scanning and verify `thread.started` survives.

## Questions to answer in the revision

Which exact metadata precedence and v1 shapes will slice one implement? How will provider changes preserve legacy grouping? Which Kiro selection/exact-key API and cache dependency rules will govern the later catalogue slice?
