# Decision Log: Apsis — LM Studio Server Logs

## Quick Decisions

| ID | Date | Decision | Rationale |
|----|------|----------|-----------|
| Q1 | 2026-11-05 | Use source, forced format and provider `lmstudio`, without a local/cloud flag | Names the mechanism while preserving the sibling's accepted provider-only policy |
| Q2 | 2026-11-05 | Limit specialised translation to seven exact inference endpoints ([2.1](requirements.md#2.1)); treat `/v1/decisions` as unsupported and aggregate warnings for other unknown POST paths | Public LM Studio documentation confirms both v0 completion endpoints; `/v1/decisions` is a distinct typed-question API, not this slice's conversation translation |
| Q3 | 2026-11-05 | Use the existing 65,536-byte detection bound, require an included request header, and retain forced parsing | A generic log header does not distinguish LM Studio; startup-heavy excerpts remain explicitly convertible |
| Q4 | 2026-11-05 | Reject raw JSON and follow mode for LM Studio, directing raw JSON users to shared JSON | The text disk log is neither the existing raw JSONL format nor the follow parser's line format |
| Q5 | 2026-11-05 | Retain physical line ranges, partial/oversized states and body-free warnings in CLI, shared JSON and web | Recovery must preserve later requests without presenting unreadable bodies as complete or leaking content through diagnostics |
| Q6 | 2026-11-05 | Never concatenate an incomplete record across files | Rotation carries no continuation grammar; unknown bytes must not become another request's body |
| Q7 | 2026-11-05 | Count only session-owned records toward logical-session size | The existing browser limit must not count other sessions merely sharing the log file |
| Q8 | 2026-11-05 | Synthetic fixtures only, with separate provenance for log grammar and public payload documentation | The repository must contain no private log content, identifiers, paths or served model names |
| Q9 | 2026-11-04 | Scope tool links and rendering groups to request occurrences or individual logged-response events, preserving original identifiers for prefix equality | Global ID lookup can overwrite results across clients and repeated snapshots; requirements 2.8/8.3 make isolation observable |
| Q10 | 2026-11-04 | Retain WARN/ERROR before applying routine-category filtering, including excluded-endpoint diagnostics | Retention and filtering clauses otherwise conflict; all other routine levels can still be ignored when they carry no included content |
| Q11 | 2026-11-04 | Map native chat's string input and typed input items to supplied user context, retaining `system_prompt` separately | Public native-chat documentation establishes both forms; endpoint coverage must test both rather than treating the string as an unsupported body |
| Q12 | 2026-11-05 | Retain complete client metadata, parent-thread identifiers and fully recorded request kind; truncated blobs stay raw/marked | Pinned Codex `responses_metadata.rs` establishes field provenance and side-request sharing; thread membership is not proof of a dialogue turn or response ownership |
| Q13 | 2026-11-05 | A valid LM Studio header signature wins over other formats' body-marker heuristics; missing bounded signatures suggest forced parsing, while discovered matching files still parse as logs | Kiro marker collisions and notice-only/late-request files are possible; this clarifies input outcomes without broadening automatic recognition |
| Q14 | 2026-11-05 | Browser session commands include all-projects scope/source, while direct-log commands identify the file and forced format | Oversized-session instructions must not suggest a bare identifier that cannot reopen this source |
| Q15 | 2026-11-05 | Direct-file request/membership IDs match discovery inside the configured root; outer log-document and out-of-root/stdin IDs remain input-scoped | Stable membership does not turn a containing multi-session file into one listed conversation, or require reading other log bodies |
| Q16 | 2026-11-05 | Explain common unknown response attribution once per human transcript while keeping per-request shared-JSON state | The state is universal for this grammar; repeating an absence-sounding sentence on every request adds noise and can misdescribe existing unattributed evidence |
| Q17 | 2026-11-09 | Blank-import `time/tzdata` for bundled IANA zone names; use `time.LoadLocationFromTZData` for an explicit absolute TZif-file override; distinguish unset from explicitly-empty `TZ` with `os.LookupEnv`, not `os.Getenv` | Decision 10 left exact API integration to design; these are the standard-library calls that satisfy its no-system-zone-database and unset-vs-empty requirements |
| Q18 | 2026-11-09 | Logical-session and request identifiers hash the full original value (thread ID text, or file-relative path plus start line) rather than truncating it | Removes the truncation-collision surface `codex-exec`'s path-mixed identifier has, so no identifier-conflict warning code is needed for this source |
| Q19 | 2026-11-09 | Listing's `request_count` is a new field, not a reuse of the existing byte-sized `size_bytes`/`Size` column | Requirement 7.2 calls it "size" but means a count for this source; overloading the byte-sized field would mislabel it in the shared-JSON schema and in `apsis -l`'s `FormatSize` output |
| Q20 | 2026-11-09 | Amended requirement 7.3 to scope its zero-log-body-read guarantee to Unix and macOS; other platforms stay correct without that guarantee | Decision 6 implements the cache fingerprint on Unix only; the requirement text should say so rather than promising a platform-agnostic guarantee the design doesn't build |
| Q21 | 2026-11-09 | The active, currently-growing log file is fully re-parsed on every refresh in which it changed; no incremental/resume parsing is built for this feature | 7.3's guarantee applies to an *unchanged* tree; the active file is expected to change on most polls, so re-parsing it fully doesn't violate 7.3, and resumption state would add real parser complexity for a file the source's own date/.N rotation keeps from growing unboundedly within one day |

Decisions 1–5, 7, 8 and 10 are **accepted**. Decisions 6 and 9 are **accepted** as of design. The repository has no `.agentic.json`; superseded decisions are not kept as separate historical entries — an earlier decision's rejected or replaced approach is folded into the current decision's Alternatives Considered instead.

---

## Decision 1: Group Only Explicit Client Threads

**Date**: 2026-11-05
**Status**: accepted

### Context

Reported Codex request bodies contain `client_metadata.thread_id`, while other requests lack identifiers. Context can shrink, unrelated clients can send the same prefixes, and truncated strings can make different original inputs appear equal.

### Decision

Group exact nonempty string thread identifiers within one configured root, including across files, models and endpoints. Keep requests without that identifier as singleton sessions, and preserve other identifiers only as evidence.

### Rationale

Recorded client identity survives compaction without treating text similarity as proof of a shared conversation. Stable identifiers must derive from recorded identity and stable file locations, not traversal order or the earliest currently retained request.

### Alternatives Considered

- **Prefix, time or model grouping:** rejected because it can merge independent clients, particularly after log truncation.
- **Cache keys, session fields or response-chain grouping:** rejected because the inspected reports do not establish general conversation-identity semantics or complete response links.

### Consequences

**Positive:**

- Thread sessions survive context changes and rotation.
- Adding older files or deleting other files does not rename surviving sessions.

**Negative:**

- Unidentified multi-turn clients produce one listing per request.
- Additional client identity conventions need separately justified support.

---

## Decision 2: Keep Request Snapshots and History References

**Date**: 2026-11-05
**Status**: accepted

### Context

Repeated history can make a log appear to contain the same conversation many times. Global deduplication or suppressing assistant context equal to a logged response can also erase intentional repetitions or obscure provenance.

### Decision

Retain request boundaries and complete parsed input sequences in shared JSON. In human-readable views, reference only an identical leading sequence of logged items from the previous request in the same session, and mark changed or shortened history as a context discontinuity.

### Rationale

The reader can distinguish resent context from recorded generation while reconstructing any request. Truncated text equality means only that logged representations match; it establishes neither original content equality nor identity.

### Alternatives Considered

- **Render every snapshot as ordinary new turns:** rejected because replayed context looks like new conversation activity.
- **Globally deduplicate messages or suppress equal responses:** rejected because equal text at a different position or with different provenance is separate evidence.

### Consequences

**Positive:**

- Routine history repetition is reduced without losing request reconstruction.
- Compaction and repeated requests remain visible.

**Negative:**

- Renderers need request boundaries and history references.
- Human-readable requests can depend on earlier sections for their repeated prefix.

---

## Decision 3: Gate Standard Discovery Behind All Projects

**Date**: 2026-11-05
**Status**: accepted

### Context

LM Studio has a reported standard server-log directory, and logs may include full prompts from every server client. Existing single-project listings should not parse unrelated server traffic.

### Decision

Discover the standard root only in explicit all-projects scope, with CLI/environment override and empty-value disable. Gate identifier opening with the same scope, keep listings metadata-only, and disclose server-wide traffic in help and the existing network warning.

### Rationale

This keeps standard-location discovery consistent with existing sources while making the expanded listing scope deliberate. Root confinement applies before candidate reads; explicit file conversion remains an independent user choice.

### Alternatives Considered

- **Require a configured root for all discovery:** rejected because it adds setup despite a standard location.
- **Read logs in ordinary project listings:** rejected because the source records no project and this would add unrelated traffic and parsing work.
- **Refuse non-loopback service:** rejected because it would override an explicitly selected bind policy beyond this parser's scope.

### Consequences

**Positive:**

- Default project listings retain their existing behaviour and avoid log-body reads.
- All-projects users can discover sessions without an extra root setting.

**Negative:**

- All-projects users may discover traffic they did not expect.
- The existing `0.0.0.0` warning does not cover every possible externally reachable bind; general bind-policy changes remain outside this feature.

---

## Decision 4: Preserve Correlation Without Borrowing Projects

**Date**: 2026-11-05
**Status**: accepted

### Context

Reported thread identifiers permit consumers to compare server and client evidence. A request's prompt can quote a working directory, and several saved harness artifacts can share the same thread identifier.

### Decision

Expose the original thread identifier in transcript and listing metadata, keep server and harness sessions separately listed, and leave LM Studio's project unknown. Automatic web links and borrowed project attribution are out of scope.

### Rationale

Correlation metadata does not establish an authoritative server-side project. Leaving linking separate also avoids an implicit selection rule between copied streams, resumed streams and rollouts.

### Alternatives Considered

- **Extract projects from environment-context prompts:** rejected because those are client content and may be truncated or quoted examples.
- **Borrow a matching harness session's project:** rejected because correlation and project ownership are different assertions and multiple matches need a policy.
- **Merge or suppress the server transcript:** rejected because server-received context and client events are distinct evidence.

### Consequences

**Positive:**

- Consumers retain exact recorded correlation evidence.
- Project-scoped behaviour remains independent of client-specific prompt parsing.

**Negative:**

- Users cannot navigate to matching harness pages automatically in the baseline.
- Server work remains outside project-specific listings even when a likely client record is available.

---

## Decision 5: Keep One Direct Log Document Shape

**Date**: 2026-11-05
**Status**: accepted

### Context

One disk file can contain several interleaved logical sessions and unattributed events. The approved sibling requires one shared session-document structure for all sources.

### Decision

Convert a supplied file or stdin into one source-ordered log transcript using the common outer document, with logical-session membership on requests and log-level events where applicable. Identifier-based opening selects a logical session.

### Rationale

Direct inspection preserves physical evidence and handles stdin without another lookup. Multiple logical sessions do not require an input-dependent object/array switch or a second LM Studio JSON contract.

### Alternatives Considered

- **Emit an array for files and one object for identifiers:** rejected because it violates the common outer-shape contract.
- **Require a session identifier for direct input:** rejected because excerpts and stdin must remain inspectable before discovery.
- **Treat every request as the same conversation:** rejected because input containment does not establish session identity.
- **Group into conversation sections instead of physical log order:** rejected because it changes direct-view ordering away from the common outer session-document structure.

### Consequences

**Positive:**

- Consumers handle one outer session shape for every input.
- Unattributed events have a truthful home in direct views.

**Negative:**

- Direct views can interleave logical sessions.
- Additive schema fields must distinguish log-transcript identity from logical-session membership.

---

## Decision 6: Cache Unchanged Logs Within the Process, Scoped to Unix

**Date**: 2026-11-09
**Status**: accepted

### Context

Grouping needs fields inside request bodies, and the existing web list refreshes every 15 seconds. An earlier cold-time target relied on an unverified single-machine measurement and a mistaken monthly-volume claim, so it cannot serve as a portable pass/fail threshold. `apsis-shared-json` hit the same zero-body-read problem for its own cross-source cache and spun the rigorous fix — a native file-identity fingerprint (inode/ctime on Unix, file-ID on Windows) — into its own gated PR phase, which has sat unimplemented for lack of a native Windows host to verify against.

### Decision

Cache a lightweight per-file summary (thread groupings, request counts, first-known model/time, warnings — never full bodies) keyed on a change fingerprint, with two build-tagged implementations: a real fingerprint (size, mtime, ctime, inode, device via `syscall.Stat_t`) on `unix`, and an always-miss stub everywhere else. Reflect added, changed, replaced, truncated or removed inputs on the next successful refresh; write no persistent index.

### Rationale

The user chose to descope native Windows fingerprinting for this feature rather than repeat the sibling's unresolved verification blocker. The Unix fingerprint is not itself blocked — unlike Windows, Unix/macOS development hardware is available, so building and testing it carries none of the sibling's stalled risk. An always-miss cache on other platforms keeps correctness identical everywhere (no risk of a weak heuristic serving stale data); it only forgoes the performance optimisation there, which the requirement's wording ("within one apsis serve process") does not make a correctness requirement.

### Alternatives Considered

- **Size+mtime heuristic on every platform:** rejected because it reintroduces exactly the false-cache-hit risk the sibling spec's Decision 23 already rejected for the same problem, for no platform-coverage benefit over the always-miss stub.
- **Full native fingerprint including Windows:** rejected because it repeats the sibling's unverified native-Windows blocker and the user explicitly asked to descope Windows for simplicity.
- **Persist an index on disk:** rejected because it introduces stored identifiers and a cache-format lifecycle without demonstrated need.
- **Require a fixed time budget for a fixed byte volume everywhere:** rejected because hardware, record count and payload shape affect runtime and no reference environment was agreed.

### Consequences

**Positive:**

- Repeated web refreshes on Unix/macOS avoid unchanged body reads, with no weaker-heuristic risk anywhere.
- No additional persistent index, and no dependency on the sibling spec's unmerged generic cache package.

**Negative:**

- Windows `apsis serve` re-reads the configured root's bodies on every refresh; large unthreaded roots pay that cost repeatedly on that platform.
- Each new CLI process still pays cold parsing cost, matching every other source's existing per-invocation behaviour.

---

## Decision 7: Keep Unsupported Request Attribution Unknown and Inspect Full Logs Through the CLI

**Date**: 2026-11-04
**Status**: accepted

### Context

The supported disk grammar has no established request-scoped correlation field. Thread identifiers name conversations; response identifiers and previous-response references name context points. Neither establishes ownership of a newly logged response or diagnostic. No existing browser route exists for the direct-file evidence the parity tests promise. Model-tagged notices and responses can interleave, and requests can outlive a rotated file; adjacent validation warnings establish a pattern, not a guarantee that a later record belongs to the immediately preceding request.

### Decision

Keep all logged response and WARN/ERROR request attribution unknown for this grammar. Translate response content as log-level events in direct CLI views. Session views retain only body-free counts, source locations and CLI inspection instructions, without suggesting candidate ownership. Direct file/stdin conversion supports Markdown, standalone HTML and shared JSON; web parity applies to logical-session views.

### Rationale

An empty set of evidenced correlation fields cannot justify a positive matcher. Explicit CLI inspection preserves full evidence without adding a browser route that opens unrelated clients' log content; no response should become a claimed answer or failure for the wrong request. Any later specialised matcher needs independently established field semantics and collision rules; synthetic fields alone are insufficient evidence.

### Alternatives Considered

- **Match a thread or response identifier that is unique in the current excerpt:** rejected because uniqueness in a partial input does not make a conversation/context identifier request-scoped.
- **Infer ownership from position, model tags, apparent serial execution, or adjacency/active-request counters:** rejected because concurrency, missing completion lines and rotation provide counterexamples even when sampled logs look serial.
- **Copy unmatched bodies into every file-sharing session:** rejected because it exposes unrelated content and multiplies output.
- **Drop unmatched records:** rejected because errors and responses would disappear without disclosure.
- **Add a root-confined browser file viewer:** rejected because it adds routing and full-file exposure beyond existing session views; CLI instructions provide an explicit inspection path.
- **Leave file references without an inspection contract:** rejected because browser readers would not know how to reach the retained evidence.

### Consequences

**Positive:**

- No undocumented matching field becomes invented request ownership.
- Response content remains inspectable in all supported direct-output formats; direct log inspection retains unattributed evidence across every renderer.
- Session pages disclose the CLI access needed for full log inspection.

**Negative:**

- Requests have no individually attributed responses or server diagnostics in this grammar; most reported response and warning records may remain unattributed.
- Browser-only readers cannot inspect full unattributed bodies without CLI access to the log host, and a session reader must inspect referenced files for full diagnostics, including later rotated files.

---

## Decision 8: Keep All-Projects Latest Inclusive and Make Its Consequences Visible

**Date**: 2026-11-05
**Status**: accepted

### Context

Enabled LM Studio discovery can add many recent requests to an all-projects list. Its selected session may be newer than the client transcript, and follow/raw-JSON modes do not support this source.

### Decision

Retain the approved sibling's inclusive all-projects latest ordering. Explain that enabled LM Studio sessions participate, report the selected source on an unsupported-mode error, and offer discovery disable or explicit opening of another session. Do not silently fall back to an older session or introduce source-filtered latest here.

### Rationale

This spec must build on the sibling's approved scope and cannot silently change what latest means. Explicit discovery controls provide the existing way to exclude LM Studio without another selection rule.

### Alternatives Considered

- **Exclude LM Studio from latest unless --source is supplied:** rejected because it changes inclusive all-projects ordering and the sibling design currently rejects --source with latest.
- **Silently select the newest followable source:** rejected because a mode flag would change session selection without disclosing the skipped newest evidence.
- **Change the discovery recommendation automatically:** rejected because discovery policy must stay an explicit, deliberate choice, not an automatically inferred default.

### Consequences

**Positive:**

- Listing and latest use the same approved ordering.
- Errors explain the source and a usable remedy.

**Negative:**

- Default all-projects discovery can make latest select server traffic frequently.
- Users may need to disable LM Studio discovery or open another session explicitly before following.

---

## Decision 9: Preserve Request Parameters as Evidence Outside Conversation Messages

**Date**: 2026-11-05
**Status**: accepted

### Context

Conversation extraction alone loses tools, stream/store settings, client metadata and integration options that explain what the server received. Directing raw-JSON users to shared JSON would otherwise hide those parsed fields. Some logged parameter values, including integration headers, can be sensitive.

### Decision

Keep remaining parsed top-level request fields as parameters in shared JSON and labelled non-conversation details in Markdown/HTML, with stream/tool-name summaries where available. Preserve values and truncation markers without vendor-specific redaction. Disclose parameter/header exposure with the existing server-log help; use synthetic fixtures only.

### Rationale

A translator should preserve logged evidence and distinguish configuration from conversation turns. Ad-hoc field redaction would silently change source content and requires a separately defined policy; it is not inferred from the source format.

### Alternatives Considered

- **Discard fields outside conversation content:** rejected because stream and tool settings explain missing output and other request behaviour.
- **Redact selected vendor header keys automatically:** rejected because arbitrary client parameters can contain equally sensitive values, and selective removal creates unrecorded content changes.
- **Render the complete body as new conversation messages:** rejected because definitions/settings are not dialogue turns and would defeat context-prefix presentation.

### Consequences

**Positive:**

- Shared JSON remains useful for inspecting otherwise untranslated request evidence.
- All renderers expose the same parameters without inventing conversation roles.

**Negative:**

- Parameter details can be large and retain sensitive logged values.
- Discovery is enabled by default, so these parameters — potentially including credential-bearing integration headers — appear in all-projects listings and `apsis serve` web HTML by default; source fixtures cannot use real values.

---

## Decision 10: Interpret Explicit Time Zones Portably Without Requiring System Zone Data

**Date**: 2026-11-05
**Status**: accepted

### Context

The reported disk grammar carries local wall time without a zone, and the project releases Windows binaries whose host-local Go zone handling does not establish Unix TZ semantics. Standard IANA lookup also cannot depend on a separately installed zone database. Go's Unix local-zone initialization can silently return UTC after a failed `TZ` load, so a usable zone object does not prove that an explicit override succeeded.

### Decision

Preserve wall time and source order. Use the host zone when `TZ` is unset and UTC when it is explicitly empty; emit an offset-bearing instant only when that interpretation is unique, leaving invalid, ambiguous or nonexistent instants (and corresponding creation times) absent. Require explicit TZ interpretation on every supported platform, with valid IANA zones available without system data: accept UTC, loadable IANA names and absolute TZif-file paths with an optional leading colon. An unsupported or unloadable nonempty override produces a visible warning and wall-time-only output.

### Rationale

This supports local inspection using the existing process setting while avoiding fabricated precision; file and line order remain reproducible regardless of timestamp interpretation. Validation prevents a failed override from acquiring a plausible but unintended offset, and the same explicit setting must not silently acquire a different interpretation on another release platform. Go's [tzdata package](https://pkg.go.dev/time/tzdata) provides a standard-library path to bundling IANA data; exact integration and platform tests belong to design.

### Alternatives Considered

- **Use each platform's time.Local (or Go's default Unix local-zone) behaviour directly, without validating the override:** rejected because it does not consistently honour the specified override, and a failed load can silently fall back to UTC, masking invalid configuration.
- **Silently assume UTC, or infer an offset from monotonicity/daylight-saving patterns:** rejected because the source does not establish the instant, and a log excerpt or missing records can contain only the second occurrence of a repeated hour.
- **Abort the whole log conversion on a zone error:** rejected because recorded content and physical ordering remain valid evidence.
- **Require users to install zone files separately, or add a new dedicated zone flag:** rejected because standard IANA settings should work with the distributed binary, and the existing `TZ` environment variable already provides a process-scoped override.
- **Substitute UTC on load failure:** rejected because it presents failed interpretation as a successful assumption.
- **Require an explicit zone setting before interpreting any wall time:** rejected because `TZ` already provides a usable default (host-local when unset, UTC when explicitly empty) for the common case, without forcing every user to configure a zone.

### Consequences

**Positive:**

- Wall-time evidence is preserved and the conversion assumption is visible; repeated hours do not reorder requests or create invented timestamps.
- Valid explicit IANA settings work across release platforms without extra setup, and invalid explicit settings cannot masquerade as successfully interpreted instants.

**Negative:**

- Copied logs require a correct zone override for useful interpreted times; some sessions lack an absolute creation time and sort under the shared unknown-time policy.
- Bundled zone information adds binary data and needs a defined update path; native path validation and platform-specific tests are required before implementation can claim portability.
