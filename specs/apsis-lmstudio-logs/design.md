# Design: Apsis — LM Studio Server Logs

## Overview

LM Studio's disk server log is a physical-line, header-delimited text grammar, not JSON or JSONL. This adds a fifth parsed source (`lmstudio`) and a fifth shared-representation format (`FormatLMStudio`) to the `transcript`/`sessions` pipeline the `apsis-shared-json` feature established for `codex-exec`, reusing that pipeline's canonicalisation, root-confinement, all-projects resolution and shared-JSON projection wherever the grammar allows, and introducing new mechanics only where the source genuinely differs: text-grammar lexing, thread identity that spans many files instead of one, a request/message split instead of one entry per source record, and unattributed log-level events. Requirements are in [requirements.md](requirements.md); decisions are in [decision_log.md](decision_log.md).

## Architecture

### Placement

| Path | Contents |
|---|---|
| `internal/transcript/lmstudio_types.go` (new) | Record/item structs mirroring the pinned grammar; `FormatLMStudio` |
| `internal/transcript/lmstudio_parser.go` (new) | `ParseLMStudioRecords`, `ParseLMStudio`, `ProbeLMStudio`, request/message translation, extension-prefix comparison |
| `internal/transcript/lmstudio_timezone.go` (new) | `resolveZone(tzEnv string) (zone, warning)`; `time/tzdata` blank import |
| `internal/transcript/types.go` | New `Entry`/`ContentItem` fields (below); `FormatLMStudio` |
| `internal/transcript/parser.go` | LM Studio probe in `DetectFormat`, ahead of the exec probe (see [Detection](#detection)); new warning codes |
| `internal/transcript/markdown.go`, `html.go`, `metadata.go` | Request-boundary headers, unattributed-event blocks, parameters block, one-time attribution notice |
| `internal/sessions/lmstudio.go` (new) | Root resolution, `YYYY-MM/YYYY-MM-DD.N.log` discovery, per-file summary cache, thread grouping, identifiers, listing, resolution |
| `internal/sessions/lmstudio_cache_unix.go`, `lmstudio_cache_other.go` (new, build-tagged) | Per-file change fingerprint: real on `unix`, a no-op (always-miss) stub elsewhere (see [Cache](#5-discovery-identity-and-cache)) |
| `internal/sessions/types.go`, `lister.go`, `resolver.go`, `parse.go` | New source, options, listing/resolve branches, ordering priority |
| `internal/sharedjson/` | Session/listing projection for the new entry shapes and listing fields |
| `cmd/apsis/main.go` | `-a lmstudio`, `--lmstudio-log-dir`, `APSIS_LMSTUDIO_LOG_DIR`, follow/raw-JSON rejection |
| `internal/apsisweb/` | Config, discovery-disclosure warning text, source badge CSS |
| `internal/transcript/testdata/lmstudio/` (new) | Synthetic fixtures plus `PROVENANCE.md` (requirement 8.4) |

### Parity audit: new source and format

This follows the same shape `codex-exec` added in `apsis-shared-json`; the table below is this feature's equivalent audit.

| Site | Needs equivalent | Note |
|---|---|---|
| `sessions/types.go` `allSources`, `displayNames`, `IsValidSource`, `sourcePriority` | yes | `lmstudio`, priority after `codex-exec` |
| `Lister.ListAll` collectors, `Lister.Sources()` | yes | `listLMStudio` gated on `LMStudioLogDir != ""`; always empty in project scope regardless of that gate (7.1) |
| `Resolver.Resolve`, `ResolvePath` | yes | `ResolvePath` returns an error: no single file backs a logical session (see [Components 6](#6-session-assembly-and-resolution)) |
| `cmd/apsis` `resolveInput` bare-ID source list, `resolveFollowInput` | partial | ID lookup joins the list; follow is rejected the same way exec streams are (8.1) |
| `cmd/apsis` `agentToFormat`, `-a`/`-f` help | yes | `lmstudio` |
| `cmd/apsis` `convertToJSON` | no | raw `-f json` is rejected for this source (8.1), so the JSONL/`-f json` path is never reached |
| `transcript` `DetectFormat` | yes | new probe, ordered before the exec probe (see [Detection](#detection)) |
| `apsisweb` `ValidateSource`, `buildSessionListData` `Sources` | yes | via `IsValidSource`/`Sources()` |
| `apsisweb` `handleTranscript` | no | already dispatches through `sessions.ParseResolved` |
| `apsisweb/static/style.css` badges, `CSSVersion` | yes | `.source-lmstudio` light and dark; bump version |
| `internal/web` (orbit), `internal/status` | no | Claude and Codex rollouts only |

### Record grammar and lexing

A **header** is a physical line matching `^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]\[(\w+)\](\[[^\]]*\])*\s*(.*)$` (timestamp, level, zero or more bracketed tags, message). A **record** is a header plus every following line up to the next header or EOF. Classification resolves the reported path first, independently of level, then applies level only to the two outcomes that depend on it:

| # | Condition | Record kind |
|---|---|---|
| 1 | `Received request: POST to <path>`, `<path>` ∈ [2.1](requirements.md#2.1) | included request, translated — any level |
| 2 | `Generated prediction:` / `Generated synchronous v1/responses response:` | logged response — any level |
| 3 | reports a POST path, not matched by row 1, not a documented non-inference path (embeddings, model operations, `/v1/responses/input_tokens`, `/v1/messages/count_tokens`) | unsupported-endpoint warning (4.6) — any level; never falls through to rows 4/5 |
| 4 | no row above matches, level WARN or ERROR | retained event, request attribution unknown (4.1–4.3) — this is what keeps a WARN/ERROR excluded-endpoint diagnostic (startup, model-management, or a documented non-inference path) visible, rather than silently dropped |
| 5 | no row above matches, any other level | routine; ignored (4.4) |

Row 3 is unconditional on level, so an unsupported path is never also silently ignored. A documented non-inference path (row 3's exclusion list) that isn't matched by row 1 falls to row 4 or 5 purely on level — retained when WARN/ERROR, ignored otherwise — which is the exact precedence 4.4/Q10 require: WARN/ERROR retention beats routine-category suppression, including excluded-endpoint diagnostics.

A request record's trailing ` with body` introduces one JSON value spanning the record's remaining lines (pretty-printed, so no single physical line is itself complete JSON — see [Detection](#detection) for why this matters). `ParseLMStudioRecords` decodes it with `json.Decoder` positioned at that offset; a decode that fails only because the value is unterminated at EOF is `incomplete` (6.1), a decode that fails for any other reason is `malformed` (2.5), and a value whose byte length exceeds the documented parser limit before decoding starts is `oversized` (6.4). A request with no ` with body` suffix is `body not logged` (2.5).

### Detection

`ProbeExecStream` (the existing Codex exec probe) reads whole lines looking for the first line that parses as a standalone JSON object. No LM Studio physical line is ever standalone JSON (headers are plain text; body lines are fragments of a pretty-printed value), so unmodified it would scan an LM Studio input to EOF before returning `other` — reading an entire large log on every detection attempt. `DetectFormat` therefore runs a bounded LM Studio probe over the same 64 KiB chunk Kiro's validated check uses, before calling the exec probe: `ProbeLMStudio(chunk) bool` reports whether the chunk contains a header whose message matches `Received request: POST to` followed by one of the [2.1](requirements.md#2.1) paths exactly. A match returns `FormatLMStudio` immediately (1.1, Q13); a miss falls through to the existing exec-probe-then-JSONL-fallback chain unchanged. A chunk with headers but no included-request match produces no detected format from this probe; `Parse`'s caller-facing detection error names `-a lmstudio` as well as the other forced formats when timestamp-and-level headers were present but unmatched (1.1).

### Logical session model

A **logical session** is every request sharing one nonempty, untruncated `client_metadata.thread_id`, or one unidentified request on its own (3.1, 3.2). Unlike `codex-exec`'s one-file-one-stream identity, a threaded session's requests can be spread across every file under the configured root, added, rotated or removed independently of each other — so "which files make up session X" is not knowable from the session ID alone; it has to come from an index built by reading request bodies. Two read paths follow from this:

- **Listing** needs, per thread, only a count, first-known model/timestamp and which files hold it — never full message bodies. [Components 5](#5-discovery-identity-and-cache) caches exactly that summary per file, keyed on a change fingerprint, so an unchanged refresh re-reads no file (7.3).
- **Opening** a listed session needs full bodies, but only from the files the cached summary says contain that thread — never every file under the root. [Components 6](#6-session-assembly-and-resolution) re-parses just those.

Direct file/stdin conversion is never filtered: it renders one source-ordered log transcript containing every record the input holds, with each request's session membership as an annotation rather than a structural split (Decision 5), exactly as `codex-exec`'s single-stream document already does for interleaved turns.

## Components and Interfaces

### 1. Shared representation additions (`transcript`)

```go
type Entry struct {
    // existing fields unchanged
    RequestID      string `json:"-"` // the owning LMStudioRequest's ID (Components 5); empty on an unattributed entry
    SuppliedContext bool  `json:"-"` // an assistant message resent as request input, not a new completion (2.4)
    Unattributed   bool   `json:"-"` // a logged response/WARN/ERROR event; request attribution unknown (4.1)
    SourceLine     int    `json:"-"` // unattributed entries only: physical start line, for event-scoped tool IDs (see below) and 4.2's locations
}

// LMStudioRequest is one request record's boundary metadata: endpoint, thread
// identity, parameters, body state and the extension/discontinuity result
// against its thread predecessor. It is parallel to Entries, not embedded in
// them, so this data is written once per request rather than duplicated on
// every message item it produced (Decision log Q18 revision).
type LMStudioRequest struct {
    ID                   string          // Components 5; "" until the sessions layer assigns it
    StartLine, EndLine   int             // physical line range of the record (6.2)
    Endpoint             string
    ThreadID             string          // client_metadata.thread_id, exactly as recorded; empty = singleton
    ParentThreadID       string          // client_metadata parent-thread identifier, when recorded
    RequestKind          string          // client_metadata recorded role (dialogue turn, compaction, memory, side request)
    Model                string
    Timestamp            string          // recorded wall time, as logged
    ResolvedTime         time.Time       // zero when not uniquely interpretable (5.2)
    BodyState            string          // "", "not_logged", "malformed", "incomplete", "oversized" (2.5, 6.1, 6.4)
    Parameters           json.RawMessage // remaining top-level fields: tools, stream/store, sampling, cache keys, integration headers, previous_response_id/server-held-context references, client_metadata (2.7, 2.9)
    ContextDiscontinuity bool            // not an extension of the thread predecessor's logged input (3.6)
    PriorPrefixCount     int             // leading items identical to the predecessor's; 0 = none (3.5)
}

type ParseResult struct {
    // existing fields unchanged
    LMStudioRequests []LMStudioRequest // parallel to Entries, in source order; empty for every other format
}
```

A request with no translatable content (every `BodyState` other than `""`) still gets exactly one `LMStudioRequest` and zero `Entry` values; renderers walk `LMStudioRequests` to place request-boundary headers in source order and fill in that request's `Entry` values (matched by `RequestID`) underneath — a request with no entries still renders its boundary header and `BodyState` (2.5), with no separate placeholder entry needed.

Logged responses, WARN and ERROR records become entries with `Unattributed: true`, `RequestID: ""`, and content under the existing `ContentItem` kinds (`text`, `tool_use`, `tool_result`, `error`, `event`, `usage` — reusing the `codex-exec` kinds rather than inventing parallel ones). `Entry.Model`/`Provider` are set from the recorded request model and the constant `"lmstudio"` on request entries. An unattributed entry carries its own recorded model tag when the record has one, displayed as-is; per 4.1 that tag is informational only and never used to infer which request produced the entry.

`ParseResultMetadata.ThreadID` is unused for this format (a direct-view document can hold many threads); per-thread/request identity lives on `LMStudioRequest` instead. *(Superseded for session views by Q42: a session view, which holds one thread or one request, sets it to its first request's thread ID; direct views still leave it empty.)*

Implementation added four fields the struct above omits (Q23–Q25): `Entry.WallTime` (a log-level entry's logged wall time, since it has no `LMStudioRequest`), `Entry.ItemDigest` (the extension-prefix comparison key; one entry per logged item), `ContentItem.Truncated` (set wherever `<Truncated in logs>` appears) and `ParseResultMetadata.TimeZone` (the assumed zone's label). WARN/ERROR records are `Type: "log"` entries holding one `text` (WARN) or `error` (ERROR) item whose `Status` is the logged level; unsupported POST paths are `log` entries with a body-free `unsupported_endpoint` event; logged responses are `assistant` entries. All carry `SourceLine`.

### 2. LM Studio record parser (`transcript`)

```go
func ProbeLMStudio(chunk []byte) bool
func ParseLMStudioRecords(r io.Reader, zone zoneResult) (requests []LMStudioRequest, entries []Entry, warnings []ParseWarning, err error)
func ParseLMStudio(r io.Reader) (*ParseResult, error) // direct-view document for one file/stdin; calls ParseLMStudioRecords, then the shared assembly step below
```

Neither function takes a path: this parser only knows the bytes it is given, never the filesystem. `LMStudioRequest.ID` and `Entry.RequestID` hold a **temporary key** — `strconv.Itoa(StartLine)` — until [Components 5](#5-discovery-identity-and-cache) assigns the final ID afterward, in `sessions`, from the path context only that package has (the same split `codex-exec` already uses: `ParseCodexExec` never computes `ExecIdentifier`; `sessions` does). `StartLine` is unique within one parse by construction, so it is a safe join key between the two returned slices and the request-scoped tool IDs (below) before rewriting. Once `sessions` computes real IDs, it does one rewrite pass over `Entry.RequestID` and every `"req:" + <temp key> + ...` tool ID, substituting the final ID for the temporary one; nothing downstream of that pass — renderers, the shared-JSON projection, `annotateExtensions` — ever sees a temporary key.

`ParseLMStudioRecords` lexes headers and records (above), classifies each, and for requests: decodes the body, extracts `client_metadata` (thread/parent-thread/request-kind — a blob that fails to parse into those fields is left inside `Parameters` verbatim, with `ThreadID`/`ParentThreadID`/`RequestKind` left empty as the "no field was recognised" signal per 3.7, rather than a separate marker field), splits recorded conversation content into role-tagged items per the endpoint's shape (`messages`; `messages` + separate `system`; `input` + `instructions`; `prompt`; native chat's string or typed `input` as supplied user context, `system_prompt` separate — 2.2), and keeps every other top-level field as `Parameters`, including `previous_response_id` or another server-held-context reference (2.7) — preserved verbatim, never read as a grouping or attribution key; only `client_metadata.thread_id` ever drives session identity. A recognised item whose shape doesn't decode into any defined translation (2.3) becomes a generic `event`-kind block (`EventType` the recorded item type, `Payload` its compact JSON — the same fallback `codex-exec`'s own unknown items already use) with a `WarningUnrecognizedRecord` warning, distinct from `BodyState`, which describes the whole body's decode outcome, not one item inside an otherwise-decodable body (2.3, 4.4). `<Truncated in logs>` inside a string value marks that item truncated (2.6) without shortening it further.

A `tool_use`/`tool_result` pair is ID-scoped to the request or event it came from. Request-embedded pairs use `"req:" + RequestID + ":" + originalID`, where `RequestID` is the temporary `StartLine`-based key described above until `sessions` rewrites it to the final ID — a threaded session's requests can be merged from several files, so the rewrite must happen before any cross-file comparison, and a per-file ordinal would collide across files if it weren't rewritten at all. Event-embedded pairs use `"evt:" + strconv.Itoa(SourceLine) + ":" + originalID` and are never rewritten, since unattributed content is never merged across files — a logical-session view never renders it (Components 6), so staying unique within one file's parse is sufficient. Either way the original ID is kept on `ContentItem.ItemID` for the prefix comparison below. Malformed or oversized bodies still carry the recorded endpoint, timestamp and thread fields when the header/metadata region parsed; only the body content state changes.

**Extension-prefix comparison** (3.5, 3.6) runs once per thread over its requests in full source/discovery order — which, for a session spanning several files, is only known after [Components 6](#6-session-assembly-and-resolution) has merged them, not inside this file-scoped parser. `annotateExtensions(requests []LMStudioRequest, entriesByRequest map[string][]Entry)` is the one comparison implementation both paths call: the direct view calls it on one file's own already-ordered requests (trivially "merged," since there is only one source); session assembly calls it on the merged, discovery-ordered sequence for one thread. Items are equal when instructions, role, content, type and recorded identifiers match (object-key order ignored, array order and string content significant, per the requirements appendix). When request *N*'s sequence has *N−1*'s complete sequence as an unbroken leading prefix, `PriorPrefixCount` is that prefix's length and only the suffix renders in Markdown/HTML (shared JSON always keeps every item). Otherwise `ContextDiscontinuity` is set and nothing is suppressed. Comparison never crosses a singleton boundary and never affects WARN/ERROR/response presence.

**Time.** Each request's recorded wall time is interpreted via `zone` ([Components 3](#3-time-zone-resolution)); `LMStudioRequest.ResolvedTime` carries the offset-bearing instant when the interpretation is unique (5.2), and stays zero otherwise, with `Timestamp` always holding the wall-time string as logged (5.2, 5.3). Source order — the order `ParseLMStudioRecords` already preserves — is never derived from these timestamps (5.3).

### 3. Time zone resolution (`transcript`)

```go
type zoneResult struct {
    loc     *time.Location
    label   string // what render output calls the zone
    warning string // "" when resolution succeeded
}
func resolveZone(tzEnv string, tzSet bool) zoneResult
```

| `TZ` | Resolution |
|---|---|
| unset (`tzSet == false`) | `time.Local` (host zone) |
| set, empty | `time.UTC` |
| set, `[:]<IANA name>` | `time.LoadLocation(name)`; the package blank-imports `time/tzdata` so this succeeds without a system zone database on every release platform (5.5) |
| set, `[:]<absolute path>` (leading `/` on Unix; drive or UNC form on Windows) | `os.ReadFile` then `time.LoadLocationFromTZData(path, data)` |
| anything else that fails to load | `loc: nil`, `warning` set; callers keep wall times and omit interpreted instants, never substituting another zone (5.5) |

`tzSet` comes from `os.LookupEnv("TZ")`, not `os.Getenv`, so an unset variable is distinguishable from an explicitly empty one — the distinction Go's own Unix local-zone init collapses (Decision 10's cited risk). `resolveZone` is called once per parse/listing pass and its warning surfaces exactly once per transcript (4.5, 5.5), not per request.

### 4. Shared JSON projection (`internal/sharedjson`)

Additive to the existing `Session`/`Listing` functions: a document gains a `requests[]` array, one object per `LMStudioRequest` (`id`, `session_id` — the listed logical-session ID this request belongs to, equal to `id` itself for a singleton — `start_line`, `end_line`, `endpoint`, `thread_id`, `parent_thread_id`, `request_kind`, `model`, timestamp fields, `body_state`, `parameters`, `context_discontinuity`, `prior_prefix_count` — each omitted when empty/false/zero, same convention as every other optional field), and each entry gains `request_id` (omitted when empty, i.e. on an unattributed entry) and `supplied_context`/`unattributed` booleans. This keeps `Parameters` and the rest of a request's metadata written once per request rather than once per entry it produced. A session-scoped document's `requests[]` holds only that thread's own requests; direct-view documents keep the unattributed entries themselves (full content, per Decision 7); session-scoped documents instead carry an `unattributed_summary` array (count, source locations, contributing/next files) in their place, matching [Components 6](#6-session-assembly-and-resolution)'s filtered result. The projection has no LM-Studio-specific logic of its own beyond this shape; it renders whatever the parse/assembly layer already produced. Listing rows gain `request_count` (new field, distinct from `size_bytes` — see [Data Models](#data-models)) and reuse the existing `thread_id` listing field `codex-exec` already defined.

Implementation (Q29–Q33): a request's logged wall time is `wall_time` and its interpreted instant `timestamp`, the same pair log-level entries carry, so `timestamp` means an RFC 3339 instant everywhere in the document. Every request also carries `response_attribution: "unknown"` (2.4, Q16). `session_id` comes from a new `LMStudioRequest.SessionID`, which the sessions layer assigns together with `ID`. Log-level entries add `source_line` and `wall_time` (4.2, 5.1), blocks add `truncated` (2.6), and session metadata adds `time_zone` (5.1). The summary's data arrives on `ParseResult.LMStudioEvents`. For this format, session `created_at` comes only from the first request's own time, never from a later entry (5.4).

### 5. Discovery, identity and cache (`sessions`)

**Roots.** `LMStudioLogDir` is a single resolved path (default `~/.lmstudio/server-logs`, applied by the CLI/web config layer exactly like `codexExecRoots`' flag/env/empty-disables resolution), canonicalised with the existing `CanonicalPath`. Discovery lists immediate subdirectories matching `^\d{4}-\d{2}$`, then within each, files matching `^\d{4}-\d{2}-\d{2}\.\d+\.log$`; a matching symlinked directory is not descended, a matching symlinked file is resolved and kept only inside the canonical root (1.3, mirroring `codex-exec`'s rule). Fingerprinting (below) always stats the resolved target, never the link itself, matching `codex-exec`'s existing candidate walk — otherwise an append to a symlinked file's target would never be seen. A discovered file is always treated as an LM Studio log, without re-running `ProbeLMStudio` (1.2) — a rotated file holding only routine/notice records is still part of the root.

**Per-file summary cache.** Listing never touches a file whose cached fingerprint is unchanged:

```go
type lmFileSummary struct {
    requests           []lmRequestSummary
    unattributedCounts map[string]int // "response"/"warn"/"error" -> count, for the 4.2 session-view summary
    warnings           []ListWarning
}
type lmRequestSummary struct {
    id        string    // the request ID (below); computed here, since it needs only this file's path and this request's StartLine
    threadID  string    // "" = singleton
    startLine, endLine int
    byteSize  int64     // the request record's byte span, for the existing session-size browser limit
    createdAt time.Time // LMStudioRequest.ResolvedTime; zero = not uniquely interpretable (5.2); never a filesystem fallback (5.4)
    model     string
}

// lmFingerprint reports whether a file's identity can be fingerprinted at
// all, and if so, its current fingerprint. cacheable is checked before fp is
// ever stored or compared — a non-cacheable result is never a cache hit,
// regardless of what fp holds, so a "same zero value" stub can never be
// mistaken for "unchanged" (see Risks and Assumptions for why this shape,
// not an opaque always-equal/always-unequal value, is the one that matters).
func lmFingerprint(path string, info os.FileInfo) (fp lmFileFingerprint, cacheable bool, err error)
```

`lmFingerprint` has two build-tagged implementations, selected through an injected package-level function variable for testability: `lmstudio_cache_unix.go` (`//go:build unix`) returns `cacheable: true` with `fp` from `golang.org/x/sys/unix`'s `Stat_t` (size, mtime, ctime, inode, device — using that package rather than raw `syscall.Stat_t` because its field names, unlike `syscall.Stat_t`'s, are consistent across Linux and Darwin), promoting `golang.org/x/sys` to a direct module requirement; `lmstudio_cache_other.go` (every other platform, in practice Windows) always returns `cacheable: false`, so the cache degenerates to "always miss" there rather than risking a false hit from a weaker heuristic. `internal/fileidentity` — the equivalent isolated package `apsis-shared-json` planned for its own generic cache — does not exist in the codebase yet (that PR phase is unimplemented), so there is nothing to reuse today; `lmFingerprint`'s narrow signature is a deliberate seam this cache can redirect to that package later without changing any caller. CI runs Ubuntu only, so lint additionally runs `GOOS=darwin go vet ./internal/sessions/...` to check the Darwin build path without native hardware. This platform split is a deliberate, scoped choice (Decision 6), not a parity gap in the format or feature set — see [Risks and Assumptions](#risks-and-assumptions).

```go
type lmCache struct {
    mu      sync.Mutex
    entries map[string]lmCacheEntry // canonical file path -> fingerprint + summary
}
type lmCacheEntry struct {
    fingerprint lmFileFingerprint
    summary     lmFileSummary
}

// buildLMStudioIndex scans every discovered file under root and returns the
// merged thread index plus listing warnings. cache == nil means "never hit,
// never store" — every call re-reads every body, matching every other
// source's per-invocation resolution cost. cache != nil holds its own mutex
// for the whole call, so two overlapping passes (an apsis serve page load
// racing an HTMX poll) serialise rather than updating the map concurrently.
func buildLMStudioIndex(root string, cache *lmCache) (index map[string][]fileRef, warnings []ListWarning, err error)
```

A `Lister` held across `apsis serve` refreshes carries one `*lmCache`, created lazily on first use, and passes it to every `buildLMStudioIndex` call it makes. `Resolver.resolveLMStudio` ([Components 6](#6-session-assembly-and-resolution)) passes `nil` — a `Resolver` is constructed fresh per call in both the CLI and `apsisweb`'s handlers, with no cross-call state of its own for any existing source, and this one is no exception. A threaded-session open therefore always re-reads every file under the root once, the same cost a cold listing pays; [Components 6](#6-session-assembly-and-resolution)'s fast path for singleton sessions avoids this entirely for the common case.

**Identifiers** (3.3, 3.4). A threaded session's identity can't be tied to one file, but a request's can — and that difference is deliberate, not an oversight, because it lets opening a singleton session (the common case: "unidentified multi-turn clients produce one listing per request," Decision 1) skip the index build entirely:

| ID | Formula |
|---|---|
| request, path form (used whenever it fits) | `"lmstudio-req-p-" + base64url(rootRelativePath, no padding) + "-" + strconv.Itoa(startLine)` |
| request, hash form (only when the path form would exceed 256 characters) | `"lmstudio-req-h-" + hex(sha256(rootRelativePath))[:48] + "-" + strconv.Itoa(startLine)` |
| singleton logical session | same as its one request's ID (either form) |
| threaded logical session | `"lmstudio-thread-" + hex(sha256(threadID))[:48]` |

`base64url` is exactly reversible, unlike a lossy character-replacement sanitiser, and its alphabet (`A-Za-z0-9_-`, no padding) already fits the existing ASCII charset constraint — so a `"lmstudio-req-p-"` ID carries its own file path, and [Components 6](#6-session-assembly-and-resolution) can decode one directly back to a path and a line without consulting any index. The hash form exists only to keep very long paths under the 256-character cap. The two forms need an explicit discriminator (`-p-` vs. `-h-`) rather than relying on decode success to tell them apart: a 48-character hex digest is itself valid base64url (it decodes to 36 unrelated bytes instead of failing), so a bare "did base64url-decode succeed" check cannot distinguish an intentionally-hashed ID from a path that happens to decode. Both forms hash or encode the full, untruncated input, so there is no truncation-collision surface and no conflict-detection warning code is needed here, unlike `codex-exec`'s path-mixed identifier. Every formula is a pure function of already-local data (this file's root-relative path and the request's own start line; the thread ID text) and never reads another file, so direct conversion of a canonical in-root file reproduces discovery's IDs without opening anything else (3.4). A direct file outside every configured root, or stdin, uses the same formulas keyed on the absolute path (or a fixed `"stdin"` marker) instead of a root-relative one, and the result is documented as input-scoped rather than a listed identifier (3.4), matching `codex-exec`'s existing input-scoped/listed distinction.

Implementation (Q34–Q38): `lmFingerprint(path)` has no `os.FileInfo` argument. The summary keeps `transcript.LMStudioEventCounts` plus event start lines instead of the `unattributedCounts` map, so session assembly can fill `LMStudioEventSummary` directly. *(Superseded by Q39: the summary keeps no event counts and `lmIndex` no file list, since session assembly counts events from its own parse.)* `buildLMStudioIndex` returns an `*lmIndex` (discovered files, per-file summaries, session ID → contributing files, session order); its error reports only an unreadable root. `lmStudioRoot`/`findLMStudioFiles` follow the root rules above, and a symlinked file is keyed by its target's root-relative path (Q37). A request's byte span comes from the new `LMStudioRequest.Size` (Q36). Listing warnings are file-level only (Q35).

**Listing** (7.1, 7.2, 7.4). `listLMStudio(projectPath string)` returns nothing when `projectPath != ""`, unconditionally — this source never appears in project scope regardless of configuration, unlike `codex-exec`'s project-matching branch. In all-projects scope, `buildLMStudioIndex` (using the `Lister`'s persistent cache) returns one `SessionInfo` per thread index key: `ID`, `RequestCount` (new field, not `Size` — see [Data Models](#data-models)), `Model` (first request's), `Provider: "lmstudio"`, `ThreadID` (reusing the existing field `codex-exec` defined), and `CreatedAt` from the thread's first request **in discovery order** (files ordered by filename date, then numeric suffix, then physical position — the Terms section's "source order," not chronological order by recorded time). When that specific first request has no uniquely interpretable time, `CreatedAt` stays zero — unknown, per the existing extended-ordering "unknown times first" rule — rather than substituting a later request's time or a filesystem time; 5.4 forbids both. A missing or unreadable configured root yields no sessions and no warning when it is the untouched default, and a `root_unreadable`/`path_unreadable` warning (reusing the existing codes) when the root was explicitly configured (7.4).

### 6. Session assembly and resolution (`sessions`)

Opening a listed session needs a filtered, assembled `transcript.ParseResult`, not a single file's parse — no `Resolver.ResolvePath` equivalent exists for this source (`ResolvePath(SourceLMStudio, id)` returns an error, matching the existing per-source error for Kiro CLI's SQLite-backed sessions).

```go
func (r *Resolver) resolveLMStudio(id string) (*ResolvedSession, error)
```

The three identifier forms take different paths:

- **`"lmstudio-req-p-"` (singleton session, or a direct request lookup):** decode the embedded path segment with `base64url`. Join it with the configured root, re-canonicalise and recheck containment (the existing root-safety discipline), and open **only that one file** for its request content — no index build, no other file's body touched. Parse it, keep the request at the decoded start line.
- **`"lmstudio-req-h-"`:** the path isn't recoverable from the ID; locate it the same way a threaded session does, below, matching the full ID against every candidate's computed hash form. *(Superseded by the implementation note below, Q39.)*
- **`"lmstudio-thread-"` (threaded session):** call `buildLMStudioIndex(root, nil)` (always a fresh, uncached pass — see [Components 5](#5-discovery-identity-and-cache)), find the matching thread's file set, and re-run `ParseLMStudioRecords` on exactly those files — never every file under the root. *(Superseded by the implementation note below, Q39.)*

The singleton fast path still needs to name the file's *successor* in discovery order, since 4.2 requires contributing-file-and-next-file instructions for any unattributed events the opened file contains. That costs one directory listing of the root's filenames (`os.ReadDir`, no bodies opened) to place the file in discovery order and name the next one — not a full index build, and not a second body read.

Whichever path resolved it: merge the resulting requests into discovery order, run `annotateExtensions` ([Components 2](#2-lm-studio-record-parser-transcript)) over the merged sequence, and replace every unattributed entry from those same files with a summary (count, root-relative location, contributing/next-file-by-discovery-order instructions, from the cached `unattributedCounts` when available) rather than its content — satisfying 4.2 without a browser file route (4.2 explicitly rules one out). *(The cached counts are superseded by Q39; the counts come from the same parse.)*

The assembled `*transcript.ParseResult` is attached directly to the resolved session rather than threaded through a reader: `SessionMetadata` gains `PreParsed *transcript.ParseResult` (lmstudio only), `ResolvedSession.Reader` is a no-op closer with no bytes, and `sessions.ParseResolved` checks `rs.Metadata.PreParsed != nil` first and returns it immediately, never touching `rs.Reader`. This is what lets a session assembled from several files pass through the same `ResolvedSession`/`ParseResolved` plumbing every other source uses, where `Reader` is otherwise assumed to be one openable stream. `SessionMetadata.Path` is left empty (no single file), so follow mode's existing `ResolvePath`-based rejection pattern does not apply here — `apsis -F` on an `lmstudio` ID is instead rejected the same way the `codex-exec` ID case already is (8.1), by checking the resolved source before attempting to follow.

A session with no surviving requests (every contributing file replaced or truncated below its request) disappears from the next listing pass (7.3) because the thread index simply has no entry for it; resolution of a now-absent ID returns `session not found`, matching every other source's convention.

Implementation (Q39–Q42): resolution needs no index. `findLMStudioFiles` lists the root (metadata only); a request identifier of either form is matched against each discovered file's computed `lmRequestID`, so a `-h-` open reads one file like a `-p-` open, and a thread identifier parses every discovered file once, keeping files that hold the thread. An identifier of neither shape, project scope or a disabled source returns `session not found` before any body read. Each file is parsed with `transcript.ParseLMStudio`, its containment rechecked on the opened path, and its temporary keys replaced through `transcript.AssignLMStudioIDs`; `transcript.AssembleLMStudioSession` then merges the selected requests (`ID` or `SessionID` equal to the opened identifier), sets `LMStudioRequest.File`, re-runs `annotateExtensions`, builds `LMStudioEvents` with `transcript.CountLMStudioEvents`, keeps only warnings inside a selected request's lines (file-prefixed) plus one time-zone warning, and sets `Metadata.ThreadID`. An unreadable file of a thread becomes a `path_unreadable` warning; a request's one file being unreadable is an error. `SessionMetadata` carries `CreatedAt` (first request's time or zero), `Size` (selected record bytes), `ThreadID` and `PreParsed`; `ResolvePath` errors. Markdown and HTML session views end with the event summary section (Q41).

### 7. CLI surface (`apsis`)

| Option | Behaviour |
|---|---|
| `-a lmstudio` | forces the record parser on a file or stdin |
| `--lmstudio-log-dir <dir>` | the discovery root; empty value disables discovery; CLI value replaces the environment variable entirely (no merge) |
| `APSIS_LMSTUDIO_LOG_DIR` | same root, read when the flag is absent; unset resolves to the default `~/.lmstudio/server-logs` |
| `-s lmstudio <id>` | existing `--source` mechanism; no new flag needed |

For direct file conversion, a new `lmStudioInput(in inputContext, requests []LMStudioRequest) inputContext` mirrors the existing `execInput` helper: it computes each request's ID from the file's path relative to the configured root when the canonical file lies inside it, or from the absolute path (input-scoped, not a listed ID) otherwise, so 3.4's "direct conversion inside the root reproduces discovery's IDs" holds without a separate code path.

`apsis serve` accepts the same flag/variable. Both resolve the root once, the same way `codexExecRoots` already resolves its flag/environment pair, before constructing `sessions.Options`. `-h`'s usage text and `apsis serve`'s usage text each gain a line for 7.6: singleton rows grow one-for-one with unthreaded requests, and an enabled default root participates in all-projects `latest`, which can select an unsupported-mode LM Studio session — naming the source and pointing at `--lmstudio-log-dir ''` or explicit session opening as remedies, extending the same usage block 7.5's disclosure already extends.

Implementation (Q43–Q46): `Config.LMStudioLogDir` is a `*string` that is nil when the flag is absent, and `lmStudioLogDir` resolves it together with `APSIS_LMSTUDIO_LOG_DIR` (read with `os.LookupEnv`, so an empty value disables the source) into the directory and the `explicit` flag Q34 needs. `lmStudioInput` calls `sessions.AssignLMStudioInputIDs(res, path, logDir)`, since the identifier formulas live in `sessions`, and the document keeps the input's own identifier. Follow-mode file checks use `DetectFormat`, which covers both exec streams and LM Studio logs. An explicit `-a` does not re-parse an assembled session.

### 8. `apsis serve`

`apsisweb.Config` gains `LMStudioLogDir string`. The existing `0.0.0.0`/discovery disclosure warning (7.5) extends to name LM Studio's full-prompt and credential-bearing-header exposure whenever this root is configured, alongside the existing `codex-exec` text. The transcript view for an `lmstudio` session shows its parse warnings with the existing `.warning-banner` style and a thread-ID/correlation line in `session-meta`, matching the `codex-exec` precedent.

Implementation (Q44): `Config` also carries `LMStudioLogDirExplicit`, and `New` passes the directory on only in all-projects scope. `Config.NetworkWarning()` builds the `0.0.0.0` warning, so `serveCommand` prints the disclosure only when the source is listed. The oversized-session error names `apsis --all-projects -s lmstudio <id>`, and listing rows show request counts. The badge is pink (`.source-lmstudio`), and `CSSVersion` is 4.

### 9. Rendering of new content

The existing patterns are the baseline; only differences are listed.

| Content | Markdown | HTML |
|---|---|---|
| request boundary | a rule + meta line: endpoint, recorded/assumed time with zone label, thread correlation ID when present, `(extends prior request)` or `⚠ context discontinuity` | same parts via the existing group-header pattern |
| suppressed prefix | `⤷ repeats the first N items of the previous request` in place of those items | same text in the fragment markup |
| unattributed event | collapsed block, `Unattributed · WARN` / `Unattributed · response` / `Unattributed · ERROR`; one leading notice per transcript explains the common state once (4.5, Q16) rather than repeating it per block | same collapsed-block pattern as `codex-exec`'s `event` kind |
| `Parameters` | labelled collapsed `<details>` block: a one-line stream/tool-name summary, then the remaining fields as escaped JSON, values and truncation markers unchanged (Decision 9) | same block, same escaping rule as every other raw-JSON block |
| truncated content | existing truncation marker, wherever `<Truncated in logs>` was recorded | same |
| zone warning | one line near the document header, not per request | same |

## Data Models

### Entry/content additions

Covered in [Components 1](#1-shared-representation-additions-transcript): one new boundary type, `LMStudioRequest`, parallel to `ParseResult.Entries`; no new `ContentItem` kinds beyond the existing `event`/`error`/`usage`/`tool_use`/`tool_result`/`text`/`thinking` set, which this source reuses rather than extending.

### Session document additions

| Field | Derivation | Omitted when |
|---|---|---|
| `requests[]` | one object per `LMStudioRequest`: `id`, `session_id`, `start_line`, `end_line`, `endpoint`, `thread_id`, `parent_thread_id`, `request_kind`, `model`, `wall_time`/`timestamp` (supersedes the earlier `timestamp`/`resolved_time` naming, Q29), `body_state`, `parameters`, `context_discontinuity`, `prior_prefix_count`, `response_attribution` | the whole array, when there are no requests (an empty direct-view input); each field within an object, when empty/false/zero |
| entry `request_id` | `Entry.RequestID` | empty (unattributed entries) |
| entry `supplied_context`, `unattributed` | `Entry` fields | false |
| session `unattributed_summary[]` | assembled per [Components 6](#6-session-assembly-and-resolution), session-scoped documents only: `file`, `next_file`, `counts` (`response`/`warn`/`error`), `lines` (Q31) | direct-view documents |
| entry `source_line`, `wall_time` | `Entry` fields, log-level entries | request entries |
| block `truncated` | `ContentItem.Truncated` | false |
| session `time_zone` | `ParseResultMetadata.TimeZone` | other formats |

### Listing document additions

| Field | Derivation | Omitted when |
|---|---|---|
| `request_count` | thread index entry count (7.2); replaces `size_bytes` for this source, since "size" in the requirements names a request count here, not a byte count, and overloading the existing byte-sized field would mislabel it in the schema | not an `lmstudio` row; `size_bytes` is omitted on `lmstudio` rows (Q32) |
| `thread_id` | the recorded value (existing field, reused) | singleton |

Text listing (`apsis -l`) prints `lmstudio` rows' count directly (e.g. `42 requests`) instead of calling `FormatSize`, which otherwise reads that column as a byte count.

## Error Handling

| Condition | Behaviour |
|---|---|
| `-F` / raw `-f json` on `lmstudio` input, an `lmstudio` ID, or `latest` resolving to one | rejected by name, matching the existing `codex-exec` rejections (8.1); the raw-JSON error directs to shared JSON |
| configured root missing/unreadable (explicitly set) | `root_unreadable` warning (existing code), other sources' results kept |
| default root missing | no sessions, no warning (7.4) |
| unreadable file under the root | `path_unreadable` warning (existing code) |
| malformed request body | `BodyState: "malformed"`; `invalid_record` warning on stderr, in shared JSON and in the web transcript view |
| oversized request body | `BodyState: "oversized"`; `oversized_record` warning, same surfaces |
| incomplete request body (final record at EOF) | `BodyState: "incomplete"`; `invalid_record` warning naming the record's lines, same surfaces (2.5, Q47); replaced on the next refresh once the record completes (6.1) |
| unrecognised item shape inside an otherwise-decodable body | generic `event`-kind block; `unrecognized_record` warning (2.3, 4.4) |
| unsupported POST path | one aggregated `unsupported_endpoint` warning per path with a count (4.6) |
| text before the first header, malformed header-like text outside a body, text after a body, or no recognised records | `unparsed_text` warning naming the line range (6.3) |
| zone override invalid or unloadable | one visible `time_zone` warning per transcript, without a line; wall times retained, no interpreted instant |
| `Resolver.ResolvePath(SourceLMStudio, id)` | always an error: no single file backs a logical session |
| session ID with no surviving requests | `session not found`, same as every other source |

## Risks and Assumptions

- Assumption: the per-file cache's Windows behaviour (always non-cacheable; see [Components 5](#5-discovery-identity-and-cache)) means requirement 7.3's zero-body-read guarantee holds on Unix/macOS only — amended into 7.3 itself (Q20), reflecting the user's explicit choice to descope native Windows fingerprinting rather than repeat `apsis-shared-json`'s unresolved native-Windows-verification blocker. Windows still refreshes correctly; it simply re-reads the root's bodies on every poll. | Verify: covered directly by the cache's own tests (Unix cacheable/fingerprint behaviour; the `cacheable` flag, not fingerprint equality, gates every hit, so a same-shaped stub can never be mistaken for "unchanged" on the other-platform build).
- Assumption: the active, currently-growing log file changes on most refreshes and so gains no benefit from the cache — it is fully re-parsed whenever its fingerprint changes, with no incremental/resume parsing (Q21). This is expected, not a defect: 7.3's guarantee is about an *unchanged* tree, and the source's own date/.N rotation keeps any one file from growing unboundedly within a day. | Verify: covered by the cache's own change-detection tests; no separate investigation needed.
- Assumption: a singleton session can disappear from a later listing — not merely change ID — if its one request's body was `incomplete` on an earlier pass and, once complete, reveals a `thread_id` that wasn't visible before; the request then joins a threaded session instead (3.1) under a different, hash-based session ID. This is compliant, not a 3.3 violation: 3.3's stability guarantee covers files being added/removed with the *recorded identity unchanged*; 6.1 only promises the request's own ID stays stable across the incomplete-to-complete transition, not that session grouping is immune to a record revealing new content about itself. | Verify: a dedicated test completes a previously-incomplete singleton request with a `thread_id` across two refreshes and asserts the singleton row disappears while the request's own ID (now inside the thread) is unchanged.
- Risk: a large unthreaded root (many singleton sessions, one row per request) could make a cold listing slow, since every file must be read once regardless of caching. | Verify: a cold-listing benchmark against a generated large singleton-heavy tree, recording file/request counts and elapsed time, before claiming a performance result. | If wrong: no fallback is required by the requirements (8.5 asks only for the benchmark, not a hardware-independent threshold); revisit only if a measured regression is reported.
- *(Superseded by Q39: a hash-form open reads only its one file, and a threaded open parses every discovered file once without building the index.)* Assumption: opening a **threaded** session, or a singleton session whose ID happened to need the hash form (`"lmstudio-req-h-"`), always rebuilds the index cold (`buildLMStudioIndex(root, nil)`), paying the same cost as a cold listing before narrowing to its files — the `Resolver` holds no cache of its own, matching every other source's stateless per-call resolution (e.g. `codex`'s `selectCodexCandidate` walk). Opening a singleton session with a `"lmstudio-req-p-"` ID — the common case for any path short enough to encode directly — skips this entirely, covering the common case Decision 1 calls out ("unidentified multi-turn clients produce one listing per request"). | Verify: covered by the cold-listing benchmark for the threaded/hash-form path, and a dedicated test asserting zero file reads outside the one decoded file for the path-form singleton case.
- Risk: `resolveInput`'s bare-ID fallback (no explicit `-s`) tries every source in turn, and a default, non-empty LM Studio root means an unmatched bare ID shaped like `"lmstudio-thread-"` or `"lmstudio-req-h-"` now also pays a full cold index build before `apsis` reports "session not found." *(Narrowed by Q39: only a well-formed thread identifier reads the root's bodies; request identifiers and malformed identifiers read none.)* A bare ID shaped like `"lmstudio-req-p-"` does not, since that form resolves without an index. | Verify: measure the index-building fallback path's added latency with a populated default root in the CLI test suite. | If wrong (measured latency is user-visible): move `lmstudio` to the end of the fallback order (already planned) and document that `-s lmstudio` or a file path avoids the fallback scan entirely; do not weaken the thread index itself.

## Testing Strategy

Fixtures are synthetic text logs, with provenance (reported 0.4.26 structure vs. publicly documented endpoint payloads) recorded in `testdata/lmstudio/PROVENANCE.md` (8.4).

| Requirement | Tests |
|---|---|
| 1.1, 1.2 | Detection table: included/excluded/unsupported endpoints, headers-without-request suggesting `-a lmstudio`, a Kiro-marker body string inside an LM Studio request not overriding the signature, discovered rotated notice-only files still treated as logs |
| 1.3 | Root/file symlink resolution, directory-symlink non-descent, escaping-symlink rejection |
| 1.4, 1.5 | Direct multi-session file parity across Markdown/HTML/shared JSON; a listed session open returning only its own requests plus unattributed summaries |
| 2.1–2.9 | One fixture per included endpoint and both native-chat input forms; thinking/tool calls/tool results; malformed/missing/oversized/truncated bodies; parameters block round-trip; an unrecognised item shape inside an otherwise-decodable body produces the generic `event` fallback with its warning, not silence |
| 3.1–3.7 | Property test (`rapid`): random request-item sequences — extension detection is correct exactly when one sequence is the other's leading prefix, under the stated equality rule (object-key order irrelevant, array order and string content significant), asserted through `annotateExtensions` directly so the property covers both the direct-view and session-assembly call sites; the parser's temporary `StartLine`-keyed `Entry.RequestID`/tool-ID references are fully rewritten to final IDs before any entry reaches a renderer or the shared-JSON projection, with no lingering temporary key; truncated thread IDs always classify as singleton; request/session identifiers are deterministic, the `-p-` form round-trips through `base64url` decode back to the original path, the `-h-` form never does (asserted directly, since a 48-hex digest is itself valid base64url input), and both are unchanged under append, added-earlier-file and other-file-removal, property-tested per 3.3/3.4 the same way `codex-exec`'s identifier property test works today; the `-h-` form is exercised with a path long enough to force it |
| 4.1–4.6 | Attribution always unknown across rotation and file-start edge cases; direct view keeps full unattributed content, session view keeps only the summary; unsupported-path aggregation counts correctly, including at WARN/ERROR level; a WARN/ERROR excluded-endpoint diagnostic (startup/model-management/documented non-inference path) is retained, not silently dropped, while the same diagnostic at another level is ignored (the classification table's row-3-before-4/5 ordering); parity across Markdown/HTML/web/shared JSON for one view scope |
| 5.1–5.5 | `resolveZone` table: unset/empty/named/path/`:`-prefixed/invalid TZ, exercised with `t.Setenv` and `t.Helper()`-wrapped assertions; ambiguous/nonexistent wall times omit an instant without reordering; source order preserved across a backward clock change |
| 6.1–6.4 | Incomplete-at-EOF then completed-on-refresh; malformed/pre-header/no-recognised-record recovery continuing at the next header; oversized-record skip and recovery |
| 7.1–7.6 | Project-scope read-nothing guarantee; cache hit/miss over append/replace/truncate/remove, asserting zero body reads on an unchanged Unix pass via a counting opener; `cacheable: false` on the other-platform build never produces a hit regardless of how two fingerprints compare (the inversion the peer review flagged); overlapping `buildLMStudioIndex` calls against one shared `*lmCache`, run under `-race`, serialise rather than corrupting the map; default-root-absent silence vs. explicit-root-absent warning; help text disclosure; `latest` inclusion and its unsupported-mode error naming the source |
| 8.1–8.5 | Follow/raw-JSON rejections; `sessions.ParseResolved` returns `PreParsed` directly without touching `Reader` for an assembled session; a `-p-` singleton open touches only its one decoded file plus one directory listing for successor-file naming, while a `-h-` singleton or threaded open builds the full index *(superseded by Q39: a `-h-` open also reads one file; a threaded open reads every file once)*; a 4.2 successor-file reference is correct for a `-p-` singleton open without that file's successor ever being parsed; the full verification matrix 8.3 lists, reusing the fixtures above; schema conformance for the new entry/listing fields, including `requests[].session_id`/`start_line`/`end_line`, against the existing shared-JSON test infrastructure; cold/unchanged-refresh benchmarks recording generated workload and environment per 8.5 |
