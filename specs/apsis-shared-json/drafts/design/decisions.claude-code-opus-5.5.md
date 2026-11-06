# Proposed Design Decisions: claude-code-opus-5.5 draft

Proposals for [draft.claude-code-opus-5.5.md](draft.claude-code-opus-5.5.md). IDs are local (P*, PQ*); merge assigns final Decision/Q numbers in `decision_log.md` (supersede mode). Nothing here is a human decision.

## Proposed Quick Decisions

| ID | Decision | Rationale |
|----|----------|-----------|
| PQ1 | New format value is `-f shared-json` | Names the outcome and the spec; sits beside `-f json` (raw) without implying a harness |
| PQ2 | All-projects is `--all-projects` (long form only) on `apsis` and `apsis serve`, with no environment variable | It selects a scope per invocation rather than configuring the machine; Q9's env-plus-flag rule targets settings such as exec roots |
| PQ3 | Source selection is `-s`/`--source <source>`, using source constants (`claude`, `codex`, `codex-exec`, `copilot`, `kiro-cli`, `kiro-ide`) | Distinct from `-a`, which forces a parser format and uses different names (Q21) |
| PQ4 | Exec roots: repeatable `--codex-exec-dir`; `APSIS_CODEX_EXEC_DIRS` split with `filepath.SplitList` | Repeatable flags need no escaping; the OS path-list separator is the documented convention for several directories (5.4) |
| PQ5 | One schema file `schemas/apsis-shared-json.schema.json` at the repo root, `kind` discriminator, `schema_version` integer field | 1.3 gives both documents one version; a root-level path is stable for consumers |
| PQ6 | Provider renders as `model (provider)`; provider alone as `(provider)` | Binds the provider to the model it served without adding a separator part to existing headers |
| PQ7 | Exec tool names are the item types (`command_execution`, `file_change`, `mcp_tool_call`, `web_search`); every exec tool item emits a `tool_result`, marked incomplete when uncompleted | Keeps source names, as the rollout parser does; the renderers draw non-Task tools only with their result |
| PQ8 | Usage block `basis` is `thread_total_at_turn_completion`; counters keep raw JSON values in source key order | Labels the 0.166.0 totals semantics (2.9) and keeps unknown or non-integer fields representable |
| PQ9 | Shared JSON emits `created_at` only for recorded times and `file_modified_at` for the fallback, for every source; human listing surfaces label the fallback only on `codex-exec` rows | Q26 requires a marked fallback; 5.1 forbids changing existing sources' text and web output |
| PQ10 | Listing `model`/`provider` come only from the bounded head read (50 lines, 1 MiB); the session document is authoritative | Listing stays cheap; 4.3 asks for these values "where known" |
| PQ11 | `ParseWarning.Unit` distinguishes physical JSONL lines from Kiro IDE chat-message indexes; JSON emits `line` or `chat_message` | Kiro IDE's `Line` today is a chat index; "source positions" must not mislabel it |
| PQ12 | Kiro IDE `metadata.modelProvider` is not surfaced | 5.1 allows output changes only for Codex providers; see question U3 |
| PQ13 | Under exec roots, `.jsonl` files whose first parseable event is not an exec event are ignored silently; exec-looking files without a valid `thread_id` are omitted with a warning | Roots can hold other JSONL; warning on every unrelated file would bury real problems (2.11) |
| PQ14 | A later `thread.started` with the same ID is dropped; one with a different ID becomes a generic `event` | Resumed output appended to one file reuses the ID (Q18); nothing recorded is lost |
| PQ15 | Session `created_at` = parser-recorded session start, else first parseable entry timestamp, else the resolver's recorded time | Same answer for file and resolved input where the transcript records it |
| PQ16 | Parse warnings appear in the web transcript view for `codex-exec` sessions only | 5.2 requires them for exec streams; 5.1 keeps other sources' web output unchanged; see question U1 |
| PQ17 | Exec malformed-line warnings use fixed text plus the JSON syntax error's byte offset | 5.2 forbids copying the line; `encoding/json` messages quote source characters |
| PQ18 | The `codex-exec` filter pill appears only when exec roots are configured | Keeps the unconfigured web page unchanged (5.1) |

---

## Proposed Decision P1: Extend `transcript.Entry`, Project to JSON Separately

**Date**: 2026-11-05
**Status**: proposed

### Context

Every parser already produces `[]transcript.Entry` and every renderer consumes it. `Entry` carries Claude's wire-format JSON tags, used to unmarshal Claude lines, so it cannot double as the published layout. Exec streams need content the type cannot express today: plans, errors, generic events, usage, exit codes, incomplete state, turns and a provider.

### Decision

Keep `Entry`/`ContentItem` as the single shared representation and add the new fields and content kinds with `json:"-"`. Add `internal/sharedjson` with its own tagged document types and a projection from `ParseResult` plus session context.

### Rationale

The user's framing is harness formats in, one representation out, rendered three ways. Extending the existing type gives Markdown and HTML the new content in the same change. A separate projection keeps the public contract independent of Claude's wire tags and of internal refactors.

### Alternatives Considered

- **Serialise `Entry` with new tags**: Least code - Rejected: its tags are Claude's input format; changing them breaks Claude parsing, and keeping them publishes a harness's wire format (1.2).
- **A new intermediate representation replacing `Entry`**: Cleaner long term - Rejected: rewrites five parsers and three renderers for no requirement.
- **Exec-only model rendered by its own code**: Isolated - Rejected: exec content would bypass the shared renderers and diverge from other sources in Markdown/HTML.

### Consequences

**Positive:**
- New content renders in Markdown, standalone HTML and the web view through existing functions.
- The JSON contract can only change through `internal/sharedjson` and the schema.

**Negative:**
- `ContentItem` grows fields only exec uses today.
- The projection must be updated whenever a parser starts filling a new field.

---

## Proposed Decision P2: Exec Identifier From Thread ID and Canonical-Path Hash

**Date**: 2026-11-05
**Status**: proposed

### Context

2.8 requires a bounded, URL-safe identifier derived from the thread ID, unique within `codex-exec`, and unchanged by appends, added copies, overlapping roots or root order. Copies and resumed runs share thread IDs (Q18).

### Decision

Identifier = `sanitise(thread_id)` truncated to 239 bytes, `-`, then the first 16 hex characters of SHA-256 of the canonical path. The suffix is always present. A hash collision between two paths omits the lexically later path with a warning.

### Rationale

The identifier depends only on facts about the stream itself, so no other file can rename it. Hashing the path also lets the resolver find the file without reading any other stream's body.

### Alternatives Considered

- **Bare thread ID, suffix only on collision**: Friendlier when unique - Rejected: adding a copy renames whichever file gains the suffix.
- **Ordinal suffix by traversal order**: Short - Rejected: root order and added files change it.
- **Hash of path relative to its root**: Shorter input - Rejected: overlapping roots give one file two relative paths.
- **Content hash**: Path-independent - Rejected: appends change it.

### Consequences

**Positive:**
- Identifiers stay stable under every change 2.8 lists.
- Opening by identifier reads one stream head.

**Negative:**
- Moving or renaming a stream changes its identifier (accepted in Decision 9).
- On case-insensitive filesystems, roots spelled with different case can list one file twice under two identifiers. Roots should be configured with on-disk case.

---

## Proposed Decision P3: Stat-Keyed Metadata Cache in a Long-Lived Lister

**Date**: 2026-11-05
**Status**: proposed

### Context

5.9 requires unchanged serve refreshes to read zero transcript-body bytes. Today every listing reads Claude and Codex heads, whole Kiro IDE `.chat` files and Kiro CLI rows on each 15-second poll.

### Decision

The `Lister` keeps a mutex-guarded cache keyed by file path. The key holds `(size, mtime, ctime, inode)` where the platform provides them, and the value holds the extracted listing metadata. Each pass enumerates and stats, re-reads heads only on key change, and sweeps unseen entries. Kiro CLI results are keyed by the stat of `data.sqlite3` and `data.sqlite3-wal`.

### Rationale

Directory enumeration and stat are metadata operations, so the cache meets the zero-read rule directly and can be tested with a counting opener. Adds, removals and edits show on the next pass without watchers.

### Alternatives Considered

- **Filesystem watchers (fsnotify)**: Push updates - Rejected: new dependency, recursive-watch limits on macOS/Linux, and a missed event leaves stale state.
- **Time-based expiry**: Simple - Rejected: re-reads unchanged bodies on expiry, violating 5.9.
- **No cache**: Simplest - Rejected: fails 5.9.

### Consequences

**Positive:**
- Unchanged refreshes read no bodies; changed files cost one head read.
- CLI behaviour is unchanged (a fresh `Lister` per command).

**Negative:**
- Listings are serialised by the mutex.
- Change detection relies on stat fidelity (see the Kiro CLI risk in the draft).

---

## Proposed Decision P4: Open Published Schema, Closed Test Copy, Test-Only Validator Library

**Date**: 2026-11-05
**Status**: proposed

### Context

1.3 requires additive fields to stay compatible with consumers that ignore unknown fields. 1.4 requires fixtures to conform to the published schema. The handoff asks for no heavy dependency.

### Decision

Publish the schema without `additionalProperties: false`. In tests, compile a copy with `additionalProperties: false` added to every object schema, and validate every fixture document with `github.com/santhosh-tekuri/jsonschema/v6` v6.0.3. Its only non-test dependency, `golang.org/x/text`, is already in the module graph. The library is imported only from `_test.go` files.

### Rationale

The open schema matches the compatibility rule. The closed copy makes CI fail when code emits a field the schema does not declare. A maintained validator avoids hand-written keyword semantics.

### Alternatives Considered

- **Hand-written validator for the keyword subset used**: No dependency - Rejected: validator bugs would pass bad output; needs its own tests.
- **Generate the schema from Go types**: Single source - Rejected: needs a generator dependency and still does not check emitted values such as `oneOf` discriminators.
- **Golden JSON only**: No schema tooling - Rejected: does not show conformance to the published schema (1.4).

### Consequences

**Positive:**
- Undeclared fields and type drift fail tests.
- Binaries gain no dependency.

**Negative:**
- `go.mod` gains a direct requirement used only by tests (question U2).
- The closed-copy transform must cover every object schema, including those in `$defs`.

---

## Proposed Decision P5: Resolve Exec Lifecycle per Turn and Item at End of Stream

**Date**: 2026-11-05
**Status**: proposed

### Context

The pinned 0.166.0 emitter numbers items `item_N` per process, so appended resumed output can reuse `item_0`. Items interleave, receive updates (todo lists), and may never complete when a turn is interrupted. 2.3 requires one final state per item within its turn, in first-appearance order, without rendering lifecycle updates.

### Decision

Key records by `(turn ordinal, item id)`, with turn 0 before the first `turn.started`. Keep first-appearance order and replace the stored item JSON on each event. Translate final states at EOF. Turn-level and stream-level events (`turn.completed`, `turn.failed`, `error`, unknown) become their own ordered records.

### Rationale

Per-turn keys survive ID reuse across appended runs. Translating once at EOF keeps lifecycle handling out of the item translators.

### Alternatives Considered

- **Key by item ID alone**: Simpler - Rejected: collides when resumed output is appended to one file.
- **Emit content as events arrive**: Streaming - Rejected: needs in-place replacement of multi-block items and still cannot know completion before EOF.
- **Order by completion**: Matches final state - Rejected: breaks first-appearance ordering of interleaved tools.

### Consequences

**Positive:**
- Interleaved tools keep their links and their order.
- Incomplete items are known exactly.

**Negative:**
- The whole stream is held in memory before output, which suits exec streams but is not a streaming parser.

---

## Proposed Decision P6: All-Projects Resolution Shares the Listing's Selection Rules

**Date**: 2026-11-05
**Status**: proposed

### Context

`ListAll("")` already lists every project. Opening does not match it: Kiro CLI and Kiro IDE resolution turn an empty project into the current directory. Claude, Codex rollout and Kiro IDE IDs can repeat across directories, so two listed rows could open the same content.

### Decision

Give the resolver explicit all-projects branches: `GetSessionByID` for Kiro CLI, and a scan of every workspace for Kiro IDE. Have listing and resolution call the same candidate-selection functions. Within one listing a `(source, id)` pair appears once; duplicates keep the resolver's choice and add a warning.

### Rationale

Listing and opening derive their answer from one rule, so every listed row opens the session it describes from any directory, in separate CLI processes and in serve.

### Alternatives Considered

- **Locator stored by the lister and used by the resolver**: Exact - Rejected: the CLI lists and opens in separate processes, so state cannot be shared.
- **Encode the location in the identifier**: Self-contained - Rejected: paths and DB keys are unsafe in URLs and change identifiers when files move.
- **Keep duplicates**: No dedupe - Rejected: a second row would open the first row's content (4.2).

### Consequences

**Positive:**
- 4.2 and 5.8 hold for every source.
- Kiro IDE selection is defined once instead of twice with different tiebreaks.

**Negative:**
- Opening Kiro IDE in all-projects scope reads every workspace's `.chat` headers.
- Rare duplicate rollout files now list once with a warning, where they previously listed twice.

---

## Proposed Decision P7: Session Metadata Comes From the Transcript First

**Date**: 2026-11-05
**Status**: proposed

### Context

The session document needs creation time and project for file input, stdin and resolved sessions. Some sources record these inside the transcript (Codex `session_meta`, Kiro IDE `startTime`, Claude `cwd`). Others record them only beside it (Copilot `workspace.yaml`, the Kiro CLI DB key), and resolvers sometimes substitute file times.

### Decision

Derive document metadata from parsed content first. Exec streams take their project from the containing repository of the input path. Resolver metadata fills only fields still missing, and never a filesystem-flagged time.

### Rationale

The same file gives the same document whether opened by path or by identifier wherever the transcript records the fact. Resolver data adds what only the listing knows without presenting file times as source times (Q26).

### Alternatives Considered

- **Resolver metadata first**: Matches listings exactly - Rejected: file and stdin input would produce different documents for the same content.
- **Parsed content only**: Simplest - Rejected: Copilot and Kiro CLI documents would lose a project the listing shows.

### Consequences

**Positive:**
- Consistent documents across input modes for transcript-recorded facts.

**Negative:**
- For Copilot and Kiro CLI, listing and session `created_at` can differ slightly, because they come from different records.

---

## Questions for the User

These need the human; merge should move them into `drafts/design/questions.md`.

- **U1 — Parse warnings in the web transcript view.** A: only for `codex-exec` (recommended; meets 5.2 and keeps 5.1). B: for every source, which changes existing web output and needs 5.1 amended.
- **U2 — Test-only schema validator dependency.** A: add `santhosh-tekuri/jsonschema/v6` to `go.mod`, imported only by tests (recommended, P4). B: hand-written validator for the keyword subset, with no new module.
- **U3 — Kiro IDE `modelProvider`.** A: leave it out now and propose a follow-up that amends 3.1/5.1 (recommended). B: amend now and display it for Kiro IDE too.
