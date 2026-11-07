# Decision Log: Apsis — LM Studio Server Logs

## Quick Decisions

| ID | Date | Decision | Rationale |
|----|------|----------|-----------|
| Q1 | 2026-11-05 | Source label and forced format are `lmstudio` (`-a lmstudio`) | Matches the provider ID Codex uses for LM Studio, so the source and the provider read the same |
| Q2 | 2026-11-05 | Inference paths are matched exactly: `/v1/chat/completions`, `/v1/completions`, `/v1/responses`, `/v1/messages`, `/api/v0/chat/completions`, `/api/v0/completions`, `/api/v1/chat` | All but `/v1/messages` and `/api/v0/completions` appear in captures, and LM Studio documents those two. Exact matching excludes `/v1/responses/input_tokens`, which appears in captures, and `/v1/messages/count_tokens` |
| Q3 | 2026-11-05 | Only WARN and ERROR entries and response bodies render; `Finished streaming response` only ends a request; other INFO and DEBUG entries are ignored silently | Progress, prompt-cache and engine lines make up most header lines and carry no conversation content |
| Q4 | 2026-11-05 | An entry is a header line plus the lines up to the next header line; errors cite the log file name and the entry's header line number | Bodies span many physical lines, so the header line is the position a reader can find in the file |
| Q5 | 2026-11-05 | A body that does not parse is reported as unreadable, never joined with the start of the next file | In all 7 size rotations inspected, the next file began with a header line, so LM Studio writes each entry whole into one file |
| Q6 | 2026-11-05 | WARN and ERROR entries directly after a request belong to it by position | In the inspected logs, request-acceptance warnings (`The provided unsupported field(s)`, `Developer role detected`, `Ignoring unsupported tool type(s)`) always directly followed their request |
| Q7 | 2026-11-05 | How a line's tag is matched to a request's `model` is settled in design against captures | Tags usually equal the request's `model`, but some name a loaded model instance whose name differs |
| Q8 | 2026-11-05 | Session size is the bytes of the session's own entries | Listings show a size and `apsis serve` refuses sessions over 50 MB; the size of a log file counts other sessions |
| Q9 | 2026-11-05 | Token usage from logged response bodies is shown | Chat-completion response bodies carry `usage`, and the shared JSON already has a token-usage field |
| Q10 | 2026-11-05 | Session identifiers never equal a Codex identifier | `apsis <id>` tries sources in order and would open the Codex session for an identifier both share |
| Q11 | 2026-11-05 | Raw `-f json` and follow mode (`-F`) exit with an error for LM Studio input | Raw JSON reads JSONL and follow mode parses Claude-shaped lines; neither fits a free-text log, and no consumer needs them |
| Q12 | 2026-11-05 | Fixtures are synthetic files with the structure of LM Studio 0.4.26 logs; endpoints absent from captures use documented request shapes, labelled as such | The repository is public and real logs hold prompts, identifiers and model names from one machine |

---

## Decision 1: Group by Codex Thread ID, Otherwise by Conversation Continuation

**Date**: 2026-11-05
**Status**: proposed

### Context

The log has no session or request identifier of its own. Codex `/v1/responses` bodies carry `client_metadata.thread_id`, which equals the `thread_id` of the matching `codex exec --json` stream and the ID of an interactive Codex rollout; `session_id` and `prompt_cache_key` hold the same value. Other clients send no identifier. A multi-turn client repeats the whole conversation in every request, so its requests chain by prefix. Codex compacts context, so its inputs sometimes shrink: across the inspected logs, 21 of 78 threads had a request whose input did not extend the previous one.

### Decision

Group requests that carry a thread ID by that ID. Add any other request to an earlier session without a thread ID when its conversation begins with that session's latest conversation, is longer, uses the same endpoint and model, and exactly one session qualifies. Otherwise start a new session.

### Rationale

The thread ID is the client's own identifier, so Codex sessions group exactly, including through compaction, and link to the Codex session they served. For other clients, extending the previous conversation is what a multi-turn exchange looks like in the log, and the "exactly one" condition means apsis splits rather than merges when the log is ambiguous. In the inspected logs this rule joined 1,263 of 5,334 unidentified requests into 203 multi-request sessions; nearly all the rest were single-shot calls with a system and a user message.

### Alternatives Considered

- **One session per request**: Simplest rule - Rejected: a 50-turn conversation would list as 50 sessions, each repeating the history.
- **Group requests within a time window**: Catches pauses - Rejected: merges unrelated clients that use the same model at the same time, and splits conversations paused longer than the window.
- **Continuation for Codex requests too**: One rule for all - Rejected: compaction would split about a quarter of Codex threads.
- **`prompt_cache_key` as a key for every client**: More clients grouped - Rejected: only Codex's use of it is verified; another client could reuse one key across conversations.

### Consequences

**Positive:**
- Codex sessions group exactly and can link to the Codex session they served.
- The rule depends only on earlier requests, so identifiers stay stable as the log grows.

**Negative:**
- Single-shot traffic, such as batch calls, lists one session per request.
- Two different long strings with the same first and last 50 characters compare equal after LM Studio truncates them, which could chain unrelated requests.
- A client that changes model mid-conversation is split into two sessions.

---

## Decision 2: Attach Server Lines Only When One Request Fits

**Date**: 2026-11-05
**Status**: proposed

### Context

LM Studio runs up to 4 requests per model in parallel, so lines from concurrent requests interleave. Follow-up lines carry the model in their tag but no request identifier. Request-acceptance warnings are written directly after their request (Q6), but response bodies and later errors can appear after other requests have arrived. Completion is not always logged: streamed responses end with `Finished streaming response`, and client disconnects carry no model tag.

### Decision

Attach acceptance warnings by position. Attach any other follow-up line only when exactly one in-progress request fits it. When several requests fit and all are in one session, attach the line to the session, marked as not tied to one request. When they span sessions, attach it to none and show each candidate session how many such lines may belong to it. End a request at an attached end line, its session's next request, the end of its file, or when its model's end lines catch up with its received requests.

### Rationale

A transcript that names the wrong request for an error or response is worse than one that says it cannot tell. The rule attaches the most common lines, Codex's acceptance warnings, exactly, and keeps every other attachment provable from the log. The count-based end condition keeps a burst of parallel requests from leaving every later line ambiguous until the end of the file.

### Alternatives Considered

- **Assign by arrival order**: Every line gets a request - Rejected: parallel slots finish out of order, and a wrong attribution looks as reliable as a right one.
- **Copy ambiguous lines into every candidate session**: Nothing hidden - Rejected: under sustained batch traffic every response would be copied into hundreds or thousands of sessions.
- **Drop ambiguous lines silently**: Simplest - Rejected: the reader would not know that information exists.

### Consequences

**Positive:**
- Apsis never attaches a line to a request the log does not single out.
- Codex's acceptance warnings, the most frequent warnings, attach exactly.

**Negative:**
- Responses to parallel single-shot calls are often only counted, not shown.
- The in-progress definition is more involved to explain and to test than an order-based rule.

---

## Decision 3: Read Timestamps in Apsis's Time Zone and Order by File and Line

**Date**: 2026-11-05
**Status**: proposed

### Context

Timestamps are written as `YYYY-MM-DD HH:MM:SS` in local time without a zone, and LM Studio starts a new file at local midnight. The shared JSON output needs instants, and the hour repeated when daylight saving ends makes timestamps ambiguous and non-monotonic.

### Decision

Read timestamps in the time zone of the apsis process, which `TZ` overrides, and emit an explicit UTC offset. Order everything by file date, file number and line, never by timestamp. In a repeated hour, take the earlier instant unless that would go backwards within the file; for a local time that does not exist, use the offset in effect before the change.

### Rationale

Apsis usually runs on the machine that ran LM Studio, so its zone is the log's zone. `TZ` already lets a user read a log from another zone, so no new setting is needed. File and line order is what LM Studio wrote and needs no interpretation.

### Alternatives Considered

- **Assume UTC**: No local-time handling - Rejected: wrong by the local offset for nearly every user.
- **A dedicated time-zone setting**: Explicit - Rejected: `TZ` already does this for one run.
- **Read the zone from LM Studio's settings**: Authoritative - Rejected: LM Studio does not record it.
- **Order by timestamp**: Matches other sources - Rejected: the repeated hour would reorder entries.

### Consequences

**Positive:**
- Instants are correct when apsis runs where LM Studio ran, without configuration.
- Order stays correct through daylight-saving changes.

**Negative:**
- Logs copied from a machine in another zone get wrong instants unless `TZ` is set.
- A file that holds only the second occurrence of a repeated hour is read one hour early.

---

## Decision 4: List LM Studio Sessions Only Across Projects, With the Linked Codex Project

**Date**: 2026-11-05
**Status**: proposed

### Context

The log records no working directory. Codex request input contains an `<environment_context>` block with a `<cwd>` element, but with `succinct` file logging LM Studio keeps only the first and last 50 characters of each string, and the path was cut in 847 of 848 Codex requests inspected. The Codex rollout or exec stream that a thread ID points to does record its project.

### Decision

Show LM Studio sessions only in listings across projects. WHERE apsis lists the Codex session with the same thread ID, show that session's project path for the LM Studio session.

### Rationale

Single-project listings, and their speed, stay as they are for every user, including those who never use LM Studio. Projects come from a record that holds them, not from a truncated string. Listing across projects is already opt-in, so the server's record is exposed only when a user asks for it.

### Alternatives Considered

- **Show the session in the linked Codex session's project listing**: Visible where the work happened - Rejected for now: every project listing would read all LM Studio logs; kept as an open question.
- **Read the working directory from `<environment_context>`**: No dependency on Codex records - Rejected: the path is truncated in nearly every request.
- **Match by time overlap with Codex sessions**: Covers clients without a thread ID - Rejected: a guess that fails when several sessions run at once.

### Consequences

**Positive:**
- No change to single-project listings.
- Project paths shown are ones a source recorded.

**Negative:**
- A project's own listing does not show the server's record of its Codex runs.
- The project path appears only while the linked Codex session is still listed.

---

## Decision 5: Link LM Studio Sessions to Codex Sessions by Thread ID

**Date**: 2026-11-05
**Status**: proposed

### Context

The combined draft left correlation with Codex or Claude Code sessions to consumers, because no shared identifier was known. Codex requests do carry one: `client_metadata.thread_id` matched the exec-stream `thread_id` for 12 of 12 requests of one run and the rollout ID for 1 of 1 interactive session.

### Decision

Record the thread ID on the LM Studio session, show it in every output, and link to the `codex` or `codex-exec` session with that ID in `apsis serve`. Keep both sessions listed.

### Rationale

Apsis holds an exact key, so consumers should not each rebuild the match. The two records differ: the server record holds the developer and system messages, tool definitions and truncated strings as sent, while the Codex record holds full content and client events. Keeping both and linking them preserves each.

### Alternatives Considered

- **Leave correlation to consumers**: Smaller scope - Rejected: every consumer would rebuild a match apsis can make exactly.
- **Merge into the Codex session**: One entry per run - Rejected: it would mix truncated server text with full client text in one transcript.
- **List the run once, under `codex`, as for exec streams**: Fewer listing entries - Rejected: the server record holds information the Codex record lacks.

### Consequences

**Positive:**
- Consumers correlate on a field instead of timestamps.
- A reviewer can move from the server's record to the client's.

**Negative:**
- The same work appears twice in listings across projects.
- Markdown shows the thread ID but cannot link to another session.

---

## Decision 6: Cache Parsed Logs in Process, With No Index on Disk

**Date**: 2026-11-05
**Status**: proposed

### Context

Thread IDs and conversations are inside request bodies, so listing must parse whole files. The inspected machine held about 130 MB of logs over nine months, with daily files up to about 10 MiB before rotation. `apsis serve` refreshes its list every 15 seconds. A prototype parse of all 131 MB, including conversation hashing, took 2.4 seconds.

### Decision

Keep parsed results in the apsis process, re-reading a file only when its size or modification time changes. Set a cold target of 3 seconds for 150 MB. Write no index to disk.

### Rationale

`apsis serve` gets cheap refreshes, and the cold target covers today's volumes with margin. Apsis writes no files today; an index on disk would add a file format, invalidation rules and a second copy of identifiers outside LM Studio's directory.

### Alternatives Considered

- **Index on disk**: Fast repeated CLI listings - Rejected for now: adds a cache format and stored identifiers; kept as an open question.
- **List only recent days**: Bounded cost - Rejected: older sessions would disappear from listings without notice.
- **Read header lines only**: Much faster - Rejected: grouping needs the thread ID and conversation from the body.

### Consequences

**Positive:**
- No new files written by apsis.
- `apsis serve` refreshes re-read only the file being written.

**Negative:**
- Each CLI listing across projects pays the full parse.
- Cost grows with log volume until LM Studio's old logs are removed.

---

## Decision 7: Render Every Session in a Log File Given Directly

**Date**: 2026-11-05
**Status**: proposed

### Context

`apsis <path>` and stdin produce one session for other sources, but one log file holds many interleaved sessions, and stdin cannot be reopened by identifier.

### Decision

Group the requests in the given input and render every session in order of first request, each under its own heading; the shared JSON output is an array of session objects.

### Rationale

`cat log | apsis` then works as it does for other formats, and each session keeps the history rule of [3.3](requirements.md#3.3).

### Alternatives Considered

- **Treat the whole input as one session**: One JSON shape - Rejected: interleaved conversations would mix and repeated history could not be removed per conversation.
- **Require a session identifier**: One session per run - Rejected: stdin cannot be selected by identifier.

### Consequences

**Positive:**
- Log files and excerpts convert without listing first.

**Negative:**
- The shared JSON shape depends on the input, and the schema must describe the array form.

---

## Decision 8: Read the Default LM Studio Log Directory Unless Disabled

**Date**: 2026-11-05
**Status**: proposed

### Context

Apsis reads other sources from their standard locations, and exec streams need a configured directory only because they have none. LM Studio logs have a standard location, but with `logSensitiveData` on they hold prompts, and when the server accepts network connections they hold requests from other machines.

### Decision

Read `~/.lmstudio/server-logs` by default, show its sessions only in listings across projects, let an empty setting disable reading, and extend `apsis serve`'s network warning to name LM Studio logs.

### Rationale

The standard location makes the source work without setup, as other sources do. Listing across projects is already an explicit choice, so default reading exposes nothing in the default views.

### Alternatives Considered

- **Opt-in through a configured directory**: No surprise exposure - Rejected for now: inconsistent with other standard-location sources; kept as an open question.
- **Refuse LM Studio sessions on non-loopback binds**: Strongest protection - Rejected: overrides a choice the user made with `--bind`.

### Consequences

**Positive:**
- Works without configuration.

**Negative:**
- Users who never asked for it see LM Studio sessions in listings across projects.
- The network warning grows longer.

---
