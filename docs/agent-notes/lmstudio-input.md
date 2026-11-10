# LM Studio Input Integration (Apsis)

## Current State

- `specs/apsis-lmstudio-logs/` defines the server-log source. Phase 1 (shared representation, time zone) is implemented; no record parser, detection, source registration or discovery exists yet.
- The sibling `apsis-shared-json` supplies the common JSON contract and all-projects scope. Its approved requirements remain authoritative; the LM Studio requirements still have unanswered user-owned choices.
- Public LM Studio API documentation defines endpoint payloads, not the disk-log grammar. Earlier structural inspections are labelled as reported observations in the handoff. Fixtures must be synthetic and distinguish those two kinds of evidence.

## Shared Representation (`internal/transcript`)

- `FormatLMStudio`, `Entry.RequestID`/`SuppliedContext`/`Unattributed`/`SourceLine` (all `json:"-"`) and `ParseResult.LMStudioRequests` exist. `LMStudioRequest` lives in `lmstudio_types.go` with the `BodyState*` constants (`not_logged`, `malformed`, `incomplete`, `oversized`; empty = translated).
- `LMStudioRequests` is parallel to `Entries`, not embedded: entries join their request through `Entry.RequestID`. A request with no translatable body has a request row and zero entries.

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
