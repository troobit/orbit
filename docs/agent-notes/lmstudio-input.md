# LM Studio Input Integration (Apsis)

## Current State

- `specs/apsis-lmstudio-logs/` defines the server-log source. Phases 1 (shared representation, time zone), 2 (record parser, detection) and 3 (Markdown/HTML rendering) are implemented. Shared JSON fields, source registration, discovery and the CLI flags are not yet.
- The sibling `apsis-shared-json` supplies the common JSON contract and all-projects scope. Its approved requirements remain authoritative; the LM Studio requirements still have unanswered user-owned choices.
- Public LM Studio API documentation defines endpoint payloads, not the disk-log grammar. Earlier structural inspections are labelled as reported observations in the handoff. Fixtures must be synthetic and distinguish those two kinds of evidence.

## Shared Representation (`internal/transcript`)

- `FormatLMStudio`, `Entry.RequestID`/`SuppliedContext`/`Unattributed`/`SourceLine` (all `json:"-"`) and `ParseResult.LMStudioRequests` exist. `LMStudioRequest` lives in `lmstudio_types.go` with the `BodyState*` constants (`not_logged`, `malformed`, `incomplete`, `oversized`; empty = translated).
- `LMStudioRequests` is parallel to `Entries`, not embedded: entries join their request through `Entry.RequestID`. A request with no translatable body has a request row and zero entries.

## Record Parser (`lmstudio_parser.go`, `lmstudio_translate.go`)

- `ParseLMStudioRecords(r, zone)` lexes physical lines: a header `[YYYY-MM-DD HH:MM:SS][LEVEL][tag]... message` starts a record, and everything up to the next header continues it. A header can't occur inside valid pretty-printed JSON, so any header-matching line ends the previous record. Lines that only start like a header (`^\[\d{4}-\d{2}-\d{2}`) get `unparsed_text` warnings when outside a JSON body. Inside a body they make the body malformed.
- Classification order matters (design table): an included `Received request: POST to <path>` first, then the response markers, then any reported POST path (`\bPOST (to )?/...`, which also catches error texts like `(POST /x)`). Only after that does level decide: WARN/WARNING/ERROR are retained, anything else is ignored. An excluded path at WARN is therefore retained as a WARN entry, with its body text as content.
- Body states: no ` with body` gives `not_logged`. A JSON decode failing with EOF/unexpected EOF on the input's last record gives `incomplete` (no warning). Any other failure, including an unterminated body followed by another header, gives `malformed` with `invalid_record`. Over `lmMaxRecordBytes` (10 MiB, whole record) gives `oversized`; the bytes are counted but not kept. Text after a complete body is `unparsed_text` and the body is still used.
- Temporary IDs: `LMStudioRequest.ID` and `Entry.RequestID` are `strconv.Itoa(StartLine)`. Request tool IDs are `req:<key>:<recorded>` and response tool IDs `evt:<SourceLine>:<recorded>`; the recorded ID stays on `ItemID`. Native chat `tool_call` output records no ID, so it links by `evt:<line>:output-<index>`. The sessions layer must rewrite `req:` keys before merging files.
- One entry per logged item: each element of `messages`/`input`/`prompt`, a string `input`/`prompt`, and separate `instructions`/`system`/`system_prompt` fields. `PriorPrefixCount` therefore counts entries. Entry types are `user`, `assistant` and `system` (system or developer role). Tool outputs are `user` entries with role `tool`, so the Claude-style result pairing still works. Assistant request entries set `SuppliedContext`.
- `Entry.ItemDigest` is SHA-256 over `field + "\x00" + json.Marshal(decodeJSONValue(item))`. `json.Marshal` sorts map keys, so key order doesn't matter, and `json.Number` keeps number text, so `1.0` and `1` differ. `annotateExtensions` compares digests and resets stale annotations, so it can run again after session assembly.
- `Parameters` is rebuilt from `orderedFields` (shared with `orderedCounters`), with source key order and compact values as logged. Only a string `model` and the endpoint's conversation fields are removed. `client_metadata` stays in, even though thread, parent and kind are read from it: flat `thread_id`, `x-codex-parent-thread-id`, and `request_kind`/`parent_thread_id` inside the `x-codex-turn-metadata` JSON string. Any value containing the truncation marker counts as absent.
- Representation choices for later phases are in decision log Q22–Q27. Log entries use `Type: "log"`, with the WARN/ERROR level in the item's `Status`. Unattributed entries carry `WallTime` because they have no `LMStudioRequest`.

## Detection (`parser.go`)

- `DetectFormat` runs `probeLMStudio` after Kiro's validated check and before the exec probe, so a matched log never reaches the exec probe, which would otherwise read the whole file. If the chunk has headers but no included request, the substring Kiro guess is skipped and `errLMStudioWithoutRequest` is returned. `Parse` checks for that error and returns it instead of falling back to the Claude parser.
- `-a lmstudio` in that error message isn't accepted by the CLI until Phase 7 adds it to `agentToFormat`. Auto-detected LM Studio input already renders through the LM Studio renderer (below).

## Rendering (`lmstudio_render.go`)

- `RenderOptions.LMStudio` (a `*LMStudioView` from `NewLMStudioView(res)`, nil for every other format) switches `RenderMarkdown` and `renderEntriesToBuilder` (standalone and fragment HTML) to the LM Studio body. The CLI and `apsisweb.handleTranscript` set it unconditionally. The CLI's "no entries" check also accepts a result that only has requests, since a body-less request still renders its boundary.
- `segments` interleaves `Requests` and log-level entries (`RequestID == ""`) by start line. That only works in a direct view of one input; a session view has no log-level entries, so its requests keep their merged order. The session view's body-free unattributed summary (4.2) is not rendered yet; it needs Phase 6's assembled data.
- Both formats build every label from the same helpers (`requestParts`, `eventParts`, `notes`, `lmParameterSummary`, `lmPrefixText`, `lmBodyStateLabels`), so parity comes from sharing them. `lmPartsMarkdown` and `lmPartsHTML` only differ in how recorded values are quoted.
- Tool pairs render at the call, not the result (Q28): `formatToolUse`/`formatToolUseHTML` defer generic tools, and the LM Studio renderer then writes the combined block immediately and marks the call `paired`, so the result site skips it. HTML uses `toolResultBlockHTML`, the section-free part of `formatToolResultHTML`; the wrapped form would open a new Assistant section and detach the call from its "supplied context" heading. `BuildToolMeta` is prebuilt over all entries so a result whose call sits in a repeated prefix still finds it.
- Tool-result-only entries get no heading. Assistant text and thinking go through `markdownToHTML` in HTML; user, system and tool text is escaped in `<p>`, as in the existing renderers.
- `lmMarkdownSource` backslash-escapes `<Truncated in logs>` in text read as Markdown. Parameter JSON and log text are in code blocks and stay as logged.
- Goldens are `testdata/lmstudio/{threads,tools,classification,recovery}.{md,html,fragment.html}.golden`; regenerate with `go test ./internal/transcript -run TestLMStudioRender_Goldens -update`. The parity test checks boundary order, shown text and marker counts across Markdown and fragment HTML, and that standalone HTML contains the fragment.

## Time Zone (`lmstudio_timezone.go`)

- `zoneFromEnvironment()` reads `TZ` with `os.LookupEnv` and calls `resolveZone(tz, set)`: unset → `time.Local` (label `host local zone`), empty → UTC, otherwise an optional leading `:` then an absolute path (`filepath.IsAbs`, so drive/UNC on Windows) via `os.ReadFile` + `time.LoadLocationFromTZData`, or an IANA name via `time.LoadLocation`. Failure gives `loc == nil` plus a warning; nothing substitutes another zone.
- `:` alone and `Local` are rejected explicitly, because `time.LoadLocation` silently maps `""` to UTC and `"Local"` to the host zone.
- `zoneResult.interpret(wall)` returns an offset-bearing instant only when the wall time maps to exactly one instant. It tries the offsets in force 24 hours either side of and at the wall time; nonexistent (gap) and ambiguous (repeated hour) times, invalid dates and a failed zone all return false.
- The package blank-imports `time/tzdata` (about 450 KB in every binary that links `transcript`, including `orbit`). The embedded data is only a fallback: Go consults `ZONEINFO` and the system zone directories first. The no-system-database path cannot be tested in-process because those directories cannot be disabled; `go list -deps ./cmd/apsis` shows the import is linked.
- Tests build a fixed-offset version-1 TZif file in-process (`fixedTZif`), so the path form is tested without platform zone files.

## Current Integration Points

- `transcript.DetectFormat` reads at most 65,536 bytes and returns the consumed prefix for replay. Detection and full record parsing are separate concerns; a startup-heavy log can require forced parsing.
- CLI raw JSON uses `convertToJSON`, which bypasses normalized transcript rendering. It treats non-Kiro input as JSONL, so a future LM Studio rejection must cover that path separately.
- Follow mode resolves through `resolveFollowInput` or `runLatest` before starting the Claude-shaped follower. Checking only the normal conversion path cannot reject every unsupported source/mode combination.
- Source registration spans `AllSources`, `IsValidSource`, CLI resolution order, the lister and resolver. A listed logical LM Studio session can span files and select only part of each; opening one backing file is insufficient.
- `apsisweb.handleTranscript` currently parses a resolved reader and renders entries without passing parse warnings into its view data. The LM Studio proposal requires visible warnings in this path.
- The web transcript size guard uses `resolved.Metadata.Size`. The LM Studio proposal counts selected session-owned records, rather than every byte in files shared with other sessions.
- Request snapshots can repeat source tool IDs. Existing renderers match IDs across the whole transcript; see [Transcript Metadata Flow](transcript-metadata.md). A future LM Studio design must preserve request-local linking and original source IDs separately.
- No request-scoped correlation field is established for the proposed disk grammar. The requirements keep all response/diagnostic attribution unknown; a unique thread or response ID in an excerpt is insufficient evidence of request ownership.

## Policy Boundaries

- A direct file/stdin view and a listed logical-session view have different scopes. Keep source order, logical membership and unknown attribution explicit rather than treating a containing file as one conversation.
- Direct log evidence is inspected through CLI conversion. Proposed browser session pages show body-free references and CLI instructions, with no new file-serving or upload route.
- Proposed request parameters remain logged evidence outside conversation messages; selective credential redaction is not implied. Discovery help and its unanswered opt-in choice disclose those values alongside prompts.
- Enabled LM Studio discovery participates in the sibling's inclusive all-projects `latest`; the independent review's source-filter workaround was rejected because it changes that contract. Singleton volume and unsupported follow/raw-JSON outcomes remain explicit policy consequences.
- Client prompt text is not authoritative project metadata. Correlation identifiers do not by themselves establish project ownership or request-response attribution.
- Headless review can resolve routine technical gaps and record local admission; it cannot manufacture approval for unanswered discovery, grouping, attribution or presentation choices.
