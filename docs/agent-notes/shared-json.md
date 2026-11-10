# Shared JSON Output (Apsis)

`apsis -f shared-json` projects a `transcript.ParseResult` into a versioned document described by `docs/schemas/apsis-shared-json.v1.schema.json`. Spec: `specs/apsis-shared-json/`.

## How It Works

- `internal/sharedjson` holds the document types, `Session` (the projection) and `Write` (two-space indent, no HTML escaping, trailing newline). It imports `transcript` and `sessions`; neither imports it.
- The projection reads parsed data only and never calls renderer helpers, so tool content keeps its full length.
- `cmd/apsis` `convert` enables `ParseOptions.AllowEmptyCodex` and `PreserveModelChanges` only for `shared-json`, and dispatches shared JSON before the `Session contains no entries` early return. Explicit `-a` still selects the parser first.
- Session metadata comes from parsed values first; resolver metadata fills only gaps. `created_at` order (Decision 30): parsed `StartedAt`, then the Kiro CLI `DatabaseCreatedAt`, then a resolver `CreatedAt` not flagged `TimeFromFilesystem`, then the first entry timestamp. Times that do not parse as RFC3339 are omitted; resolver times are written in UTC.
- `session.source` comes from the input context, then the resolver, then `ParseResult.Format`; the Claude fallback (`FormatUnknown`) has none. `session.id` is the resolver ID or the `.jsonl` filename stem; stdin has none.
- `Listing` (`internal/sharedjson/listing.go`) projects `[]sessions.SessionInfo` in the order given; `ListingRow` embeds `Metadata`, adding `size_bytes` and `file_modified_at`. A row carries `created_at` for a recorded time or `file_modified_at` when `TimeFromFilesystem` is set, both in UTC. `Scope` is `{project}` or `{all_projects: true}`. `apsis --list -f shared-json` writes it with extended ordering.
- Warning positions map by `ParseWarning.Unit`: empty is `line`, `transcript.WarningUnitChatMessage` is `chat_message`, `WarningUnitExecutionAction` is `execution_action`.
- Content kinds other than text, thinking, tool_use and tool_result become `other` blocks with `source_type`.

## Schema and Tests

- The schema is open: no `additionalProperties: false`. Known block types are disjoint `oneOf` branches; `extensionBlock` accepts only types outside that list, so a malformed known block cannot slip through as an unknown one. Warning codes are listed as `examples`, not an `enum`, so later codes stay valid (Q101).
- Tests validate against three compiled schemas: the published file, `internal/sharedjson/testdata/apsis-shared-json.v1.initial.schema.json` (a byte copy of the first published schema; never edit it), and a closed copy built in the test with `unevaluatedProperties: false` on apsis-owned definitions. `unevaluatedProperties` is required because listing rows and blocks reuse definitions through `$ref` (Q98).
- The validator (`github.com/santhosh-tekuri/jsonschema/v6`) is imported only from `_test.go` files. Each schema compiles in its own compiler because the copies share an `$id`; `UseLoader(jsonschema.SchemeURLLoader{})` stops any file or network loading.
- Goldens live in `internal/sharedjson/testdata/golden`; regenerate with `go test ./internal/sharedjson -run TestSession_Goldens -update` and review the diff. `testdata/handwritten` holds documents for producers that did not exist when the schema was published (exec streams, listings); they must keep validating against the initial schema.
- Exec streams (PR 3) added the remaining Go types: `Metadata.ThreadID`, `Entry.Turn`, an embedded `BlockCommon` (`item_id`, `status`, `incomplete`) on every block, `ToolResultBlock.ExitCode` (`*int`, so 0 is emitted), and `PlanBlock`, `ErrorBlock`, `EventBlock` and `UsageBlock`. `Counters` marshals usage counters as one object in source key order with raw values, so a counter beyond float64 precision is written unchanged. `ErrorBlock` emits `message` only; the parsed `Payload` is not projected (Q112).
- `SessionContext.Project` carries an exec stream file's containing repository, which the CLI derives (Q110); the precedence is parsed project, then context project, then resolver project.
- `testdata/golden/codex-exec.json` is the exec golden, parsed with `transcript.ParseCodexExec` from `internal/transcript/testdata/codex_exec/all_items.jsonl`.

## LM Studio Additions

- Session documents gain an optional `requests` array (one per `transcript.LMStudioRequest`, omitted when empty) and, for logical-session views only, `unattributed_summary` (from `ParseResult.LMStudioEvents`). Entries gain `request_id`, `supplied_context`, `unattributed`, `source_line` and `wall_time`; `BlockCommon` gains `truncated`; `Metadata` gains `time_zone`.
- A request's `wall_time` is the logged string and `timestamp` its RFC 3339 instant with the assumed zone's offset (not converted to UTC). `timestamp` means an RFC 3339 instant everywhere in the document (Q29).
- `response_attribution` is the constant `"unknown"` on every request, set by the projection (Q30). `session_id` is copied from `LMStudioRequest.SessionID`, which the sessions layer must set together with `ID`. Until it does (Phase 5), direct views show the parser's temporary `StartLine` keys and no `session_id`.
- For `FormatLMStudio`, `created_at` never falls back to entry timestamps: the first request's own time or nothing (requirement 5.4).
- `parameters` is a `json.RawMessage` written as logged; `Write`'s `SetEscapeHTML(false)` also covers raw messages, so `<Truncated in logs>` stays unescaped.
- Listing: `ListingRow.SizeBytes` is `*int64`. LM Studio rows (`sessions.SourceLMStudio`) omit it and emit `request_count` from `SessionInfo.RequestCount`; every other row still emits `size_bytes`, including `0` (Q32).
- Schema validation caught LM Studio `usage` blocks with an empty `basis`; they now use `transcript.UsageBasisResponse` (Q33). Running every LM Studio fixture through `requireValid` (`TestSession_LMStudioFixturesValidate`) is what found it.
- Handwritten `lmstudio-session-view.json` and `listing-lmstudio.json` stand in for producers Phases 5 and 6 add; `TestSession_LMStudioSessionView` and `TestListing_LMStudioRows` build the matching Go values and compare with `JSONEq`. The LM Studio golden is `testdata/golden/lmstudio.json` (from `threads.log` with `TZ=Australia/Melbourne`).
