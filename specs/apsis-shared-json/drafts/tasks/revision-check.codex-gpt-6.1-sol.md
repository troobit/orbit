# Shared tasks revision check

Fresh-context NON-AUTHOR SAME-MODEL verification of the completed 22:36:41Z revision; neither independent-model evidence nor human approval. Reviewed task SHA-256: `a2f6f8da5f9190611354d79070b2a7e14af0f3ce46ff94c42505a1a9b607dbee`. Read both completed task reviews, canonical documents, reviewed snapshots and relevant source. This verifies the revised plan, not an implementation.

**Critical:** none found. **Major:** none remaining within the requested revision scope.

## Dispositions

1. **Peer major 1 — resolved.** [Design section 6](../../design.md#6-resolution-in-all-projects-scope), Q74 and tasks 14.1/14.2 now require `ParseResolved(rs, opts ...transcript.ParseOptions)` to copy caller options and overlay the selected Kiro IDE execution-detail cost path. Both Codex flags survive. Explicit CLI `-a` dispatch stays before the helper and receives the same options; default web parsing receives neither flag. This preserves the current [CLI precedence](../../../../cmd/apsis/main.go) and source-bound [web cost association](../../../../internal/apsisweb/handlers.go). Task 14.1 explicitly regresses header-only Codex through source-selected ID and `latest`: shared JSON returns metadata with empty entries, while Markdown/HTML/web retains the prior error. No source-based inference of output format is needed.

2. **Warning-position correction — resolved.** Q75 supersedes Q39 without changing an existing published contract. The [Kiro IDE parser](../../../../internal/transcript/kiro_ide_parser.go) uses chat indexes for missing/unknown roles and execution-action indexes for unknown actions. Tasks 2.1/2.2 assign the unit at each creation site, retaining numeric `Line` and existing diagnostic text. Tasks 3.1/3.2 project physical/chat/action units to `line`/`chat_message`/`execution_action` and validate both Kiro paths against the published, exact-initial and structurally closed schemas, including mistyped-position rejection. This corrects provenance under 1.1 without changing stderr output under 5.1.

3. **Metadata-change compatibility — resolved.** [Decision 28](../../decision_log.md#decision-28-preserve-codex-metadata-changes-with-an-explicit-compatibility-option) matches the current [Codex consolidation behavior](../../../../internal/transcript/codex_parser.go): shared-JSON-only `PreserveModelChanges` permits separate model-bearing runs, while default provider-less model-only consolidation remains unchanged. Recorded provider transitions create the appropriate new entry under 3.1; identical metadata never creates a boundary or retroactively relabels earlier content. Tasks 2.1/2.2 require both modes, first-known summaries, timestamps and interleaved tool links/content across transitions. This reconciles approved 1.6/3.1/5.1 through one common parser and entry representation, without a second format-specific parser or new human policy.

4. **Future helper regressions — resolved.** Tasks 4.1/4.2 pair the initial shared-JSON flag selection with file/stdin/ID acceptance and model-run tests. Tasks 14.1/14.2 repeat the checks after helper replacement, covering explicit-source ID, `latest`, web defaults, identical headers, both flags, forced-format precedence and the selected Kiro cost association. The later refactor therefore has concrete failing-test obligations for the original defect and the new compatibility option.

5. **Codex creation-time provenance — resolved.** The design source table, Q77 and tasks 2.1/2.2 use outer `CodexEntry.Timestamp` on `session_meta`, matching the current listing extractor. Tests explicitly distinguish outer/payload timestamps and omit parsed creation time when the outer value is absent. No payload fallback is introduced.

## Preservation and remaining boundaries

`rune list --format json` parses 46 items, sixteen parents and four phases. Numbered/stable IDs, statuses, owners, dependencies and streams match the exact reviewed input; task 1 remains Completed with owner `codex-gpt-6`. Human Decisions 1–11 and Q25–Q28 are byte-identical to that input. U1–U3 remain unanswered; no approval is inferred.

No additional blocking question or concrete fix remains before coordinator consideration of task 2. Native Windows evidence and any conditional WAL fallback audit still gate the cache work as documented. No implementation/build/native tests, private logs or external actions were used for this check.
