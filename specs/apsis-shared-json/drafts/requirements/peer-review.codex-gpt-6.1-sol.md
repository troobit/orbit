# Independent peer review: shared JSON requirements, revision 2

Reviewer: `codex-gpt-6.1-sol`. The current critique and earlier reviews were not read. Evidence: the requirements, decision log, handoffs, repository code and public documentation; no local transcript or server-log content was inspected. Recommendation: revise before approval. No critical findings.

## Major

1. **Duplicate streams lack a complete identity contract (2.5–2.7, 4.2).** Listing every file, giving each a unique stable identifier, and deriving identifiers from a possibly repeated thread ID need an explicit collision outcome. Adding a copy must not rename an existing session. A rollout can also record a different project from the repository containing a copied stream, making filtering and de-duplication order observable. **Fix:** require stable, bounded, CLI/URL-safe identifiers, independent of traversal order and overlapping roots; define project precedence when selecting the rollout and behavior for missing/invalid thread IDs. Test adding a duplicate, repeated listings, overlapping roots and different-project rollout/stream pairs. Preserve the human's rollout preference.

2. **The shared output can lose relationships (1.1, 2.2–2.3).** Inputs and outputs alone cannot associate interleaved tool calls with results. Existing `ContentItem.ID` and `ToolUseID` already carry that relationship. Suppressing `turn.started` also leaves usage's association with its turn unspecified, especially around failed turns and incomplete items. **Fix:** require preserved tool-call/result links, identifiable turn membership, one final state per item and deterministic ordering for incomplete items. Keep usage fields as recorded; do not infer totals before verifying whether counters are cumulative. Exercise two turns, interleaved tools, updates, failure and EOF in JSON, Markdown and HTML.

3. **Generic events can preserve only a label, and partial JSON can look complete (1.1, 2.4, 5.2).** Rendering an unknown type's name satisfies 2.4 while discarding its entire payload. Malformed-line diagnostics exist only on stderr; a saved shared JSON document cannot explain omitted content. **Fix:** preserve unknown event/item payloads in shared JSON, expose their content through the human renderers, and include structured parse warnings with physical line numbers in the session document. Keep stderr reporting. Test payload retention and a malformed middle line followed by a valid event.

4. **Fixture provenance overstates the cited documentation (2.2, 5.6).** The [official non-interactive guide](https://learn.chatgpt.com/docs/non-interactive-mode) identifies item families and supplies a short example, but does not establish complete MCP, web-search, plan or failure payload shapes. It does not prove every required type name against version 0.166.0. **Fix:** require version-pinned, authoritative type definitions or documented captures for those shapes; record each synthetic fixture's source and distinguish verified support from assumptions. Types whose shapes remain unverified should retain generic-event behavior until verified, rather than acquire invented structures.

5. **The configured-directory boundary covers opening but not discovery (2.5, 5.5).** Listing reads candidate files before opening an identifier. The current wording permits listing to inspect a symlink target outside the configured roots, although resolving that result is prohibited. Existing source listers already reject escaping file symlinks. **Fix:** apply the resolved-root boundary before discovery reads too; define treatment of directory symlinks and cycles, and de-duplicate overlapping roots. Explicit file/stdin conversion remains available independently. Verify that an escaping candidate is neither read nor listed.

## Minor

6. **Missing timestamps need an observable policy (1.1, 4.3).** Exec streams have no recorded timestamps, while existing listers fall back to modification time and `latest` sorts by `CreatedAt`. Copying or appending a stream can therefore change apparent session creation. **Fix:** distinguish unknown source creation time from any filesystem fallback used for listing/sorting; specify stdin behavior and deterministic ordering when times are absent or equal. Do not synthesize message timestamps from file metadata.

7. **Compatibility is decided but not fully verifiable (1.3–1.4, 4.3; Q13).** The criteria only require a matching integer and fixture validity, not Q13's breaking-change rule or additive compatibility. Listing documents are not explicitly covered by schema validation. **Fix:** carry the compatibility rule into acceptance criteria, cover both session and listing documents with the published schema, and verify omitted optional metadata, empty listings and warnings. Leave concrete field names and schema location to design.

8. **Verification omits the listing-to-opening path (4.1–4.2, 5.6).** Parser fixtures cannot catch all-project sessions that list successfully but cannot resolve. Today Kiro CLI lookup requires a directory key, and Kiro IDE resolution hashes a supplied project path; neither becomes global merely by passing an empty project. **Fix:** require CLI and web list/open round trips from an unrelated working directory for every source, including Kiro execution details, plus same-identifier sessions from different sources.

## Questions to resolve before design

- What distinguishes copied streams without changing existing identifiers, and which project owns a de-duplicated rollout/stream pair?
- Will session JSON retain unknown payloads and parse warnings so consumers can recognize incomplete translations?
- Which version-pinned source establishes each synthetic exec fixture's complete shape?
