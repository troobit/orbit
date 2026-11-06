# Design: Apsis — Shared JSON Output, Codex Exec Streams and Cross-Project Listing

## Overview

`transcript.Entry` stays the one shared representation that every parser produces and every renderer consumes. This feature adds the content kinds exec streams need, a provider field, and a separate projection of that representation into versioned, harness-neutral JSON (`-f shared-json`). `codex exec --json` streams become source `codex-exec`, discovered only under configured roots. The existing `ListAll("")` path becomes reachable through `--all-projects`, and resolution gains all-projects branches so every listed session opens from any directory. Requirements are in [requirements.md](requirements.md); decisions referenced below are in [decision_log.md](decision_log.md).

## Architecture

```
harness files ──parse──▶ transcript.ParseResult (Entries, Warnings, Metadata, Format)
                               │
            ┌──────────────────┼──────────────────────────┐
     RenderMarkdown     RenderHTML / RenderHTMLFragment   sharedjson.Session
                                                          sharedjson.Listing ◀── sessions.Lister.ListAll
```

### Placement

| Path | Contents |
|---|---|
| `internal/transcript/types.go`, `parser.go` | New `Entry`/`ContentItem` fields, `FormatCodexExec`, `ParseResult.Format`, metadata fields, warning codes/units, exec probe in `DetectFormat` |
| `internal/transcript/codex_exec_types.go`, `codex_exec_parser.go` (new) | Exec event structs, `ParseCodexExec`, `ProbeExecStream` |
| `internal/transcript/codex_types.go`, `codex_parser.go` | `model_provider` capture and propagation |
| `internal/transcript/markdown.go`, `html.go`, `metadata.go` | Rendering of new blocks; `modelLabel` |
| `internal/sharedjson/` (new) | Document types, `Session`, `Listing`, `Write`, `SchemaVersion`; imports `transcript` and `sessions`, so neither imports it |
| `docs/schemas/apsis-shared-json.v1.schema.json` (new) | Published JSON Schema (draft 2020-12) for both document kinds |
| `internal/sessions/codexexec.go` (new) | Root canonicalisation, walk, qualification, identifier, repository root, suppression |
| `internal/sessions/cache.go` (new) | Stat-keyed listing-metadata cache |
| `internal/sessions/lister.go`, `resolver.go`, `types.go` | New source, listing fields, all-projects resolution, de-duplication, ordering |
| `internal/sessions/parse.go` (new) | `ParseResolved`, shared by CLI and web |
| `internal/agents/kiro/logs/session.go`, `discover.go` | `GetSessionByID`; deterministic tie order in `DiscoverAll` |
| `cmd/apsis/main.go` | Flags, input context, shared-JSON dispatch, follow checks |
| `internal/apsisweb/` | Config, list and transcript views, warning banner in the polled fragment, source badge CSS |
| `internal/transcript/testdata/codex_exec/` (new) | Synthetic exec fixtures plus `PROVENANCE.md` |

### Parity audit: new source and format

| Site | Needs equivalent | Note |
|---|---|---|
| `sessions/types.go` `allSources`, `displayNames`, `IsValidSource` | yes | `codex-exec` |
| `Lister.ListAll` collectors | yes | `listCodexExec` runs after `listCodex`, which supplies the suppression index |
| `sortSessionsByTimestamp` | yes | priority 5, then identifier tiebreak for every source |
| `Resolver.Resolve`, `ResolvePath` | yes | `ResolvePath` returns the stream path; follow callers reject it |
| `Resolver.resolveFileCreatedAt` | yes | exec returns no recorded time; modification time only as the flagged fallback |
| `cmd/apsis` `resolveInput` bare-ID source list | yes | `codex-exec` last, only when roots are configured |
| `cmd/apsis` `resolveFollowInput`, `runLatest` `fileBackedSources` | no | exec is rejected with a named error ([Error Handling](#error-handling)) |
| `cmd/apsis` `agentToFormat`, `-a`/`-f` help | yes | `codex-exec`, `shared-json` |
| `cmd/apsis` `convertToJSON` | yes | strips a leading BOM for exec input; the JSONL line path is otherwise unchanged |
| `transcript` `DetectFormat`, `detectJSONLFormat`, `Parse`, `ParseJSONLWithFormat` | yes | exec probe and dispatch |
| `transcript` `detectFormatFromLine`, `ParseJSONL` | no | used by follow mode and orbit's web view, neither of which accepts exec |
| `apsisweb` `ValidateSource` | yes | via `IsValidSource` |
| `apsisweb` `buildSessionListData` `Sources` | yes | `codex-exec` pill only when roots are configured |
| `apsisweb` `handleTranscript` parse branch | yes | replaced by `sessions.ParseResolved` |
| `apsisweb/static/style.css` badges, `CSSVersion` | yes | `.source-codex-exec` light and dark; bump version |
| `internal/web` (orbit), `internal/status` last-entry | no | Claude and Codex rollouts only |

### Parity audit: new content kinds

| Site | Needs equivalent | Note |
|---|---|---|
| `markdown.go` `formatAssistantMessage` | yes | `plan`, `error`, `event`, `usage`, incomplete state |
| `html.go` `formatAssistantMessageHTML` | yes | both the `hasContent` check and the render pass, else entries holding only new kinds vanish |
| `formatToolInput`, `formatToolInputHTML` | yes | `command_execution`, `file_change`, `mcp_tool_call`, `web_search` |
| `formatToolResult`, `formatToolResultHTML` | yes | exit code, incomplete marker |
| `formatUserMessage`, `formatUserMessageHTML` | no | exec entries are assistant entries |
| `grouping.go` `isReadOnlyEntry`, `isEditOnlyEntry` | no | the `default` branch rejects unknown kinds; exec tool names are never `Read`/`Edit` |
| `BuildToolMeta`, `preprocessEntries` result map | no | exec tool IDs are turn-qualified, so transcript-wide maps keyed by tool ID stay unambiguous |
| `last_entry.go` | no | orbit status only |

### Parity audit: provider

| Site | Needs equivalent | Note |
|---|---|---|
| `codex_parser.go` `processSessionMeta`, `ensureAssistantEntry` | yes | provider state beside model state; an entry takes both when it is created |
| Assistant header meta in `formatAssistantMessage` and `formatAssistantMessageHTML` | yes | `modelLabel(entry.Model, entry.Provider)` |
| Read/Edit group headers, user headers, slash commands, deferred tool-result sections | no | they show no model today; the provider appears wherever the model appears (Decision 16) |
| Copilot, Kiro CLI, Kiro IDE parsers | no | 3.1 and 5.1 limit provider output to Codex (question U2 covers Kiro IDE) |

## Components and Interfaces

### 1. Shared representation additions (`transcript`)

```go
type Entry struct {
    // existing fields unchanged
    Provider string `json:"-"` // recorded provider; set only by the Codex rollout parser
    Turn     int    `json:"-"` // 1-based exec turn; 0 = no turn
}

type ContentItem struct {
    // existing fields unchanged; new fields are `json:"-"` so Claude unmarshalling is unaffected
    ItemID     string          // source item identifier (exec)
    Status     string          // source item status as recorded
    Incomplete bool            // started, never completed before EOF
    ExitCode   *int            // command exit code; nil when unrecorded
    Plan       []PlanItem      // Type "plan"
    EventType  string          // Type "event": source event or item type
    Payload    json.RawMessage // Type "event": parsed payload, compact
    Usage      *Usage          // Type "usage"
    // Type "error" uses Text for the message
}
type PlanItem struct { Text string; Completed bool }
type Usage struct {
    Basis    string         // "thread_total_at_turn_completion" for Codex 0.166.0
    Counters []UsageCounter // source key order; unknown keys kept
}
type UsageCounter struct { Name string; Value json.RawMessage }

type ParseResult struct { /* existing */ Format Format } // format parsed; FormatUnknown for the Claude fallback
type ParseResultMetadata struct { /* existing */ StartedAt, Project, ThreadID string }
type ParseWarning struct { Line int; Message string; Code string; Unit string } // Unit "" = physical line, "chat-message" = Kiro IDE chat index
```

`ParseResult.Metadata` may now be non-nil with a nil `TotalCost`; renderers already guard on `TotalCost != nil`. The Kiro IDE parser sets `Unit: "chat-message"` on its existing warnings.

Session-level metadata comes from parsed content first; resolver metadata fills only fields still missing and never supplies a filesystem time as `created_at` (Decision 19):

| Source | `created_at` | `project` | `model` | `provider` |
|---|---|---|---|---|
| claude | first parseable entry timestamp | first entry `cwd` | omitted (question U1) | omitted |
| codex | `session_meta.timestamp` | `session_meta.cwd` | first entry model | `session_meta.model_provider` |
| copilot | first entry timestamp | resolver: `workspace.yaml` `git_root`, else `cwd` | first entry model | omitted |
| kiro-cli | first entry timestamp | resolver: database `key` | first entry model | omitted |
| kiro-ide | `metadata.startTime` | omitted (the workspace hash is one-way) | first entry model | omitted (question U2) |
| codex-exec | omitted | containing repository of the input path; none for stdin | omitted | omitted |

### 2. Codex exec stream parser

`ParseCodexExec(r io.Reader) (*ParseResult, error)` returns an error only on reader failure. An empty or event-less stream yields zero entries. Lines are decoded with `json.Decoder.UseNumber` or kept as `json.RawMessage`, so numbers in inputs, payloads and usage keep their precision.

**Detection** (2.1, Decision 20). `ProbeExecStream(r)` reads whole lines until the first line that parses as a JSON object, skipping blank, malformed and over-10-MiB lines, and classifies the stream: `exec` when that object is `thread.started` with a nonempty string `thread_id`; `exec-without-thread` when its `type` is another exec event (`thread.started` without a valid ID, `turn.*`, `item.*`, `error`); `other` otherwise. `DetectFormat` strips the BOM and applies the same test to the first parseable object in its 64 KiB chunk before Kiro detection, whose truncated Kiro IDE check matches on substrings that an exec stream's text can contain; a Kiro file's first line is never a complete `thread.started` object. When the chunk holds no parseable object and is not Kiro, it continues reading lines until one parses and returns every consumed byte for replay; that extension only decides between `FormatCodexExec` and the existing result, so detection of other formats is unchanged. `-a codex-exec` forces the parser. Discovery uses the same probe.

**Lifecycle** (2.3, Decision 15). The parser keeps an ordered list of turns. Turn 0 holds records before the first `turn.started`; each `turn.started` opens the next turn. Within a turn, records are keyed and kept in first-appearance order:

| Event | Key | Effect |
|---|---|---|
| `thread.started` | — | the first nonempty `thread_id` becomes `Metadata.ThreadID`; a later one with the same ID is dropped and does not reset turn numbering; a different ID becomes an `event` record plus a `thread_mismatch` warning |
| `item.started`, `item.updated`, `item.completed` | `(turn, item.id)`; a missing ID keys by physical line | first sighting appends; later events replace the stored item JSON; `completed` marks it done |
| `turn.completed` | own record | `usage` block |
| `turn.failed`, top-level `error` | own record | `error` block with `message`; the original JSON is kept as payload |
| any other event type | own record | `event` block |

At EOF each turn becomes one assistant `Entry` with `Turn` set and no timestamp, model or provider. Its content is the translation of each record's final state. Items without completion get `Incomplete: true`, on their tool result as well. Lifecycle events never render separately.

**Item translation** (2.2; shapes from the pinned 0.166.0 `exec_events.rs`):

| Item `type` | Content |
|---|---|
| `agent_message` | `text` |
| `reasoning` | `thinking` |
| `command_execution` | `tool_use` (name `command_execution`, input `{command}`) + `tool_result` (content `aggregated_output`, `ExitCode`, `Status`; `IsError` when status is `failed`/`declined` or exit code ≠ 0) |
| `file_change` | `tool_use` (input `{changes}`) + `tool_result` (one `kind path` line per change, `Status`; `IsError` when `failed`) |
| `mcp_tool_call` | `tool_use` (input `{server, tool, arguments}`) + `tool_result` (compact JSON of `result`, or `error.message`; `IsError` when `failed` or `error` is set) |
| `web_search` | `tool_use` (input `{query, action}`) + `tool_result` (compact JSON of `results` when present) |
| `todo_list` | `plan` |
| `error` | `error` |
| anything else, including `collab_tool_call` | `event` with `EventType` = item type, payload = final item JSON |

Every exec tool item emits its `tool_result`, because the renderers draw non-Task tools only when their result arrives. Tool IDs are `t<turn>:<item id>`; `ItemID` keeps the original. A known item type whose payload fails typed decoding becomes an `event` block with an `item_decode_failed` warning, so unverified shapes fall back to generic handling (5.6).

**Warnings** (5.2). A malformed line yields `ParseWarning{Line: <physical 1-based line>, Code: "malformed_line", Message: "malformed JSON event"}`, with ` (offset N)` appended for a `*json.SyntaxError`. A line over 10 MiB is discarded through its newline with `oversized_line`. Messages never contain source text. Parsing continues at the next line; I/O errors are fatal.

### 3. Shared JSON projection (`internal/sharedjson`)

```go
const SchemaVersion = 1
type SessionContext struct {
    Source   string                    // from the resolver; else from ParseResult.Format
    ID       string                    // listing identifier when known
    Path     string                    // canonical input path; "" for stdin
    Resolved *sessions.SessionMetadata // nil for file and stdin input
}
func Session(res *transcript.ParseResult, ctx SessionContext) SessionDocument
func Listing(list []sessions.SessionInfo, warnings []sessions.ListWarning, scope Scope) ListingDocument
func Write(w io.Writer, doc any) error // json.Encoder, two-space indent, SetEscapeHTML(false), trailing newline
```

- The projection reads parsed data only and never calls renderer helpers, so content is never shortened (1.5, Q29).
- Document-level `model`/`provider` are the first nonempty entry values in transcript order; entries keep their own values (1.6).
- Root arrays are `[]`, never `null`.
- `convert` dispatches shared JSON before its `Session contains no entries` early return, so an empty session still emits a document.
- File and stdin conversion need no listing: everything comes from the parse result and the input context.

### 4. Listing, scope and cache (`sessions`)

```go
type SessionInfo struct {
    ID, Source string; CreatedAt time.Time; Size int64 // existing; CreatedAt stays the sort time
    TimeFromFilesystem bool  // CreatedAt is the file modification time, not a recorded time
    Project            string
    Model, Provider    string // when the listing read yields them
    ThreadID           string // codex-exec only
}
type SessionMetadata struct { /* existing */ TimeFromFilesystem bool; Project, ThreadID, Path string }
type ListWarning struct { Source string; Code string; Err error } // Code added

type Option func(*Lister)
func WithCodexExecRoots(roots []string) Option
func NewLister(opts ...Option) (*Lister, error)
func NewResolver(projectPath string, opts ...Option) (*Resolver, error) // "" = all projects
func (l *Lister) Sources() []string // AllSources minus codex-exec when no roots are configured
```

`ListAll("")` is the all-projects listing. A `Lister` is long-lived in `apsis serve` and per-command in the CLI (Decision 13). Each pass:

1. Holds `l.mu` for the whole pass, so a page load and an HTMX poll never interleave cache updates.
2. Enumerates directories and stats candidates and their companions without opening transcript bodies.
3. Looks up `cache[path]`. The key is `(size, mtime, ctime, inode, device)`, with ctime, inode and device where the platform provides them (Windows falls back to size and mtime). Companion files are part of the key, and an absent companion is recorded as absent, so creating one changes the key.
4. On a hit, reuses the cached metadata. On a miss, runs the source's extraction (table below), restats, and caches the result only when the key is unchanged; a changed key uses the result for this pass and retries on the next.
5. Omits a changed file whose extraction fails, with a warning, rather than serving its previous metadata.
6. Sweeps cache entries not seen in the pass.
7. Returns a freshly allocated slice; callers such as the web handler's `slices.Reverse` cannot reorder cached state.

| Source | Key files | Extracted metadata | Project | Time |
|---|---|---|---|---|
| claude | transcript | first timestamp and first `cwd`, within 50 lines and 1 MiB | first `cwd` | first timestamp, else mtime (flagged) |
| codex | rollout | one `session_meta` scan (same limits): `id`, `timestamp`, `cwd`, `model`, `model_provider` | `cwd` | `timestamp`, else mtime (flagged) |
| copilot | `events.jsonl`, `workspace.yaml` | `workspace.yaml` fields | `git_root`, else `cwd` | `created_at`, else mtime (flagged) |
| kiro-cli | `data.sqlite3`, `-wal`, `-journal` | `DiscoverAll` result; unchanged keys skip opening the database | database `key` | `created_at` |
| kiro-ide | `.chat` file | execution ID, chat count, `startTime`, `modelId` (whole file, as today) | omitted | `startTime`, else mtime (flagged) |
| codex-exec | stream | probe result and `thread_id` | containing repository | mtime (flagged) |

Repository roots for exec streams are found by statting ancestors on each pass, memoised per directory within the pass only, so a newly created nearer repository is picked up on the next pass.

Listing `model`/`provider` come only from these extraction reads; the session document is authoritative.

**Ordering and uniqueness** (4.4). Sort by `CreatedAt` ascending; zero times sort first. Ties break by source priority (claude, copilot, codex, kiro-cli, kiro-ide, codex-exec), then identifier. In all-projects scope a `(source, id)` pair appears once: a duplicate keeps the candidate the resolver opens and adds a `duplicate_session` warning. The resolver's choice is first in sorted directory order for Claude, first in walk order for Codex rollouts, the shared best-candidate rule for Kiro IDE, and most recent `updated_at` then lowest `key` for Kiro CLI.

### 5. Exec discovery, identity, project and suppression

**Roots** (2.5, 5.5). Each configured root is made absolute, passed through `filepath.EvalSymlinks`, and rewritten to its on-disk spelling component by component (an exact name in the parent's listing wins; otherwise a unique case-insensitive match), so case aliases on case-insensitive volumes canonicalise to one path. Identical canonical roots collapse to one. A root that fails yields a `root_unreadable` warning and is skipped. `filepath.WalkDir` walks each root and never descends directory symlinks. Subdirectories named `.git`, `node_modules` or `testdata` are skipped during descent; a configured root is scanned whatever its name, including when it is nested inside another root's skipped subdirectory. A `.jsonl` regular file is a candidate at its walked path. A `.jsonl` symlink is resolved and canonicalised the same way, and kept only when the target is a regular file inside some canonical root. Candidates are de-duplicated by canonical path. Unreadable directories and files yield warnings and the walk continues.

**Qualification** (2.1, 2.11). The cached probe result decides: `exec` is listed; `exec-without-thread` is omitted with a `missing_thread` warning; `other` is ignored silently.

**Identifier** (2.8, Decision 14). `sanitise(thread_id)[:191] + "-" + hex(sha256(canonical path))`, at most 256 characters. `sanitise` maps every byte outside `[A-Za-z0-9_-]` to `_`. The identifier depends only on the stream's own canonical path and recorded thread ID, so appends, added copies, overlapping roots and root order cannot change it. If two distinct canonical paths produce the same identifier, both are omitted with an `identifier_conflict` warning. `ThreadID` keeps the original value. Direct file conversion computes the same identifier without configured roots; stdin has none.

**Project** (2.6). Walk up from the stream's directory to the nearest ancestor holding `.git`: a directory for a repository, or a file starting with `gitdir:` for a worktree. No `git` executable is invoked. Project scope keeps a stream only when `normalizePath(root) == normalizePath(project)`. A stream outside any repository lists only in all-projects scope.

**Suppression** (2.7, Decision 17). `listCodex` scans every rollout's `session_meta` in every scope before applying its project filter. The suppression index holds each rollout's recorded `session_meta.payload.id` when it equals, ignoring case, the UUID in that rollout's filename, which is what `findCodexSession` opens. A stream whose `thread_id` is in the index is dropped before project filtering, in every scope. If rollout listing fails, the index is incomplete and streams are kept; the rollout warning is already reported. Suppression never blocks opening a stream by file or by its `codex-exec` identifier.

### 6. Resolution in all-projects scope

`Resolver` with `projectPath == ""`:

| Source | Behaviour |
|---|---|
| claude | existing all-projects search, sorted directory order |
| codex, copilot | unchanged (already global) |
| kiro-cli | new `logs.(*DB).GetSessionByID(ctx, id)`: `SELECT key, value … WHERE conversation_id = ? ORDER BY updated_at DESC, key ASC LIMIT 1`; `DiscoverAll` uses the same order, so both pick the same row; returns `Project = key` |
| kiro-ide | scans every workspace under `KiroIDEBasePath` with the `bestKiroIDECandidate` function the lister also uses (most chat entries, newest mtime, lexically smallest path); `CostPath` derives from the winning workspace |
| codex-exec | both scopes: take the 64-hex suffix, hash each candidate canonical path from a metadata-only walk of the roots, probe only the matching file, require its sanitised thread prefix to match, recheck root containment on the resolved path, then open |

These branches stop `logs.GetSession` and `KiroIDEWorkspaceDir` from turning `""` into the current directory through `filepath.Abs`. The resolver also returns `Path`, `Project`, `ThreadID` and `TimeFromFilesystem` in `SessionMetadata`.

`sessions.ParseResolved(rs *ResolvedSession) (*transcript.ParseResult, error)` replaces the duplicated parse branch in `cmd/apsis` and `apsisweb`: it forces `FormatKiroIDE` with the cost path, forces `FormatCodexExec` for codex-exec, and auto-detects otherwise.

### 7. CLI surface (`apsis`)

| Option | Behaviour |
|---|---|
| `-f shared-json` | session document for an ID, file, stdin or `latest`; listing document with `--list` |
| `--all-projects` | lists, `latest` and ID lookups use project `""`; usage error with `-p`/`--project`; no environment variable (Q31) |
| `-s`, `--source <source>` | the positional argument is an identifier of that source, using source names (`claude`, `codex`, `codex-exec`, `copilot`, `kiro-cli`, `kiro-ide`); no file-path detection; usage error without an argument, with an unknown source, or with stdin, `--list` or `latest` |
| `--codex-exec-dir <dir>` (repeatable) | exec roots; any occurrence replaces `APSIS_CODEX_EXEC_DIRS` entirely; a single empty value means no roots; mixing empty and nonempty values is a usage error |
| `APSIS_CODEX_EXEC_DIRS` | split with `filepath.SplitList` (`:` on Unix, `;` on Windows); unset or empty means no roots |
| `-a codex-exec` | forces the exec parser |

`apsis serve` accepts `--all-projects` and `--codex-exec-dir`, and reads the same environment variable (5.4). Relative roots resolve against the working directory.

`resolveInput` returns an input context: reader, source, ID, canonical path, cost path and resolver metadata. `convert` builds `SessionContext` from it. For exec input, the Markdown/HTML header shows `Thread ID` from content and `Session ID` (the derived identifier) when a path exists; it never uses the filename stem (2.11). Text listing output is unchanged except:

- In all-projects scope a fifth tab-separated column holds the project, or `-`.
- codex-exec rows print the time followed by ` (file modified)`.
- `apsis latest` on a codex-exec session adds the same suffix to its `Using … from …` stderr line.

### 8. `apsis serve`

`apsisweb.Config` gains `AllProjects bool` and `CodexExecRoots []string`. One `Lister` lives for the server's lifetime and carries the cache across refreshes. `SessionView` gains `Project` (all-projects scope only) and `FileTime bool`, which renders `file modified` beside codex-exec times. The `sessions_list` fragment renders the warning banner, so warnings follow each refresh; the page includes the fragment, so the banner is not duplicated. `TranscriptViewData` gains `Warnings`. For codex-exec sessions only, the transcript view shows parse warnings with the existing `.warning-banner` style and shows thread ID and modification time in `session-meta` (Q47).

### 9. Rendering of new content

The existing patterns are the baseline; only differences are listed.

| Content | Markdown | HTML (standalone and fragment share `renderEntriesToBuilder`) |
|---|---|---|
| assistant meta | `modelLabel`: `model (provider)`, provider alone `(provider)`; exec turns append `turn N` as another ` · ` part | same parts via `FormatMessageMetaHTML` |
| `command_execution` input | same block as `Bash` | same as `Bash` in `formatToolInputHTML` |
| `file_change` input | `- kind path` list | same list markup |
| exit code | `**Result (exit code N):**` | same label |
| incomplete item | ⏳ and `(incomplete)` in the summary instead of ✅/❌ | same |
| `plan` | checkbox list matching the `TodoWrite` rendering | `TodoWrite` markup |
| `error` | `❌ **Error:** message` | failed-tool styling |
| `event` | collapsed `<details>` `📦 Event: type` with an escaped JSON block | collapsed block like non-Task tools; JSON escaped, never raw HTML |
| `usage` | `**Token usage (thread totals at turn completion):** name value · …` | same text in a `usage` div; one CSS rule |

## Data Models

### Session document

```json
{
  "schema_version": 1,
  "kind": "session",
  "session": {
    "source": "codex-exec",
    "id": "0199aaaa-0000-7000-8000-000000000001-3f2a9c0d1b7e4a56c8d90e1f2a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d",
    "thread_id": "0199aaaa-0000-7000-8000-000000000001",
    "project": "/work/example-repo"
  },
  "entries": [
    {
      "role": "assistant",
      "turn": 1,
      "content": [
        { "type": "tool_use", "id": "t1:item_1", "item_id": "item_1", "name": "command_execution", "input": { "command": "make test" } },
        { "type": "tool_result", "tool_use_id": "t1:item_1", "item_id": "item_1", "content": "ok\n", "is_error": false, "exit_code": 0, "status": "completed" },
        { "type": "plan", "item_id": "item_2", "incomplete": true, "items": [{ "text": "Run tests", "completed": true }] },
        { "type": "usage", "basis": "thread_total_at_turn_completion", "counters": { "input_tokens": 1200, "cached_input_tokens": 800, "output_tokens": 90 } }
      ]
    }
  ],
  "warnings": [{ "code": "malformed_line", "message": "malformed JSON event (offset 14)", "line": 4 }]
}
```

| Field | Derivation | Omitted when |
|---|---|---|
| `session.source` | resolver source, else `ParseResult.Format` | Claude fallback for unrecognised input |
| `session.id` | listing identifier; for file input the existing filename stem, or the derived exec identifier | stdin; exec without thread |
| `session.thread_id` | `Metadata.ThreadID` | not exec |
| `session.created_at`, `project`, `model`, `provider` | metadata table in [Components 1](#1-shared-representation-additions-transcript) | unknown |
| `session.cost` | `{amount, unit}` from `TotalCost`/`CostUnit` | `TotalCost` nil |
| entry | `role` (message role, else entry type), `timestamp`, `model`, `provider`, `turn`, `content` | empty values |
| block common | `type`, `item_id`, `status`, `incomplete` | empty/false |
| block types | `text` and `thinking` (`text`); `tool_use` (`id`, `name`, `input`); `tool_result` (`tool_use_id`, `content`, `is_error` always present, `exit_code` emitted for 0 and omitted only when unrecorded); `plan` (`items`); `error` (`message`); `event` (`event_type`, `payload`); `usage` (`basis`, `counters`); `other` (`source_type`, `text`) for parsed content kinds outside this list | — |
| `warnings[]` | `code`, `message`, and `line` (physical line) or `chat_message` (Kiro IDE index) | no position |

### Listing document

```json
{
  "schema_version": 1,
  "kind": "listing",
  "scope": { "all_projects": true },
  "sessions": [
    { "source": "codex-exec", "id": "0199aaaa-…-3f2a9c0d…", "thread_id": "0199aaaa-…", "size_bytes": 5120,
      "file_modified_at": "2026-10-01T09:00:00Z", "project": "/work/example-repo" },
    { "source": "codex", "id": "0199bbbb-…", "size_bytes": 40960, "created_at": "2026-10-02T10:00:00Z",
      "project": "/work/other", "model": "example-model", "provider": "openai" }
  ],
  "warnings": [{ "source": "codex-exec", "code": "missing_thread", "message": "stream has no thread.started event; omitted" }]
}
```

Listing rows use the same `$defs` session-metadata object as the session document, plus `size_bytes` and `file_modified_at`. `scope` is `{"project": "<absolute path>"}` or `{"all_projects": true}`. Each row carries `created_at` (recorded) or `file_modified_at` (the marked fallback that sets its sort position), and neither when no time is known. Rows follow CLI listing order (oldest first).

Warning codes: `malformed_line`, `oversized_line`, `thread_mismatch`, `item_decode_failed`, `root_unreadable`, `path_unreadable`, `missing_thread`, `identifier_conflict`, `duplicate_session`, `source_failed`. Existing per-source listing failures map to `source_failed` and existing Copilot `workspace.yaml` warnings to `path_unreadable`.

### Schema file

- `$schema` draft 2020-12; `$id` is the file's raw URL on upstream `main`.
- Top-level `oneOf` on `kind` (`session`, `listing`); `schema_version` is `const: 1`.
- `$defs` hold session metadata, entry, block (`oneOf` on `type`), warning and scope.
- No object sets `additionalProperties: false`, so version-1 consumers accept additive fields (1.3). `input`, `payload` and `counters` are open objects or arbitrary JSON by definition.

## Error Handling

| Condition | Behaviour |
|---|---|
| `-F` with `-f shared-json` | `follow mode does not support -f shared-json` |
| `-F` with `-a codex-exec`, an exec file, a `--source codex-exec` ID, or `latest` resolving to codex-exec | `follow mode does not support Codex exec streams`; files are probed before following; `latest` and IDs are checked after resolution |
| `--all-projects` with `-p`; `--source` misuse; mixed empty and nonempty `--codex-exec-dir` | usage error |
| configured root missing or unreadable; unreadable subdirectory or file; exec-looking file without thread; identifier conflict; duplicate `(source, id)` in all-projects scope | `ListWarning` with its code; other results kept (5.9) |
| exec malformed or oversized line | warning on stderr (existing `Warning: line N:` format), in `warnings[]`, and in the web transcript view |
| `-f json` on an exec stream | leading BOM stripped, then the existing JSONL path emits the event array |
| empty exec stream with `-f shared-json` | document with `entries: []`; Markdown/HTML keep `Session contains no entries` |
| codex-exec identifier that no longer matches a contained file | `session not found` |

## Risks and Assumptions

- Risk: walking large configured roots every 15 s may be slow even with zero body reads | Verify: benchmark `ListAll` on a synthetic tree of 10,000 files in the first discovery task | If wrong: reuse a directory's cached entry list while its own mtime is unchanged, still statting every known file so in-place edits are detected.
- Risk: Kiro CLI may commit through WAL without changing the stat key of `data.sqlite3`, `-wal` or `-journal` | Verify: test two writes through a WAL-mode connection between refreshes | If wrong: run the discover query on every refresh and record its measured page reads against 5.9.
- Risk: cold listings stay costly because Kiro IDE extraction reads whole `.chat` files | Verify: benchmark a cold multi-source listing with generated fixtures in the cache task | If wrong: stream-decode only `executionId`, `chat` length and `metadata`.
- Assumption: serde writes `web_search` items with two `id` keys and `encoding/json` keeps the last consistently across started and completed events, so lifecycle keying stays stable | Verify: fixture built from the pinned definition with both keys, in the parser task.

## Testing Strategy

Fixtures are synthetic. Exec fixtures record their provenance (pinned definition or sanitised structural observation) in `testdata/codex_exec/PROVENANCE.md`. Tests need no external CLI; implementation runs `make test` and `make lint`.

| Requirement | Tests |
|---|---|
| 1.1–1.4, 1.6 | Session documents for every source's existing fixtures and the exec fixtures, and listing documents (empty, warnings, omitted metadata), validated against the published schema and against a test copy that adds `additionalProperties: false` to apsis-owned structural objects only (Decision 18); golden JSON for one fixture per source; first-model summary with a mid-session Copilot model change; schema accepts future nested fields, unknown usage counters and a large integer, and rejects invalid types in known fields |
| 1.5 | Tool output longer than 3,000 runes survives byte-for-byte |
| 2.1 | Detection table: BOM, leading blanks, malformed lines, a malformed prefix longer than 64 KiB, missing or empty `thread_id`, other first events; existing-format detection results unchanged; forced parsing of a pre-thread failure |
| 2.2–2.4, 5.6, 5.7 | Fixtures per item type; multiple turns, interleaved tool links, `item.updated`, EOF without completion, unknown event, item and usage fields; two turns reusing `item_0` with different commands render each result beside its own command; Markdown, standalone HTML and fragment golden files; entries holding only new kinds are visible in both HTML paths |
| 2.5, 5.5, 5.8 | Temp trees: overlapping roots, root permutation, skipped names below a root and as a root, nested explicit root inside a skipped name, escaping file symlink, directory-symlink cycle, root symlink, case-alias roots where the volume is case-insensitive |
| 2.6, 2.7 | Worktree (`.git` file) versus parent repository; rollout and stream in different projects across both scopes; rollout whose filename UUID and recorded ID disagree; unreadable or malformed rollout with a matching filename |
| 2.8 | Property test (`rapid`): identifiers match `^[A-Za-z0-9_-]{1,256}$`, are deterministic, and are unchanged under root permutation, added copies and appends |
| 2.3 | Property test: random lifecycle sequences give one record per `(turn, id)` in first-appearance order with the last payload, incomplete exactly when uncompleted |
| 2.9 | Usage counters preserved, labelled, not summed across turns |
| 2.10, 4.3, 4.4 | Stdin has no time; listing time fields; property test that sort order is total, unknown times first, ties by source then identifier, and `latest` is the last element |
| 3.1 | Rollout with `model_provider` in Markdown, HTML and JSON, including provider without model; rollout without it unchanged against existing golden files; Read/Edit group output unchanged |
| 4.1, 4.2, 5.8 | `t.Setenv("HOME", …)` (plus XDG on Linux) with synthetic fixtures for every source, including a Kiro CLI test database with one conversation under two keys and Kiro IDE execution details; list through the CLI and `httptest`, then open each row from an unrelated working directory; identical identifiers across sources |
| 5.1 | Existing golden and CLI tests unchanged; with no roots configured, output and web pills are unchanged |
| 5.2, 5.3, 5.4 | Malformed middle line on all three surfaces; every follow combination including `latest`; flag and environment precedence, empty environment value, empty flag value, mixed values; raw JSON of a BOM-prefixed stream keeps `thread.started` |
| 5.9 | Counting opener injected into `Lister` plus discover-call counters: a second unchanged refresh reads 0 bytes; edits, additions, removals, companion-only changes (workspace, WAL, journal created or removed) and nested repository creation are reflected; identifiers of unaffected streams unchanged; the web handler's reversal leaves the next listing's order intact |

The schema validator, `github.com/santhosh-tekuri/jsonschema/v6`, is imported only from `_test.go` files, loads the schema from disk with network loading disabled, and stays out of the binaries (Decision 18, question U3).
