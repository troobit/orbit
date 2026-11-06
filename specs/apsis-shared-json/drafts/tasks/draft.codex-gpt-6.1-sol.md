---
references:
    - ../../requirements.md
    - ../../design.md
    - ../../decision_log.md
    - ../design/questions.md
    - decisions.codex-gpt-6.1-sol.md
metadata:
    admission: local-planning-only
    author: codex-gpt-6.1-sol
    executable: "false"
    freeze: source-code-and-canonical-tasks
    mode: planning-only
    status: draft
---
# Apsis Shared JSON — Planning Draft

- [ ] 1. Repair the two remaining follow-test receive checks without changing production behavior <!-- id:ep7jb4l -->
  - Planning-only baseline repair: internal/transcript/follow_test.go:1056 and :1120 have the two known errcheck receives; read current code and callers before editing, check received errors, and preserve the test intent.
  - The canonical file has 11 tasks and canonical task 1 remains in progress under codex-gpt-6. This separate pending draft neither claims nor releases that work; coordinate with that owner only after the freeze is lifted.
  - Test-only repair is exempt from a production red/green pair. Verify with make lint and make test; distinguish the previously observed apsisweb shutdown flake from a new regression and do not mark success from an unperformed rerun.
  - Stream: 1
  - Requirements: [5.1](../../requirements.md#5.1)
  - References: ../../design.md, ../../decision_log.md, ../../../../internal/transcript/follow_test.go

- [ ] 2. Red: test recorded provider, first-known parser summaries and unchanged existing rendering <!-- id:ep7jb4m -->
  - Pair with task 3. Add behavior assertions in internal/transcript parser and metadata/render tests for Codex session_meta model_provider, model/provider-only headers, later model/provider changes and absent metadata.
  - Cover header-only and user-only Codex sessions, first known Copilot model_change before assistant content, Kiro CLI request metadata and Kiro IDE metadata.modelId; summaries must not require an assistant entry.
  - Assert parsed creation/project metadata and ParseResult.Format for existing sources, first parseable Claude timestamp/first cwd, Codex metadata, existing cost units and Kiro IDE chat-message warning positions.
  - Preserve Claude model omission and Kiro IDE provider omission under the unanswered U1-A/U2-A recommendations. Existing Read/Edit groups, user/slash-command/deferred-result headers and provider-less goldens remain unchanged.
  - Tests must fail for missing metadata/provider behavior before task 3; use synthetic fixtures and retain the existing source-specific timestamp/model semantics.
  - Blocked-by: ep7jb4l (Repair the two remaining follow-test receive checks without changing production behavior)
  - Stream: 1
  - Requirements: [1.1](../../requirements.md#1.1), [1.6](../../requirements.md#1.6), [3.1](../../requirements.md#3.1), [5.1](../../requirements.md#5.1)
  - References: ../../design.md, ../../decision_log.md, ../../../../internal/transcript/codex_parser_test.go, ../../../../internal/transcript/copilot_parser_test.go, ../../../../internal/transcript/kiro_parser_test.go, ../../../../internal/transcript/kiro_ide_parser_test.go, ../../../../internal/transcript/metadata_test.go

- [ ] 3. Green: preserve parser summaries and render recorded Codex providers through existing assistant headers <!-- id:ep7jb4n -->
  - Pass task 2 in internal/transcript/types.go, parser.go, codex_types.go, codex_parser.go and the existing Copilot/Kiro parsers. Populate StartedAt, Project, ThreadID, FirstModel and FirstProvider from recognized source facts; retain active values on entries.
  - Add modelLabel in metadata.go and use it only where assistant Markdown/HTML currently shows model: model (provider), or (provider) alone. Keep grouping, deferred results and other sources unchanged.
  - Add the design's json-ignored Entry/ContentItem/Usage/PlanItem representation fields and warning Code/Unit types. Add SessionInfo/SessionMetadata/ListWarning data fields plus shared Options/Option types in internal/sessions/types.go for the next projection tests; type-only additions need no separate behavior pair.
  - Keep optional TotalCost nil-safe. Do not infer models/providers, extract Claude message.model or Kiro IDE metadata.modelProvider, or add unrelated token extraction.
  - Run make test and make lint. These shared type additions are consumed immediately by task 5; exec parsing and runtime option behavior remain in their later paired tasks.
  - Blocked-by: ep7jb4m (Red: test recorded provider, first-known parser summaries and unchanged existing rendering)
  - Stream: 1
  - Requirements: [1.1](../../requirements.md#1.1), [1.6](../../requirements.md#1.6), [3.1](../../requirements.md#3.1), [5.1](../../requirements.md#5.1)
  - References: ../../design.md, ../../decision_log.md, ../../../../internal/transcript/types.go, ../../../../internal/transcript/parser.go, ../../../../internal/transcript/codex_parser.go, ../../../../internal/transcript/metadata.go, ../../../../internal/transcript/markdown.go, ../../../../internal/transcript/html.go, ../../../../internal/sessions/types.go

- [ ] 4. Red: test shared session/listing documents and open versus structurally closed schema validation <!-- id:ep7jb4o -->
  - Pair with task 5. Add internal/sharedjson tests projecting existing-source fixture ParseResults and synthetic new-content ParseResults/SessionInfo values using task 3 types; do not require the later exec parser or runtime listing cache for these tests.
  - Assert common v1 session/listing envelopes, integer const version, shared metadata definition, all block fields/links, false is_error, recorded zero exit code versus absent exit code, cost amount/unit, physical line versus chat_message warnings, omitted optional fields, empty arrays as [] and trailing-newline output.
  - Test parsed-first metadata with resolver filling only missing facts; FirstModel/FirstProvider plus entry fallback retain header-only/user-only facts and later entry changes. Never promote filesystem fallback to created_at; stdin has no file time.
  - Check tool text over 3000 runes byte-for-byte, large integer precision and arbitrary nested inputs/payloads/unknown usage counters. Test every existing supported source with golden JSON and generated exec representation; parser-produced exec conformance is added in task 10.
  - Load docs/schemas/apsis-shared-json.v1.schema.json locally with network loading disabled. Validate fixtures against the published open Draft 2020-12 schema and a test copy closing only apsis-owned structural objects; input, payload and counters stay open.
  - Prove additive future structural fields pass the published schema, undeclared producer structural fields fail the closed copy, valid arbitrary payload keys pass both and wrong known-field types fail.
  - Add cmd/apsis tests for -f shared-json on file/stdin/ID, empty-session dispatch before the early return, unchanged raw -f json for existing sources, and the named -F/shared-json rejection.
  - U3-A (test-only github.com/santhosh-tekuri/jsonschema/v6) is an unanswered recommendation, not human approval or current dependency authorization under the freeze; plan the import only in _test.go and change this pair if U3 is answered differently.
  - Blocked-by: ep7jb4n (Green: preserve parser summaries and render recorded Codex providers through existing assistant headers)
  - Stream: 1
  - Requirements: [1.1](../../requirements.md#1.1), [1.2](../../requirements.md#1.2), [1.3](../../requirements.md#1.3), [1.4](../../requirements.md#1.4), [1.5](../../requirements.md#1.5), [1.6](../../requirements.md#1.6), [4.3](../../requirements.md#4.3), [5.1](../../requirements.md#5.1), [5.3](../../requirements.md#5.3)
  - References: ../../design.md, ../../decision_log.md, ../../../../internal/transcript/testdata/valid.jsonl, ../../../../internal/transcript/testdata/codex_valid.jsonl, ../../../../cmd/apsis/main_test.go

- [ ] 5. Green: publish shared JSON v1 projection/schema and wire CLI conversion without changing raw output <!-- id:ep7jb4p -->
  - Pass task 4 by creating internal/sharedjson document types, Session, Listing, Write and SchemaVersion, and docs/schemas/apsis-shared-json.v1.schema.json at the design's exact paths.
  - Keep the projection outside transcript to avoid the sessions-to-transcript import cycle; project parsed values without renderer helpers or truncation, preserve arbitrary JSON numbers, reuse session-metadata definitions for listing rows and write two-space indented JSON with SetEscapeHTML(false).
  - Implement parsed-first session metadata and resolver-fill precedence, first-known summaries, all design block/warning/cost/time fields, open additive schema objects and the structurally closed test copy. Follow U3's eventual resolution; no validator import may enter production binaries.
  - Update cmd/apsis Config/help and conversion input context to carry source, ID, canonical path, cost path and resolver metadata; dispatch shared JSON before zero-entry handling, expose session/listing documents, and reject -F shared-json by name.
  - Keep the raw JSON path and existing Markdown/HTML behavior unchanged apart from task 3's recorded Codex providers. No JSON HTTP endpoint, live shared JSON or new source discovery is introduced here.
  - Run make build-apsis, make test and make lint; schema fixture validation is part of make test.
  - Blocked-by: ep7jb4o (Red: test shared session/listing documents and open versus structurally closed schema validation)
  - Stream: 1
  - Requirements: [1.1](../../requirements.md#1.1), [1.2](../../requirements.md#1.2), [1.3](../../requirements.md#1.3), [1.4](../../requirements.md#1.4), [1.5](../../requirements.md#1.5), [1.6](../../requirements.md#1.6), [4.3](../../requirements.md#4.3), [5.1](../../requirements.md#5.1), [5.3](../../requirements.md#5.3)
  - References: ../../design.md, ../../decision_log.md, ../../../../cmd/apsis/main.go, ../../../../internal/transcript/parser.go, ../../../../internal/sessions/types.go

- [ ] 6. Red: test metadata-only platform fingerprints and native Windows replacement capability <!-- id:ep7jb4q -->
  - Pair with task 7, before every cache consumer. Add injectable-adapter unit tests and build-tagged native integration tests in internal/sessions for Unix and Windows fingerprint fields and errors.
  - Assert equal-size replacement and in-place rewrite with preserved mtime change the key through identity or change time; include companion absence/presence and unsupported/incomplete metadata outcomes. A failed fingerprint must never become a size/mtime-only cache hit.
  - Unix expectations include device, inode, ctime, mtime and size; Windows expectations include volume serial number, file ID, change time, last-write time and size obtained through metadata-only handles.
  - Tests must count transcript-body bytes separately from metadata operations, use generated temporary files, and establish real native Windows evidence in addition to injected values. Cross-compilation or mocked Windows results alone cannot satisfy the capability gate.
  - Native Windows execution is unavailable in the current planning context and is not claimed verified. Failure/unavailability must leave task 7 incomplete and block task 8/cache admission pending a platform/requirements decision.
  - Blocked-by: ep7jb4p (Green: publish shared JSON v1 projection/schema and wire CLI conversion without changing raw output)
  - Stream: 1
  - Requirements: [5.1](../../requirements.md#5.1), [5.9](../../requirements.md#5.9)
  - References: ../../design.md, ../../decision_log.md, ../../../../internal/sessions/lister_test.go, ../../../../docs/agent-notes/go-rules.md

- [ ] 7. Green: implement platform identity/change adapters and pass the native capability gate <!-- id:ep7jb4r -->
  - Pass task 6 with internal/sessions/fingerprint_unix.go and fingerprint_windows.go behind one injectable interface; use existing golang.org/x/sys/windows metadata-handle APIs on Windows and the required Unix identity/change fields.
  - Provide explicit warning/error results when fields or storage support are unavailable; never read transcript bodies for fingerprints or downgrade to size/mtime. Add only platform type/wiring support needed by the next cache pair.
  - Completion requires the native Windows equal-size preserved-mtime replacement/rewrite tests plus supported Unix tests; record actual platform/filesystem evidence when available. Until then keep this task pending/incomplete and do not admit dependent cache implementation.
  - If the capability fails, record the storage/platform limitation for a requirements decision rather than silently weakening 5.9. Native Windows success has not occurred during this draft.
  - Run make test and make lint on supported hosts; exercise native Windows tests there and use make build-apsis where supported.
  - Blocked-by: ep7jb4q (Red: test metadata-only platform fingerprints and native Windows replacement capability)
  - Stream: 1
  - Requirements: [5.1](../../requirements.md#5.1), [5.9](../../requirements.md#5.9)
  - References: ../../design.md, ../../decision_log.md, ../../../../internal/sessions/types.go, ../../../../docs/agent-notes/go-rules.md

- [ ] 8. Red: test all-projects list/open parity, legacy ordering and cached refreshes across existing sources <!-- id:ep7jb4s -->
  - Pair with task 9, and do not start cache-dependent work before task 7's native gate passes. Add internal/sessions, Kiro logs, cmd/apsis and apsisweb tests using synthetic HOME/XDG roots and unrelated working directories.
  - Round-trip CLI rows and httptest web source/ID routes for Claude, Codex, Copilot, Kiro CLI and Kiro IDE, including identical IDs across sources, Kiro IDE execution-detail actions/cost and shared ParseResolved behavior.
  - Test Kiro CLI one conversation under two database keys: GetSessionByID and DiscoverAll both choose updated_at DESC then key ASC without re-normalizing stored keys. Share Kiro IDE best candidate by chat count, newest mtime then lexical path and derive CostPath from its winning workspace.
  - Test all-projects (source,id) duplicate warnings and resolver-selected winners: sorted Claude directories; first filename-match Codex walk winner; invalid first Codex head followed by valid duplicate is warned/omitted without promoting the second. This selected rollout metadata is reused by later suppression tests. Add a no-flag cross-project duplicate fixture: a later rollout remains its own project's per-file cwd-filtered row even when another project holds the global winner (Q62); all-projects rows still choose the global winner.
  - Add rapid extended-order properties: unknown before known time, source priority then ID ties, latest equals the final ascending row. No-flag equal-time fixtures retain legacy order/latest; Claude recorded timestamp after 50 lines/1 MiB and long Codex session_meta retain existing scan limits/fallback/error behavior.
  - Count body bytes and DB discovery calls: unchanged refresh reads zero after warmup; edits/replacements, additions/removals, companion-only changes (including absent workspace.yaml, -wal and -journal), and changed extraction failures invalidate without serving stale rows. Restat changes prevent caching unstable reads.
  - Test two writes on one WAL-mode connection between refreshes before relying on the DB key; a failed WAL assumption blocks cache completion for a requirements decision, with measured page reads rather than a claimed zero-read pass.
  - Assert whole-pass mutex behavior, fresh result slices and web slices.Reverse isolation. Add generated cold multi-source listing benchmarks including whole-file Kiro IDE metadata; use results to choose the design's narrower decoder only if needed.
  - Test CLI --all-projects/project conflict, explicit -s/--source IDs versus -a forced format, source misuse/unknown source, legacy bare IDs, shared-json scope metadata and flagged fallback times.
  - Test serve AllProjects/long-lived Lister, all-projects project column and polled sessions_list warning banner appearing once and updating with refresh. Keep legacy no-flags output, metadata scan semantics and source pills unchanged.
  - Blocked-by: ep7jb4r (Green: implement platform identity/change adapters and pass the native capability gate)
  - Stream: 1
  - Requirements: [1.1](../../requirements.md#1.1), [1.2](../../requirements.md#1.2), [1.4](../../requirements.md#1.4), [1.6](../../requirements.md#1.6), [4.1](../../requirements.md#4.1), [4.2](../../requirements.md#4.2), [4.3](../../requirements.md#4.3), [4.4](../../requirements.md#4.4), [5.1](../../requirements.md#5.1), [5.7](../../requirements.md#5.7), [5.8](../../requirements.md#5.8), [5.9](../../requirements.md#5.9)
  - References: ../../design.md, ../../decision_log.md, ../../../../internal/sessions/lister_test.go, ../../../../internal/sessions/resolver_test.go, ../../../../internal/agents/kiro/logs/session_test.go, ../../../../internal/agents/kiro/logs/discover_test.go, ../../../../cmd/apsis/main_test.go, ../../../../internal/apsisweb/handlers_test.go

- [ ] 9. Green: integrate all-projects resolution and the verified long-lived listing cache <!-- id:ep7jb4t -->
  - Pass task 8 in sessions/lister.go, resolver.go, types.go and new cache.go/parse.go; keep Lister/Resolver entry points and shared Options rather than introducing a catalogue or locators. Task 7 is a hard admission dependency.
  - Implement source extraction and companion fingerprint tables from design section 4. Preserve existing Claude timestamp and Codex scan semantics, bound only optional new Claude cwd extraction to 50 lines/1 MiB, and retain recorded-versus-filesystem flags.
  - Implement mutex-held enumeration/stat/extract/restat/sweep passes with explicit warnings, no stale metadata after changed extraction failures and fresh allocated return slices; reuse unchanged Kiro DB discovery only after the WAL regression passes.
  - Share selectCodexCandidate for resolution and all-projects deduplication, preserving the established first filename-match winner and validated selected head; keep legacy project rows enumerated and cwd-filtered per file (Q62). Share bestKiroIDECandidate and implement Kiro GetSessionByID/DiscoverAll deterministic tie ordering.
  - Implement ParseResolved with Kiro IDE CostPath and existing-format detection, reserving the exec branch for task 11; propagate Path, Project and time provenance to session context.
  - Wire --all-projects and -s/--source, common scope/settings, extended ordering only for shared JSON/all-projects/configured roots, JSON listing metadata and latest. Preserve legacy source probing and default tie winners.
  - Keep one Lister in apsisweb, add project presentation in all-projects scope, and render listing warnings in the polled fragment with no duplicate page banner. Do not expose a JSON HTTP API.
  - Run make build-apsis, make test and make lint, including round-trip and zero-body-read regressions; evaluate the generated cold-listing benchmark before completing the cache risk.
  - Blocked-by: ep7jb4r (Green: implement platform identity/change adapters and pass the native capability gate), ep7jb4s (Red: test all-projects list/open parity, legacy ordering and cached refreshes across existing sources)
  - Stream: 1
  - Requirements: [1.1](../../requirements.md#1.1), [1.2](../../requirements.md#1.2), [1.4](../../requirements.md#1.4), [1.6](../../requirements.md#1.6), [4.1](../../requirements.md#4.1), [4.2](../../requirements.md#4.2), [4.3](../../requirements.md#4.3), [4.4](../../requirements.md#4.4), [5.1](../../requirements.md#5.1), [5.7](../../requirements.md#5.7), [5.8](../../requirements.md#5.8), [5.9](../../requirements.md#5.9)
  - References: ../../design.md, ../../decision_log.md, ../../../../internal/sessions/lister.go, ../../../../internal/sessions/resolver.go, ../../../../internal/sessions/types.go, ../../../../internal/agents/kiro/logs/session.go, ../../../../internal/agents/kiro/logs/discover.go, ../../../../cmd/apsis/main.go, ../../../../internal/apsisweb/handlers.go, ../../../../internal/apsisweb/templates/sessions.html

- [ ] 10. Red: test complete-line exec detection, lifecycle translation and all rendered content/state <!-- id:ep7jb4u -->
  - Pair with task 11. Add synthetic testdata/codex_exec fixtures plus pinned 0.166.0 definition provenance; unobserved shapes cite their definition, and unverified shapes use generic events instead of invented specialized payloads.
  - Detection/probe tables cover BOM, blanks, malformed middle/prefix lines, over-10-MiB skipped lines, malformed prefixes beyond 64 KiB containing every false Kiro heuristic marker, split long first thread.started containing those markers, all three probe outcomes and byte-exact replay.
  - Only first parseable object thread.started with nonempty string thread_id auto-selects exec; other exec outcomes require -a codex-exec. Keep complete structured Kiro recognition, large legacy Kiro recognition and all non-exec fallback outcomes unchanged.
  - Cover each specialized item, plans, errors before thread creation, turn.failed, unknown event/item/usage payloads, large numbers, multiple/interleaved turns, item.updated, EOF/incomplete, duplicate/same/different thread.started and repeated source item IDs across resumed turns.
  - Add rapid lifecycle properties for one final record per (turn,id), first-appearance order, last payload and incomplete iff never completed. Test the pinned web_search duplicate-id shape for stable encoding/json last-key behavior across started/completed events.
  - Two missing/empty/non-string-ID tool items and known invalid typed payloads must remain distinct generic events with item_decode_failed and no tool links; two turns reusing item_0 must each show their own command/result through t<turn>:<item> links.
  - Golden shared JSON, Markdown, standalone HTML and HTML fragment for every new block/state, including entries containing only plan/error/event/usage so both HTML visibility and render passes are exercised; assert escape handling and every tool_result emission.
  - Usage retains every source counter in order with thread_total_at_turn_completion, labels each turn without summing snapshots or deriving deltas. No exec source/entry time, model/provider or invented thread; file mtime is listing-only and stdin has none.
  - Test fixed-text physical one-based malformed/oversized warnings without source-line copies on stderr/JSON/web; raw -f json with a leading BOM must keep thread.started under existing raw line-limit behavior.
  - Test all named -F rejections for shared JSON, forced exec and detected direct files; source-ID/latest exec follow checks are extended in task 12. Direct empty shared JSON emits entries: [].
  - For direct file conversion, assert canonical on-disk path-derived ID (sanitized thread prefix plus full SHA-256), nearest containing repository/worktree project, content thread rather than filename stem, and no ID/project for stdin. Test web warning/template rendering from synthetic parsed/resolved values here; actual exec route lookup is tested in task 12.
  - Blocked-by: ep7jb4t (Green: integrate all-projects resolution and the verified long-lived listing cache)
  - Stream: 1
  - Requirements: [1.1](../../requirements.md#1.1), [1.2](../../requirements.md#1.2), [1.4](../../requirements.md#1.4), [1.5](../../requirements.md#1.5), [1.6](../../requirements.md#1.6), [2.1](../../requirements.md#2.1), [2.2](../../requirements.md#2.2), [2.3](../../requirements.md#2.3), [2.4](../../requirements.md#2.4), [2.9](../../requirements.md#2.9), [2.10](../../requirements.md#2.10), [2.11](../../requirements.md#2.11), [5.1](../../requirements.md#5.1), [5.2](../../requirements.md#5.2), [5.3](../../requirements.md#5.3), [5.6](../../requirements.md#5.6), [5.7](../../requirements.md#5.7), [2.6](../../requirements.md#2.6), [2.8](../../requirements.md#2.8)
  - References: ../../design.md, ../../decision_log.md, ../../../../internal/transcript/parser_test.go, ../../../../internal/transcript/markdown_test.go, ../../../../internal/transcript/html_test.go, ../../../../cmd/apsis/main_test.go, ../../../../internal/apsisweb/handlers_test.go, ../../requirements.md

- [ ] 11. Green: parse Codex exec events and render final turn/item state through every output <!-- id:ep7jb4v -->
  - Pass task 10 with new transcript/codex_exec_types.go and codex_exec_parser.go: shared complete-line ProbeExecStream, forced ParseCodexExec, DetectFormat replay and Parse/ParseJSONLWithFormat dispatch. Preserve legacy detection outside an established exec signature.
  - Keep first-appearance records by turn/item, physical-line keys for invalid IDs, final payload replacement and EOF incompleteness; create turn-qualified tool IDs plus original ItemID and mandatory result blocks.
  - Validate specialized item IDs/payloads against pinned types, translate recognized families, and retain unknown/invalid shapes as generic events with fixed structured warnings. Preserve raw JSON precision and ordered usage totals.
  - Update Markdown and both HTML passes for plans, errors, events, usage, tool families, exit codes and incomplete markers; escape event JSON and retain existing grouping behavior. Extend sharedjson fixture validation to real exec parser results.
  - Wire -a codex-exec and direct-input metadata, implementing reusable canonical-path/derived-ID/containing-repository helpers in sessions/codexexec.go now so file conversion has no forward dependency on discovery. Emit content Thread ID and derived path ID, strip BOM only for raw exec conversion and reject exec follow; stdin invents no identity/time/project. Task 13 reuses these working helpers.
  - Extend sessions.ParseResolved for exec, add exec warning/template rendering and shared usage styling, and verify those with synthetic resolved values without requiring configured-root route lookup. Discovery/source registration/exec ID routes remain in task 13.
  - Run make build-apsis, make test and make lint; retain fixture provenance without private captures or machine-specific data.
  - Blocked-by: ep7jb4u (Red: test complete-line exec detection, lifecycle translation and all rendered content/state)
  - Stream: 1
  - Requirements: [1.1](../../requirements.md#1.1), [1.2](../../requirements.md#1.2), [1.4](../../requirements.md#1.4), [1.5](../../requirements.md#1.5), [1.6](../../requirements.md#1.6), [2.1](../../requirements.md#2.1), [2.2](../../requirements.md#2.2), [2.3](../../requirements.md#2.3), [2.4](../../requirements.md#2.4), [2.9](../../requirements.md#2.9), [2.10](../../requirements.md#2.10), [2.11](../../requirements.md#2.11), [5.1](../../requirements.md#5.1), [5.2](../../requirements.md#5.2), [5.3](../../requirements.md#5.3), [5.6](../../requirements.md#5.6), [5.7](../../requirements.md#5.7), [2.6](../../requirements.md#2.6), [2.8](../../requirements.md#2.8)
  - References: ../../design.md, ../../decision_log.md, ../../../../internal/transcript/parser.go, ../../../../internal/transcript/types.go, ../../../../internal/transcript/markdown.go, ../../../../internal/transcript/html.go, ../../../../cmd/apsis/main.go, ../../../../internal/apsisweb/handlers.go, ../../../../internal/apsisweb/static/style.css

- [ ] 12. Red: test configured exec discovery, stable identity, rollout suppression and list/open refresh integration <!-- id:ep7jb4w -->
  - Pair with task 13. Use generated temp trees and the shared probe: no configured roots discovers nothing; unrelated JSONL is silent; threadless exec is warned/omitted; recursive .jsonl qualification matches direct detection including false Kiro prefixes.
  - Cover repeatable --codex-exec-dir versus APSIS_CODEX_EXEC_DIRS on apsis and serve: OS path-list splitting, CLI replacement, relative roots, unset/empty environment, single empty flag and mixed empty/nonempty usage error.
  - Cover overlapping/permuted roots, resolved root/file symlinks, case aliases on case-insensitive storage, escaped/nonregular targets, directory-symlink cycles, unreadable roots/subdirectories/files, skipped descent names and explicit roots nested inside skipped names. Assert containment before body reads and identifier opening.
  - Add rapid identity properties: sanitized 191-byte thread prefix plus full canonical-path SHA-256 yields only [A-Za-z0-9_-], length 1..256, unchanged under appends/root permutations/added copies; same thread different paths differs. Inject digest conflict to verify both candidates omitted with identifier_conflict.
  - Assert metadata-only path-hash lookup probes only the matching contained stream, rechecks recorded prefix, and shares direct-file ID derivation; stdin has no ID. Missing/mismatched candidate returns session not found.
  - Test nearest repository and gitdir worktree ownership, outside-repo all-projects only, normalised project matching and new nearer repository picked up on next refresh.
  - Exercise suppression before project filtering with stream/rollout in different projects and explicit file/exec-ID opening still available. Invalid/unreadable/outside/mismatched selected rollout and invalid first duplicate followed by valid duplicate must keep the stream, warn and never promote a later suppressing winner. Preserve Q62: a valid later duplicate remains its legacy project row while only the verified global winner controls stream suppression; assert both behaviors together.
  - Round-trip CLI and web rows from an unrelated directory for all six sources, duplicate/resumed streams and identical IDs across sources; verify labels, filter pill only with roots, source badges/CSSVersion, Thread ID, file modified and schema-valid listing/session documents.
  - Extend zero-body-read and changed/added/removed refresh assertions to exec heads, root warnings and repository changes without renaming unaffected IDs; use a 10,000-file synthetic ListAll benchmark to resolve the discovery risk.
  - Assert shared listing/latest extended ordering agrees, filesystem fallback stays distinct from source time, warning banners refresh, malformed-line transcript warnings surface, and -F rejects exec selected by latest or --source codex-exec.
  - Blocked-by: ep7jb4r (Green: implement platform identity/change adapters and pass the native capability gate), ep7jb4t (Green: integrate all-projects resolution and the verified long-lived listing cache), ep7jb4v (Green: parse Codex exec events and render final turn/item state through every output)
  - Stream: 1
  - Requirements: [1.1](../../requirements.md#1.1), [1.2](../../requirements.md#1.2), [1.4](../../requirements.md#1.4), [2.1](../../requirements.md#2.1), [2.5](../../requirements.md#2.5), [2.6](../../requirements.md#2.6), [2.7](../../requirements.md#2.7), [2.8](../../requirements.md#2.8), [2.10](../../requirements.md#2.10), [2.11](../../requirements.md#2.11), [4.1](../../requirements.md#4.1), [4.2](../../requirements.md#4.2), [4.3](../../requirements.md#4.3), [4.4](../../requirements.md#4.4), [5.1](../../requirements.md#5.1), [5.2](../../requirements.md#5.2), [5.3](../../requirements.md#5.3), [5.4](../../requirements.md#5.4), [5.5](../../requirements.md#5.5), [5.6](../../requirements.md#5.6), [5.7](../../requirements.md#5.7), [5.8](../../requirements.md#5.8), [5.9](../../requirements.md#5.9)
  - References: ../../design.md, ../../decision_log.md, ../../../../internal/sessions/lister_test.go, ../../../../internal/sessions/resolver_test.go, ../../../../cmd/apsis/main_test.go, ../../../../internal/apsisweb/handlers_test.go, ../../../../internal/apsisweb/integration_test.go

- [ ] 13. Green: integrate bounded-root exec discovery, stable IDs and globally verified rollout precedence <!-- id:ep7jb4x -->
  - Pass task 12 with sessions/codexexec.go, lister/resolver/source registry wiring, shared settings resolution and CLI/serve flags; no built-in roots, directory-link traversal, external git command or prompt-file read.
  - Canonicalize roots/targets to absolute resolved on-disk spelling, deduplicate paths, check containment before reads/opening, apply the approved skip policy and preserve other results with structured warnings.
  - Reuse ProbeExecStream and the verified fingerprint/cache from tasks 7/9. Derive IDs only from recorded thread ID and canonical path, omit both conflicts, discover containing repository per pass and resolve IDs by metadata-only path hash before one probe.
  - Run rollout collection first; share selectCodexCandidate with resolution/all-projects deduplication and suppress only its validated contained readable matching head globally, before stream scope filtering. Keep project rollout rows on legacy per-file cwd filtering (Q62) and explicit exec-file/exec-ID conversion available.
  - Complete source/format parity sites from the design: AllSources/displayNames/IsValidSource, collectors, extended sort, Resolve/ResolvePath/time context, configured bare-ID lookup, help, web sources/filter/badges/CSSVersion and parse/warning views. Follow/last-entry/orbit status paths must reject or remain outside exec scope as specified.
  - Display project only in all-projects text/web listings and file modified only for exec fallback labels; shared JSON retains explicit provenance for every source. Reject exec follow after ID/latest resolution with the named error.
  - Evaluate the 10,000-file benchmark and cold/refresh evidence before completion; use directory-entry caching only if the documented risk is observed while still statting known files. Finish with make build-apsis, make test and make lint.
  - No current execution is authorized by this draft. Require lifted source/canonical-task freeze, review/reconciliation and implementation admission; preserve every explicit human decision and do not claim approval for U1-U3.
  - Blocked-by: ep7jb4r (Green: implement platform identity/change adapters and pass the native capability gate), ep7jb4t (Green: integrate all-projects resolution and the verified long-lived listing cache), ep7jb4w (Red: test configured exec discovery, stable identity, rollout suppression and list/open refresh integration)
  - Stream: 1
  - Requirements: [1.1](../../requirements.md#1.1), [1.2](../../requirements.md#1.2), [1.4](../../requirements.md#1.4), [2.1](../../requirements.md#2.1), [2.5](../../requirements.md#2.5), [2.6](../../requirements.md#2.6), [2.7](../../requirements.md#2.7), [2.8](../../requirements.md#2.8), [2.10](../../requirements.md#2.10), [2.11](../../requirements.md#2.11), [4.1](../../requirements.md#4.1), [4.2](../../requirements.md#4.2), [4.3](../../requirements.md#4.3), [4.4](../../requirements.md#4.4), [5.1](../../requirements.md#5.1), [5.2](../../requirements.md#5.2), [5.3](../../requirements.md#5.3), [5.4](../../requirements.md#5.4), [5.5](../../requirements.md#5.5), [5.6](../../requirements.md#5.6), [5.7](../../requirements.md#5.7), [5.8](../../requirements.md#5.8), [5.9](../../requirements.md#5.9)
  - References: ../../design.md, ../../decision_log.md, ../../../../internal/sessions/types.go, ../../../../internal/sessions/lister.go, ../../../../internal/sessions/resolver.go, ../../../../cmd/apsis/main.go, ../../../../internal/apsisweb/handlers.go, ../../../../internal/apsisweb/server.go, ../../../../internal/apsisweb/static/style.css
