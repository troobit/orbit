# Design: Apsis Shared JSON and Codex Exec Streams

## Overview

Add an explicit JSON projection of parsed sessions, an exec-stream parser, and a reusable session catalogue shared by CLI and web requests. The approved [requirements](requirements.md) and [decisions](decision_log.md) govern this implementation plan.

## Architecture and Placement

Keep `transcript.Entry` as the parser/rendering boundary. Extend it for new content and metadata; project it into public JSON types instead of exporting its Claude-shaped JSON tags.

| Component | Location | Contract |
|---|---|---|
| Public documents and encoder | `internal/transcript/document.go`, `json.go` | Project parsed entries and metadata without presentation grouping/truncation. |
| Exec types/parser | `internal/transcript/codex_exec_types.go`, `codex_exec_parser.go` | One item state per turn/item, stable first-appearance order, structured warnings. |
| Session catalogue | `internal/sessions/catalog.go`, `metadata.go` | Discover metadata and private locators; publish immutable, deterministically ordered snapshots. |
| Exec discovery | `internal/sessions/codex_exec.go` | Resolve containment before reads; identify canonical files and owning repositories. |
| Published schema | `docs/schemas/apsis-v1.schema.json` | Draft 2020-12 schema for both document kinds, version 1. |
| CLI settings/dispatch | `cmd/apsis/main.go` | Resolve flags/environment once; reuse one catalogue for list/latest/ID resolution. |

Existing `NewLister` and `NewResolver` remain compatible. Add catalogue-backed operations for apsis's new paths; keep legacy bare-ID lookup priority and direct file/stdin conversion. A web server owns one catalogue for its lifetime.

### Pattern Extension Audit

The audit covers production references found by searching parser/render/source functions and constants.

| Existing site | Needs equivalent? | Change |
|---|---|---|
| `types.go`: Format and session source constants, display names/validation | Yes | Add `FormatCodexExec` and `codex-exec`. |
| `DetectFormat`, `detectJSONLFormat`, `detectFormatFromLine`, `Parse`, `ParseJSONL`, `ParseJSONLWithFormat` | Yes | Recognise/dispatch exec while preserving legacy parsers and BOM handling. |
| CLI `agentToFormat`, `convert`, `convertToJSON` | Yes | Forced exec parser, shared encoder, raw exec events before renderer selection. |
| CLI `listSessions`, `resolveLatestSession`, `runLatest`, `resolveInput` | Yes | Use scope and catalogue locators; preserve source selection through conversion. |
| CLI `resolveFollowInput`, `validateFollowMode`, latest follow branch | Yes | Reject exec after source resolution as well as forced-format selection. |
| `Lister.ListAll`, source collectors, sorting | Yes | Recorded metadata, global rollout precedence, exec roots, stable ties. |
| `Resolver.Resolve`, `ResolvePath` | Yes | Source/ID lookup through private locators; Kiro global opening. |
| `apsisweb.New`, `buildSessionListData`, `handleTranscript` | Yes | Shared catalogue, parse format/options from resolved source, warnings and metadata. |
| Web source filters, session links and badges | Yes | Register the source using existing source badge/filter patterns. |
| `RenderMarkdown`, `RenderEntries`, `RenderHTML`, `RenderHTMLFragment` | Yes | New blocks through the common content dispatch; JSON bypasses rendering. |
| `preprocessEntries`, Read/Edit groups, `toolResultInfo`, tool-result HTML | Yes | Carry provider/model metadata and preserve turn/state boundaries. |
| `FormatMessageMetaMarkdown/HTML` and their callers | Yes | Accept optional provider; omit empty parts using existing metadata styling. |
| `Follower` | No new parser | Keep existing live behavior; reject unsupported combinations before entering it. |
| Orbit logging/conversion callers | No new source discovery | Existing parser/render signatures continue to work; optional fields default empty. |

## CLI Contract

| Surface | Choice |
|---|---|
| Shared output | `-f shared-json`; session and `--list` forms use the same schema version. |
| Cross-project scope | `--all-projects` on CLI and serve; reject an explicit project filter combined with it. |
| Source selection | `--source <source>` for ID opening; `-a` continues to select a parser. Reject source selection with file/stdin/list/latest rather than silently interpreting it as a filter. |
| Stream directories | Repeatable `--codex-exec-dir` on CLI and serve, overriding the complete `APSIS_CODEX_EXEC_DIRS` value. |
| Environment directories | `filepath.SplitList` syntax; unset/empty means none. A supplied empty CLI value clears the list; reject mixed empty/nonempty repeated values. |

Resolve environment presence separately from value. Repeated flags preserve paths containing the platform list separator. All-projects changes the scope passed to discovery/resolution; a direct path remains an explicit input independent of configured roots.

## Public Documents and Internal Metadata

Both roots are objects; `kind` discriminates session versus listing. These lines are separate output examples:

```json
{"schema_version":1,"kind":"session","session":{"source":"codex-exec","thread_id":"example-thread"},"entries":[],"warnings":[]}
{"schema_version":1,"kind":"listing","sessions":[],"warnings":[]}
```

Session metadata supports optional `id`, `thread_id`, `created_at`, `project_path`, `model`, `provider` and `size_bytes`. A session document's optional top-level `cost` object contains `amount` and `unit`, retaining existing parsed cost precision. A summary may additionally carry `sort_time:{value,basis:"file_mtime"}`; this never populates `created_at` or entry timestamps.

Entry fields are `role`, `content`, optional timestamp/model/provider, derived `turn_index`, source `item_id` and incomplete state. Content variants are text, thinking, tool use, tool result, error, plan, event and usage. Tool blocks retain `id`/`tool_use_id`, structured inputs/results, status and nullable recorded exit codes. Exit code zero is represented explicitly, not lost through zero-value omission.

Generic event blocks retain type and parsed payload. Usage blocks contain an open field map and `basis:"reported_thread_total"`. Plans contain ordered text/completed items. Warning objects contain source, stable diagnostic code, message, and physical line when applicable; no malformed source body is interpolated into diagnostics.

Add optional internal fields to `Entry`/`ContentItem` and parser metadata for those values. `ParseResult` also carries detected format and source-recorded session metadata, so direct file/stdin conversion can project a document without a catalogue lookup. Keep the wire-entry tags out of the public encoder. Use `json.RawMessage` for uninterpreted payloads and `json.Decoder.UseNumber` when decoding arbitrary exec inputs to avoid introducing float precision loss. Omit unknown metadata; initialise root arrays as `[]`, never `null`. Emit model/provider summaries from first known values while preserving entry changes.

Add `ModelProvider` to `CodexSessionMeta` and capture it in `codexParser.processSessionMeta`; propagate provider state alongside model state into assistant entries and deferred tool metadata. Catalogue summaries and direct conversion then use the same recorded provider.

The schema has common definitions for metadata, content and warnings and allows additive unknown properties. Discriminators constrain known content shapes; event payloads and usage maps remain open. Runtime encoding uses the standard library only. Schema tests use the test-only `github.com/santhosh-tekuri/jsonschema/v6` validator, compiling the checked-in schema without HTTP loading; enable format assertions for emitted timestamps ([validator API](https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v6)).

## Exec Parsing and Rendering

Use the pinned [0.166.0 definitions](https://raw.githubusercontent.com/openai/codex/rust-v0.166.0/codex-rs/exec/src/exec_events.rs) for typed shapes, and the [emitter](https://raw.githubusercontent.com/openai/codex/rust-v0.166.0/codex-rs/exec/src/event_processor_with_jsonl_output.rs) for total-counter semantics. Keep specialised mappings limited to requirement 2.2; other types use the generic block.

A line reader reports physical line numbers and skips malformed lines. Probe through the first parseable nonempty JSON event, preserving consumed input for the selected parser; exec auto-detection requires `thread.started` and a nonempty string ID. An oversized line above the existing 10 MiB parser allowance is discarded through its newline with a warning; I/O errors remain fatal. Forced exec parsing accepts valid errors without thread creation and leaves identity absent.

Increment a derived turn ordinal at `turn.started`; entries before it remain unassigned. Index items by turn ordinal and item ID, recording their first source position. Updates replace state at that position; completion wins over prior snapshots. At EOF or a failed turn, started items lacking completion retain their final snapshot and incomplete marker. Repeated `thread.started` events for the same thread do not reset the ordinal, so appended resumed segments cannot collide through reused item IDs. A different thread ID within one file produces a warning and a generic boundary event rather than rewriting the file's session identity.

Command and MCP activity becomes tool-use/result blocks linked within the item. Qualify internal tool-call identifiers by derived turn and item ID to prevent collisions when resumed segments reuse item IDs; retain the original item ID separately. Preserve command status separately from exit code; absent exit code is unknown. File changes retain path/kind/status; searches retain query/action/results; plans retain their final ordered state. Emit usage at its completion position with the associated turn, retain every field, and calculate no sums or deltas.

New content dispatch renders explicit error, plan, generic event, usage and incomplete blocks in Markdown and common HTML entry rendering. Match existing thinking/tool-result disclosure patterns; show generic JSON as escaped code, never HTML. Provider propagation includes standalone deferred tool-result sections and Read/Edit groups. Prevent groups from crossing differing turn/model/provider/state boundaries; unchanged old entries retain their existing grouping and output.

Raw exec JSON follows `convertToJSON`'s JSONL event-array convention. Follow validation occurs both before conversion and after resolving `latest`/an ID so indirect exec selection cannot enter the Claude line follower.

## Catalogue, Identity and Opening

```go
type Options struct { ProjectPath string; CodexExecDirs []string }
type Key struct { Source, ID string }

func NewCatalog(Options) (*Catalog, error)
func (*Catalog) List(context.Context) (Snapshot, error)
func (*Catalog) Resolve(context.Context, Key) (*ResolvedSession, error)
```

Snapshot summaries contain public metadata plus private locators. A locator records canonical file path, source format, project/directory key, workspace and execution-detail path as applicable; it is never serialized. Web handlers accept source/ID only and resolve those against the current scoped snapshot.

| Source | Global locator and project extraction |
|---|---|
| Claude | Retain the discovered project-directory/file path; read recorded `cwd` rather than attempting to decode its directory name. |
| Codex | Retain canonical rollout path; read `session_meta` identifier, timestamp, cwd/provider and first known model. |
| Copilot | Retain events path and `workspace.yaml`; missing workspace leaves project unknown and warnings follow current semantics. |
| Kiro CLI | Retain `logs.SessionMetadata.Directory` from `DiscoverAll`; pass that exact key to `GetSession`. |
| Kiro IDE | Retain discovered workspace, chosen chat path and `KiroIDEExecutionDetailPath`; never hash the caller's unrelated current directory or reverse the workspace hash into a project. |
| Exec | Retain canonical file, first thread ID and nearest repository/worktree root. |

For legacy duplicate copies sharing a source/ID, choose a deterministic canonical-path representative and report conflicting locators; do not conflate equal IDs from different sources. Preserve existing source IDs and legacy single-project resolver behavior.

Canonicalise roots once and de-duplicate identical roots, retaining nested explicit roots even when parent descent would skip them; ignore directory symlinks, and resolve each file symlink before any content read. Require the resolved candidate to lie within at least one configured root. Skip approved subdirectory names during descent, including when deciding which roots' discovered files overlap; explicitly configured roots are scanned independently.

Identify repositories using ancestor `.git` directories or valid `gitdir:` marker files, caching the nearest marker by directory; this requires no installed git executable. File/project paths are normalised consistently with existing `normalizePath`.

Compute IDs as `exec-` plus the full SHA-256 digest of length-delimited thread ID and canonical file path. Detect a digest collision between distinct identities, exclude conflicting entries and warn instead of resolving arbitrarily. Copies/resumed files have separate artifact IDs; appends and overlapping roots preserve existing IDs. Direct files can compute the same artifact ID without configured roots; stdin retains a recorded thread ID but omits the file-artifact ID.

Build a global index of resolvable rollouts by recorded thread ID before project filtering. Suppress matching exec summaries even when the rollout's project is outside the requested scope. Use the rollout's metadata unchanged; direct stream conversion remains independent. Select `latest` by recorded creation time or filesystem sort fallback, unknown times first, then source and ID.

## Refresh and Failure Contracts

The catalogue scans directory names/stat metadata on each refresh but caches extracted summaries by canonical path, file identity, size and nanosecond modification time. Include companion files and repository markers in dependencies. Kiro CLI's source fingerprint includes the database, WAL and rollback-journal files; unchanged fingerprints skip database opening entirely. A changed source refreshes metadata and locators before publishing the next snapshot.

Cache summaries and locators, not full transcripts. Parse once on opening; cache the first-known metadata needed for later listings. When a source cannot expose model metadata cheaply, one initial body parse supplies the summary and the parsed body is discarded. Removed files evict their summaries; additions and replacements are processed normally.

Serialize refresh construction with a mutex and publish one immutable snapshot only after it is complete. Concurrent requests can use the previous snapshot during construction; resolution checks that its selected locator still exists and remains contained. A stale locator triggers one refresh/retry, then a not-found response.

Missing optional standard source locations yield no warning. An explicit exec root that is absent/unreadable, an escaping file link, malformed identity or a failed metadata read yields a structured listing warning while other results survive. CLI renders warnings on stderr; shared documents retain them; web list/transcript pages expose them through existing warning styling. A failed changed file is omitted with a warning rather than silently served from stale cached metadata.

## Testing and Requirement Coverage

Use synthetic fixtures only, with pinned shape provenance separate from observed structural provenance. Run `make test` and `make lint` for implementation; no external CLI is required for tests.

| Requirements | Concrete verification |
|---|---|
| [1.1–1.6](requirements.md#1.1) | Every source's session and listing schema validation; empty arrays, optional omissions, zero exit, cost/unit, model changes, tool links and long payload preservation. |
| [2.1–2.4](requirements.md#2.1), [2.9](requirements.md#2.9), [2.11](requirements.md#2.11) | BOM/malformed prefix, forced pre-thread errors, interleaved updates, failure/EOF, resumed IDs, unknown events and future usage keys; JSON plus all human renderers. |
| [2.5–2.8](requirements.md#2.5), [5.5](requirements.md#5.5) | Overlapping roots, symlink escapes/cycles, explicit skipped-name roots, worktrees, copies, appends, ID bounds and rollout precedence across differing projects. |
| [2.10](requirements.md#2.10), [4.1–4.4](requirements.md#4.1) | Missing times versus mtime fallback, deterministic ties/latest, source collisions, every-source list/open from unrelated directories, Kiro companion details. |
| [3.1](requirements.md#3.1), [5.1–5.4](requirements.md#5.1) | Provider presence/absence, existing output goldens, raw exec events, indirect follow rejection, environment/CLI precedence and repeated paths. |
| [5.6–5.9](requirements.md#5.6) | Provenance checks, renderer parity, cold/unchanged/changed refreshes, instrumented body-read counts and concurrent snapshot requests. |

Use the existing rapid framework for properties: generated legal lifecycle sequences yield one final state per turn/item in first-appearance order; arbitrary payloads survive projection; discovery order/root permutation does not change IDs or results. Generate temporary trees and synthetic IDs/content, not user directories. Preserve exact raw generic payloads when comparing JSON and compare visible text/state in renderer assertions.

## Risks and Assumptions

- Risk: cold metadata extraction is expensive across many sources | Verify: benchmark generated multi-source catalogues before completing discovery work | If wrong: replace full-body summary extraction with source-specific streaming metadata reads while retaining the unchanged-refresh contract.
- Risk: filesystem stat costs dominate broad roots despite cached body reads | Verify: benchmark configured trees during catalogue implementation | If wrong: optimise traversal/stat batching without widening roots or relaxing containment.
