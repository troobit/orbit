# Design: Apsis — Shared JSON Output, Codex Exec Streams and Cross-Project Listing

Independent design draft by `claude-code-opus-5.5`, written from approved requirements revision 3, `decision_log.md` (Q1–Q28, Decisions 1–11) and the code on `7259a6c`. Proposed decisions are in [decisions.claude-code-opus-5.5.md](decisions.claude-code-opus-5.5.md) as P1–P7 and PQ1–PQ18; merge assigns final IDs.

## Overview

`transcript.Entry` stays the one shared representation every parser produces and every renderer consumes. This feature adds the content kinds exec streams need (plans, errors, generic events, usage, exit codes, incomplete state) and a provider. A new `internal/sharedjson` package projects that representation into a versioned, harness-neutral JSON layout. Exec streams become a sixth source, `codex-exec`, discovered only under configured roots. The existing `ListAll("")` path becomes reachable through `--all-projects`, with resolution fixed so every listed session opens from any directory (P1).

## Architecture

```
harness files ──parse──▶ transcript.ParseResult (Entry, Warnings, Metadata, Format)
                               │
            ┌──────────────────┼──────────────────────┐
     RenderMarkdown      RenderHTML(/Fragment)   sharedjson.Session(...)
                                                  sharedjson.Listing(...)  ◀── sessions.Lister.ListAll
```

### Placement

| Path | Contents |
|---|---|
| `internal/transcript/codex_exec_types.go`, `codex_exec_parser.go` | Exec event structs and parser |
| `internal/transcript/types.go`, `metadata.go`, `markdown.go`, `html.go` | New `Entry`/`ContentItem` fields, model label, rendering of new blocks |
| `internal/sharedjson/` (new) | Document types, `Session`, `Listing`, `SchemaVersion`; imports `transcript` and `sessions` |
| `schemas/apsis-shared-json.schema.json` (new, repo root) | Published JSON Schema (draft 2020-12), one file for both document kinds |
| `internal/sessions/codexexec.go` (new) | Root canonicalisation, walk, qualification, identifier, repository root |
| `internal/sessions/cache.go` (new) | Stat-keyed metadata cache |
| `internal/sessions/lister.go`, `resolver.go`, `types.go` | New source, listing fields, all-projects resolution, dedupe, ordering |
| `internal/sessions/parse.go` (new) | `ParseResolved`, shared by CLI and web |
| `internal/agents/kiro/logs/session.go` | `GetSessionByID` |
| `cmd/apsis/main.go` | Flags, input context, shared-json output, follow checks |
| `internal/apsisweb/` | Config, list/transcript views, source badge CSS |
| `internal/transcript/testdata/codex_exec/` | Synthetic exec fixtures plus `PROVENANCE.md` |

### Parity audit: new source and format

| Site | Needs equivalent | Note |
|---|---|---|
| `sessions/types.go` `allSources`, `displayNames`, `IsValidSource` | yes | `codex-exec`, display `codex-exec` |
| `Lister.ListAll` collectors | yes | `listCodexExec`, runs after `listCodex` (needs rollout IDs) |
| `sortSessionsByTimestamp` | yes | priority 5, then identifier tiebreak for all sources |
| `Resolver.Resolve`, `ResolvePath` | yes | `ResolvePath` returns the path; follow callers reject it |
| `Resolver.resolveFileCreatedAt` | yes | exec returns no recorded time, never `modTime` as creation |
| `cmd/apsis` `resolveInput` bare-ID source list | yes | `codex-exec` appended last, only when roots are configured |
| `cmd/apsis` `resolveFollowInput` file sources, `runLatest` `fileBackedSources` | no | exec is rejected with a named error ([Error Handling](#error-handling)) |
| `cmd/apsis` `agentToFormat`, `-a`/`-f` help | yes | `codex-exec` |
| `cmd/apsis` `convertToJSON` Kiro branch | no | exec is JSONL; the existing line path applies |
| `transcript` `DetectFormat`/`detectJSONLFormat`, `ParseJSONLWithFormat` | yes | |
| `transcript` `detectFormatFromLine`/`ParseJSONL` | no | used by follow and orbit's web view, neither accepts exec |
| `apsisweb` `ValidateSource` | yes | via `IsValidSource` |
| `apsisweb` `buildSessionListData` `Sources` | yes | `codex-exec` pill only when roots are configured |
| `apsisweb` `handleTranscript` parse branch | yes | replaced by `sessions.ParseResolved` |
| `apsisweb/static/style.css` badges, `CSSVersion` | yes | `.source-codex-exec` light/dark, bump version |
| `internal/web` (orbit), `internal/status` last-entry | no | Claude/Codex rollout only |

### Parity audit: new content kinds

| Site | Needs equivalent | Note |
|---|---|---|
| `markdown.go` `formatAssistantMessage` | yes | `plan`, `error`, `event`, `usage` |
| `html.go` `formatAssistantMessageHTML` (content check and render pass) | yes | both passes, else entries with only new kinds disappear |
| `formatToolInput`/`formatToolInputHTML` | yes | `command_execution`, `file_change` cases |
| `formatToolResult`/`formatToolResultHTML` | yes | exit code, incomplete marker |
| `formatUserMessage`/`formatUserMessageHTML` | no | exec entries are assistant entries |
| `grouping.go` `isReadOnlyEntry`/`isEditOnlyEntry` | no | their `default` branch already rejects unknown kinds |
| `BuildToolMeta`, `BuildSkillDescriptionMap` | no | unchanged keys |
| `last_entry.go` | no | orbit status only |

### Parity audit: provider

| Site | Needs equivalent | Note |
|---|---|---|
| `codex_parser.go` `processSessionMeta`, `ensureAssistantEntry` | yes | `model_provider` onto assistant entries |
| `markdown.go:425`, `html.go:583` assistant meta | yes | pass `modelLabel(entry.Model, entry.Provider)` |
| Other meta call sites (user, groups, slash commands) | no | already pass `""` |
| Copilot, Kiro CLI, Kiro IDE parsers | no | 5.1 limits output changes to Codex providers (PQ12) |

## Components and Interfaces

### 1. Shared representation additions (`transcript`)

```go
type Entry struct {
    // existing fields unchanged
    Provider string `json:"-"` // recorded provider; set only by the Codex rollout parser
    Turn     int    `json:"-"` // 1-based exec turn; 0 = no turn
}

type ContentItem struct {
    // existing fields unchanged; new fields all `json:"-"` so Claude unmarshalling is unaffected
    ItemID     string          // source item identifier (exec)
    Status     string          // source item status as recorded
    Incomplete bool            // started, never completed before EOF
    ExitCode   *int            // command exit code, nil when unrecorded
    Plan       []PlanItem      // Type "plan"
    EventType  string          // Type "event": source event or item type
    Payload    json.RawMessage // Type "event": parsed payload, compact
    Usage      *Usage          // Type "usage"
    // Type "error" uses Text for the message
}
type PlanItem struct { Text string; Completed bool }
type Usage struct {
    Basis    string                     // "thread_total_at_turn_completion" for Codex 0.166.0
    Counters []UsageCounter             // source key order; unknown keys kept
}
type UsageCounter struct { Name string; Value json.RawMessage }

type ParseResult struct {   // existing fields plus:
    Format Format            // format actually parsed; FormatUnknown for the Claude fallback
}
type ParseResultMetadata struct { // existing fields plus:
    StartedAt string // recorded session start: Codex session_meta.timestamp, Kiro IDE metadata.startTime
    Project   string // recorded project: Codex session_meta.cwd
    ThreadID  string // exec thread_id
}
type ParseWarning struct { Line int; Message string; Unit string } // Unit "" = physical line, "chat-message" = Kiro IDE chat index
```

`FormatCodexExec` joins the `Format` enum. `ParseResult.Metadata` may now be non-nil with a nil `TotalCost`; renderers already guard on `TotalCost != nil`.

### 2. Codex exec stream parser

`ParseCodexExec(r io.Reader) (*ParseResult, error)`; it returns an error only on reader failure. An empty or event-less stream yields zero entries.

**Detection** (`detectJSONLFormat`): the first non-empty line that parses as a JSON object decides. If its `type` is `thread.started` and `thread_id` is a nonempty string, the format is `FormatCodexExec`; otherwise detection continues with the existing rules. BOM stripping already happens in `DetectFormat`. `-a codex-exec` forces the parser.

**Lifecycle** (P5): the parser keeps an ordered list of turns. Turn 0 holds records before the first `turn.started`; each `turn.started` opens turn *n*+1. Within a turn, records are keyed and kept in first-appearance order:

| Event | Key | Effect |
|---|---|---|
| `thread.started` | — | first nonempty `thread_id` becomes `Metadata.ThreadID`; a later one with the same ID is dropped; a different ID becomes an `event` record (PQ14) |
| `item.started` / `item.updated` / `item.completed` | `(turn, item.id)`; a missing ID keys by line number | first sighting appends; later events replace the stored item JSON; `completed` sets done |
| `turn.completed` | own record | `usage` block, basis `thread_total_at_turn_completion` |
| `turn.failed`, top-level `error` | own record | `error` block with `message`; the original JSON is kept as payload |
| any other event type | own record | `event` block |

At EOF each turn becomes one assistant `Entry` (`Turn` set; no timestamp, model or provider). Its content is the translation of each record's final state. Items not completed get `Incomplete: true`, including on their tool result. Lifecycle events never render separately.

**Item translation** (shapes from pinned `exec_events.rs` 0.166.0):

| Item `type` | Content |
|---|---|
| `agent_message` | `text` |
| `reasoning` | `thinking` |
| `command_execution` | `tool_use` (name `command_execution`, input `{command}`) + `tool_result` (content `aggregated_output`, `ExitCode`, `Status`, `IsError` when status is `failed`/`declined` or exit code ≠ 0) |
| `file_change` | `tool_use` (input `{changes}`) + `tool_result` (content: one `kind path` line per change, `Status`, `IsError` when `failed`) |
| `mcp_tool_call` | `tool_use` (input `{server, tool, arguments}`) + `tool_result` (content: compact JSON of `result`, or `error.message`; `IsError` when `failed` or `error` set) |
| `web_search` | `tool_use` (input `{query, action}`) + `tool_result` (content: compact JSON of `results` when present) |
| `todo_list` | `plan` |
| `error` | `error` |
| anything else, including `collab_tool_call` (not in 2.2) | `event` with `EventType` = item type, payload = final item JSON |

Every exec tool item always emits its `tool_result`, because the renderers draw non-Task tools only when their result arrives. Tool IDs are the item IDs. A known item type whose payload fails typed decoding (wrong JSON types) becomes an `event` block with a warning, so unverified shapes fall back to generic handling (5.6).

**Warnings** (5.2): a malformed line yields `ParseWarning{Line: physical 1-based line, Message: "malformed JSON event"}` plus, for `*json.SyntaxError`, the byte offset. The message never contains source text. Parsing continues at the next line.

### 3. Shared JSON projection (`internal/sharedjson`)

```go
const SchemaVersion = 1
type SessionContext struct {
    Source   string                     // from resolver; else from ParseResult.Format
    ID       string                     // listing identifier when known
    Path     string                     // canonical input path; "" for stdin
    Resolved *sessions.SessionMetadata  // nil for file/stdin input
}
func Session(res *transcript.ParseResult, ctx SessionContext) SessionDocument
func Listing(list []sessions.SessionInfo, warnings []sessions.ListWarning, scope Scope) ListingDocument
func Write(w io.Writer, doc any) error // two-space indent, trailing newline, no HTML escaping
```

- The projection reads parsed data only and never calls renderer helpers, so content is never shortened (1.5).
- Field derivation is in [Data Models](#data-models). Document-level `model`/`provider` are the first nonempty entry values in transcript order (1.6). Entries keep their own values.
- `Write` uses `json.Encoder` with `SetEscapeHTML(false)` so text keeps `<`, `>` and `&`.

### 4. Listing, scope and cache (`sessions`)

```go
type SessionInfo struct {
    ID, Source string; CreatedAt time.Time; Size int64   // existing; CreatedAt stays the sort time
    TimeFromFilesystem bool    // CreatedAt is file modification time, not a recorded time
    Project, ProjectBasis string // basis "recorded" | "containing-repository"
    Model, Provider string       // when the listing read already yields them
    ThreadID string              // codex-exec only
}
type SessionMetadata struct { /* existing */ TimeFromFilesystem bool; Project, ProjectBasis, ThreadID, Path string }

type Option func(*Lister)
func WithCodexExecRoots(roots []string) Option
func NewLister(opts ...Option) (*Lister, error)
func NewResolver(projectPath string, opts ...Option) (*Resolver, error) // "" = all projects
func (l *Lister) Sources() []string // AllSources minus codex-exec when no roots are configured
```

`ListAll("")` remains the all-projects listing. Each listing pass:

1. Holds `l.mu` for the whole pass, so serve's page load and HTMX poll never interleave cache updates.
2. Enumerates directories and stats candidates as today, without opening transcript bodies.
3. For each candidate, looks up `cache[path]`. The key is `(size, mtime, ctime, inode)`, with ctime and inode where the platform provides them (extend `getFileInfo`). On a hit it reuses the extracted metadata. On a miss it re-reads the bounded head: at most 50 lines and 1 MiB, the existing `codexMetaScanLimit` convention.
4. Sweeps cache entries not seen in the pass (P3).

| Source | Cached head metadata | Project (basis) | Time |
|---|---|---|---|
| claude | first timestamp, first `cwd`, first assistant `model` within the head | first `cwd` (recorded) | first timestamp, else mtime (flagged) |
| codex | `session_meta` timestamp, `cwd`, `model`, `model_provider` (one scan replaces the two helpers) | `cwd` (recorded) | as today, flagged on fallback |
| copilot | `workspace.yaml` (keyed by its own stat) | `git_root`, else `cwd` (recorded) | as today, flagged on fallback |
| kiro-cli | discover result, keyed by stat of `data.sqlite3` and `data.sqlite3-wal` | DB `key` (recorded) | DB `created_at` |
| kiro-ide | `.chat` header: execution ID, chat count, `startTime`, `modelId` | omitted (workspace hash is one-way) | `startTime`, else mtime (flagged) |
| codex-exec | head qualification and `thread_id` | containing repository | always mtime (flagged) |

Listing model/provider come only from what the head read yields. A session whose model first appears later lists without one; the session document is authoritative (PQ10).

**Ordering and uniqueness** (4.4): sort by `CreatedAt` ascending. Zero times sort first, ties break by source priority (claude, copilot, codex, kiro-cli, kiro-ide, codex-exec) then identifier. Within one listing a `(source, id)` pair appears once. A duplicate keeps the candidate the resolver would open and adds a warning (P6). The same rule applies to Claude across project directories (first in sorted directory order), Codex rollouts (first in walk order), Kiro IDE across workspaces (the shared best-candidate rule) and Kiro CLI (most recent `updated_at`, as `DiscoverAll` already does).

### 5. Exec discovery, identity, project and suppression

**Roots** (5.5): each configured root is made absolute and passed through `filepath.EvalSymlinks`. A root that fails yields a `ListWarning` and is skipped. `filepath.WalkDir` walks each canonical root and never descends directory symlinks. Subdirectories named `.git`, `node_modules` or `testdata` are skipped; a root is scanned whatever its name. A `.jsonl` regular file is a candidate at its walked path. A `.jsonl` symlink is resolved, and kept only when the target is a regular file inside some canonical root. Candidates are deduplicated by canonical path string. Unreadable directories and files yield warnings and the walk continues.

**Qualification** (2.1, 2.11): the cached head read finds the first parseable JSON object line.

- `thread.started` with a nonempty string `thread_id`: listed.
- Another exec event type (`thread.started` without a valid ID, `turn.*`, `item.*`, `error`): omitted with a `ListWarning`.
- Anything else: not an exec stream, silently ignored (PQ13).

**Identifier** (2.8, P2): `sanitise(thread_id)[:239] + "-" + hex(sha256(canonical path))[:16]`. `sanitise` maps every byte outside `[A-Za-z0-9_-]` to `_`. The identifier depends only on the stream's own path and recorded thread ID, so appends, copies elsewhere, root overlap and root order cannot change it. If two canonical paths hash to the same identifier, the lexically later path is omitted with a warning. `ThreadID` keeps the original value.

**Project** (2.6): walk up from the stream's directory to the nearest ancestor containing a `.git` entry (directory for repositories, file for worktrees). Answers are memoised per directory for the pass. Project scope keeps a stream only when `normalizePath(root) == normalizePath(project)`. A stream outside any repository lists only in all-projects scope.

**Suppression** (2.7): `listCodex` already walks every rollout file. It records the UUID found in each filename by `uuidPattern` in every scope, which is the set `findCodexSession` can resolve. A stream whose `thread_id` matches an entry in this set is dropped before project filtering, in every scope. If listing rollouts fails, the set is incomplete and streams are kept; the rollout warning is already reported.

### 6. Resolution in all-projects scope

`Resolver` with `projectPath == ""`:

| Source | Behaviour |
|---|---|
| claude | existing all-projects search, sorted directory order |
| codex, copilot | unchanged (already global) |
| kiro-cli | new `logs.(*DB).GetSessionByID(ctx, id)`: `SELECT key, value … WHERE conversation_id = ? ORDER BY updated_at DESC LIMIT 1`, the row `DiscoverAll` keeps; returns `Project = key` |
| kiro-ide | scans every workspace under `KiroIDEBasePath` with the same `bestKiroIDECandidate` function the lister uses (most chat entries, newest mtime, lexically smallest path); `CostPath` derives from the winning workspace |
| codex-exec | both scopes: parse the 16-hex suffix, hash each candidate path from a metadata-only walk of the roots, read only the matching file's head, require its sanitised thread prefix to match, recheck root containment on the resolved path, then open; suppression does not block opening |

Without these branches, `logs.GetSession` and `KiroIDEWorkspaceDir` would turn `""` into the current directory through `filepath.Abs`. The resolver also returns `Path`, `Project`, `ProjectBasis`, `ThreadID` and `TimeFromFilesystem` in `SessionMetadata`. For codex-exec, `CreatedAt` holds the mtime with the flag set.

`sessions.ParseResolved(rs *ResolvedSession) (*transcript.ParseResult, error)` replaces the duplicated branch in `cmd/apsis` and `apsisweb`. It forces `FormatKiroIDE` with the cost path, forces `FormatCodexExec` for codex-exec, and auto-detects otherwise.

### 7. CLI surface (`apsis`)

| Option | Behaviour |
|---|---|
| `-f shared-json` | session document for an ID, file, stdin or `latest`; listing document with `--list` |
| `--all-projects` | lists, `latest` and ID lookups use project `""`; error when combined with `-p` |
| `-s`, `--source <source>` | the positional argument is an identifier of that source; no file-path detection; error without an argument |
| `--codex-exec-dir <dir>` (repeatable) | exec roots; when given, `APSIS_CODEX_EXEC_DIRS` is ignored |
| `APSIS_CODEX_EXEC_DIRS` | `filepath.ListSeparator`-separated (`:` on Unix, `;` on Windows); unset or empty means no roots |
| `-a codex-exec` | forces the exec parser |

The same `--all-projects` and `--codex-exec-dir` options, and the environment variable, apply to `apsis serve` (5.4). Relative roots resolve against the working directory.

`resolveInput` returns an input context: reader, source, ID, canonical path, cost path and resolver metadata. `convert` builds `SessionContext` from it. For exec file input the Markdown/HTML header shows `Thread ID` from content, plus `Session ID` (the derived identifier) when a path exists; it never uses the filename stem (2.11). Listing text output is unchanged except:

- In `--all-projects` scope a fifth tab-separated column holds the project, or `-`.
- codex-exec rows print the time followed by ` (file modified)`.
- `apsis latest` on a codex-exec session adds the same suffix to its `Using … from …` stderr line.

### 8. `apsis serve`

`apsisweb.Config` gains `AllProjects bool` and `CodexExecRoots []string`. A single long-lived `Lister` carries the cache across refreshes. `SessionView` gains `Project` (all-projects scope only) and `FileTime bool`, which renders `file modified` beside codex-exec times. The `sessions_list` fragment renders the warning banner so warnings follow each refresh. The page includes the fragment, so the banner is not duplicated. `TranscriptViewData` gains `Warnings`. For codex-exec sessions, the transcript view shows the parse warnings with the existing `.warning-banner` style, and shows thread ID and modification time in `session-meta` (PQ16).

### 9. Rendering of new content

The existing pattern is the baseline; only the differences are listed.

| Content | Markdown | HTML (standalone and fragment share `renderEntriesToBuilder`) |
|---|---|---|
| assistant meta | `modelLabel`: `model (provider)`, provider alone `(provider)`; exec turns append `turn N` as another ` · ` part | same parts via `FormatMessageMetaHTML` |
| `command_execution` input | same block as `Bash` | same as `Bash` in `formatToolInputHTML` |
| `file_change` input | `- kind path` list | same list markup |
| exit code | `**Result (exit code N):**` | same label |
| incomplete item | ⏳ icon and `(incomplete)` in the summary instead of ✅/❌ | same |
| `plan` | checkbox list matching the `TodoWrite` rendering (`[x]`/`[ ]`) | `TodoWrite` markup |
| `error` | `❌ **Error:** message` | failed-tool styling |
| `event` | collapsed `<details>` `📦 Event: type` with a JSON block | collapsed block like non-Task tools |
| `usage` | `**Token usage (thread totals at turn completion):** name value · …` | same text in a `usage` div; one CSS rule |

## Data Models

### Session document

```json
{
  "schema_version": 1,
  "kind": "session",
  "source": "codex-exec",
  "id": "0199aaaa-0000-7000-8000-000000000001-3f2a9c0d1b7e4a56",
  "thread_id": "0199aaaa-0000-7000-8000-000000000001",
  "project": "/work/example-repo",
  "project_basis": "containing-repository",
  "warnings": [{ "message": "malformed JSON event (offset 14)", "line": 4 }],
  "entries": [
    {
      "role": "assistant",
      "turn": 1,
      "content": [
        { "type": "tool_use", "id": "item_1", "name": "command_execution", "input": { "command": "make test" }, "item_id": "item_1" },
        { "type": "tool_result", "tool_use_id": "item_1", "content": "ok\n", "is_error": false, "exit_code": 0, "status": "completed", "item_id": "item_1" },
        { "type": "plan", "items": [{ "text": "Run tests", "completed": true }], "item_id": "item_2", "incomplete": true },
        { "type": "usage", "basis": "thread_total_at_turn_completion", "counters": { "input_tokens": 1200, "cached_input_tokens": 800, "output_tokens": 90 } }
      ]
    }
  ]
}
```

| Field | Derivation | Omitted when |
|---|---|---|
| `source` | resolver source, else `ParseResult.Format` | Claude fallback for unrecognised input |
| `id` | listing identifier; for file input: existing filename stem, or derived from path and thread for exec | stdin; exec without thread |
| `thread_id` | `Metadata.ThreadID` | not exec |
| `created_at` | `Metadata.StartedAt`, else first parseable entry timestamp, else resolver time if not flagged as filesystem | none recorded (all exec) |
| `project`, `project_basis` | `Metadata.Project` or first entry `cwd` (recorded); exec: containing repository of `Path`; else resolver `Project` | unknown |
| `model`, `provider` | first nonempty in entry order | none recorded |
| `cost` | `{amount, unit}` from `TotalCost`/`CostUnit` | `TotalCost` nil |
| `warnings[]` | `message` plus `line` or `chat_message` from `Unit` | position 0 |
| entry fields | `role`, `timestamp`, `model`, `provider`, `turn`, `cwd`, `source_session_id`, `uuid`, `parent_uuid`, `meta`, `source_tool_use_id`, `tool_use_result` (`file_path`, `structured_patch`) | empty |
| block common | `item_id`, `status`, `incomplete` | empty/false |
| block types | `text`, `thinking` (`text`), `tool_use` (`id`, `name`, `input`), `tool_result` (`tool_use_id`, `content`, `is_error`, `exit_code`), `plan`, `error` (`message`), `event` (`event_type`, `payload`), `usage` (`basis`, `counters` map of raw JSON values), `other` (`source_type`, for parsed content kinds no renderer handles) | — |

### Listing document

```json
{
  "schema_version": 1,
  "kind": "listing",
  "scope": { "all_projects": true },
  "sessions": [
    { "source": "codex-exec", "id": "0199aaaa-…-3f2a9c0d1b7e4a56", "thread_id": "0199aaaa-…", "size": 5120,
      "file_modified_at": "2026-10-01T09:00:00Z", "project": "/work/example-repo", "project_basis": "containing-repository" },
    { "source": "codex", "id": "0199bbbb-…", "size": 40960, "created_at": "2026-10-02T10:00:00Z",
      "project": "/work/other", "project_basis": "recorded", "model": "example-model", "provider": "openai" }
  ],
  "warnings": [{ "source": "codex-exec", "message": "stream has no thread.started event; omitted" }]
}
```

`scope` is `{"project": "<abs path>"}` or `{"all_projects": true}`. Each listing session carries exactly one of `created_at` (recorded) or `file_modified_at` (the marked fallback that sets its sort position), unless no time is known. Order is the CLI listing order (oldest first).

### Schema file

- `$schema` draft 2020-12; `$id` is the file's raw URL on upstream `main`.
- Top-level `oneOf` on `kind` (`session`, `listing`); `schema_version` is `const: 1`.
- `$defs` hold entry, block (`oneOf` on `type`), warning and listing session.
- Objects leave `additionalProperties` unset, so consumers on version 1 accept additive fields (1.3).
- Tests compile a closed copy with `additionalProperties: false` added to every object schema, so any field the code emits but the schema does not declare fails CI (P4).

## Error Handling

| Condition | Behaviour |
|---|---|
| `-F` with `-f shared-json` | `follow mode does not support -f shared-json` |
| `-F` with `-a codex-exec`, an exec file, a `--source codex-exec` ID or `latest` resolving to codex-exec | `follow mode does not support Codex exec streams`; files are sniffed with `DetectFormat` before following |
| `--all-projects` with `-p`; `--source` without an argument or with an unknown source | usage error |
| configured root missing or unreadable; unreadable subdirectory or file; exec-looking file without thread; identifier collision; duplicate `(source, id)` | `ListWarning`; other results kept (5.9) |
| exec malformed line | warning on stderr (existing `Warning: line N:` format), in `warnings[]`, and in the web transcript view |
| `-f json` on an exec stream | existing JSONL path emits the event array |
| empty exec stream with `-f shared-json` | document with `entries: []`; Markdown/HTML keep `Session contains no entries` |

## Risks and Assumptions

- Risk: walking large configured roots every 15 s in serve may be slow even with zero body reads | Verify: benchmark `ListAll` on a synthetic tree of 10,000 files in the first discovery task | If wrong: skip rescanning a directory whose own mtime is unchanged and reuse its cached children.
- Risk: Kiro CLI may commit through WAL without changing the stat key of `data.sqlite3` or `data.sqlite3-wal` | Verify: test writing twice through a WAL-mode connection between refreshes | If wrong: run the metadata-only discover query on every refresh. It reads no `value` overflow pages, but inline pages are still read.
- Assumption: serde serialises `web_search` items with two `id` keys and `encoding/json` keeps the last consistently across started/completed events, so lifecycle keying stays stable | Verify: fixture built from the pinned definition with both keys, in the parser task.

## Testing Strategy

| Requirement | Tests |
|---|---|
| 1.1–1.4, 1.6 | Session documents for every source's existing fixtures and new exec fixtures, plus listing documents (empty, warnings, omitted metadata), validated against the closed schema with `santhosh-tekuri/jsonschema/v6` (P4); golden JSON for one fixture per source; first-model/provider summary with a mid-session Copilot model change |
| 1.5 | Tool output longer than 3,000 runes survives byte-for-byte |
| 2.1 | Detection table: BOM, leading blanks and malformed lines, missing or empty `thread_id`, other first events; forced parsing of a pre-thread failure |
| 2.2–2.4, 5.6, 5.7 | Fixtures per item type (provenance listed in `testdata/codex_exec/PROVENANCE.md`); multiple turns, interleaved tool links, `item.updated`, EOF without completion, unknown event/item/usage fields; Markdown, standalone HTML and fragment golden files |
| 2.5, 5.5, 5.8 | Temp trees: overlapping roots, skipped names below and as root, escaping file symlink, directory-symlink cycle, root symlink |
| 2.6, 2.7 | Worktree (`.git` file) versus parent repository; rollout and stream in different projects across both scopes |
| 2.8 | Property test (`rapid`): identifiers match `^[A-Za-z0-9_-]{1,256}$`, are deterministic, and are unchanged under root permutation, added copies and appends |
| 2.3 | Property test: random lifecycle event sequences give one record per `(turn, id)` in first-appearance order with the last payload, incomplete exactly when uncompleted |
| 2.9 | Usage counters preserved, labelled, not summed across turns |
| 2.10, 4.3, 4.4 | Stdin has no time; listing time fields; property test that sort order is total, unknown times first, ties by source then identifier, and `latest` is the last element |
| 3.1 | Rollout with `model_provider` in Markdown/HTML/JSON; rollout without it unchanged against existing golden files |
| 4.1, 4.2, 5.8 | `t.Setenv("HOME", …)` (plus XDG on Linux) with synthetic fixtures for every source, including a Kiro CLI test DB and Kiro IDE execution details; list through CLI and `httptest`, then open each row from an unrelated working directory; identical IDs across sources |
| 5.1 | Existing golden and CLI tests unchanged; with no roots configured, output and web pills are unchanged |
| 5.2, 5.3, 5.4 | Malformed middle line on all three surfaces; every follow combination; flag/env precedence and empty env |
| 5.9 | Counting `open` hook injected into `Lister` (plus discover-call counters): second unchanged refresh reads 0 bytes; edit, add and remove are reflected; identifiers of unaffected streams unchanged |

The schema library is test-only: it is imported only from `_test.go` files and stays out of the binaries.
