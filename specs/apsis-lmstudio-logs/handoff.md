# Handoff: apsis-lmstudio-logs

Context for whoever works on this spec next. Delete this file before opening the upstream PR. The process and current state are in `SPEC-DRIVE.md` at the worktree root and in `drafts/status.md`.

## What this spec is

Adds LM Studio server logs as an apsis input. It is a sibling of `specs/apsis-shared-json/`, split out of it on 2026-11-05 because the log is undocumented free text with its own grouping, time-zone, performance and sensitive-data questions. It depends on that spec for:

- the shared JSON output (`-f` value with an integer schema version), which this spec extends with added fields only (no breaking change);
- the all-projects listing option for `apsis --list` and `apsis serve`;
- the provider field (LM Studio sessions report provider `lmstudio`). Apsis does not mark sessions local or cloud; consumers decide.

Status: requirements merge recovered from the stale 15:00Z claim and revised after same-model fresh-context reviews plus the independent Claude Code/Opus peer review at 18:32Z. Both independent original drafts, their cross-reviews, all three populated unlogged outputs and the 44-criterion reviewed revision are preserved in `drafts/requirements/`. The current proposal has 47 criteria and 13 decisions; Decision 13 supersedes 10, while 9/10 retain their links to 3/4. `drafts/requirements/questions.md` still holds eight unanswered user choices, now exposing listing-volume, latest and parameter-retention consequences. No LM Studio requirements approval is recorded. See `drafts/status.md` for current ownership and verification.

## Direction from the user

- Apsis translates harness formats into one shared representation, rendered as Markdown, HTML and JSON. New sources and fields must render in Markdown and HTML like existing ones.
- Name mechanisms explicitly (LM Studio server logs, their line format and endpoints). Vague naming was rejected.
- The code must be defensible, toolset-agnostic and good enough for upstream maintainers to merge. The repo is public: no personal or machine-specific details, log content, identifiers or model names from a real machine in the spec, code or fixtures.
- The feature must stand on upstream main plus `apsis-shared-json`. Unmerged local-execution work in a fork (`orbit local`) must be able to build on it, not be needed by it.
- No agent or harness is the hero; the process is.

## Reported observations: LM Studio 0.4.26 server logs

These observations came from earlier agents' structural inspections. The recovery session checked public API documentation and repository code, without reading private logs or server configuration. They are evidence for synthetic grammar fixtures, not a public disk-format contract. The 13:39Z note checked only aggregate volume directly; its other corrections remain reported observations.

- Reported standard path: `~/.lmstudio/server-logs/YYYY-MM/YYYY-MM-DD.N.log`. The earlier aggregate check found about 131 MB across nine months, with a largest month of 42 MB; this is not a monthly rate or a portable performance target. Reported server settings include `verbose`, `logSensitiveData` and `fileLoggingMode: succinct`.
- Line format `[YYYY-MM-DD HH:MM:SS][LEVEL][tag] message`, local time, no zone. A request starts with `[DEBUG] Received request: POST to /v1/responses with body {` followed by multi-line JSON up to the next header line.
- Follow-up lines can carry model tags. The earlier correction reports that `Returning {` contains model lists, while inference bodies use `Generated prediction:` or `Generated synchronous v1/responses response:`. Most streamed responses are not logged. WARN and ERROR records have no demonstrated unique request link; adjacency and model tags are not proof of attribution.
- Long strings are reported as cut to head and tail with `... <Truncated in logs> ...`; the succinct-mode observation reports 50 characters at each end, often removing working-directory text. Every request body in the earlier two-day sample still parsed as JSON. Rotation near 10 MiB without splitting a record was reported, not established as a guarantee; incomplete records still need recovery.
- Up to 4 parallel slots per model, so log lines from concurrent requests can interleave.
- Earlier structural counts include `/v1/responses`, `/v1/chat/completions`, `/v1/completions`, `/api/v0/chat/completions`, `/api/v1/chat`, model-listing and embedding traffic, and `/v1/responses/input_tokens`. Exact inference endpoint selection is specified in requirements 2.1; token-counting, embedding and management paths are excluded.
- Chat conversations (`~/.lmstudio/conversations/`) are out of scope.

## Reported observations: client identifiers in request bodies

- Codex `/v1/responses` requests (504 of 550) carry `prompt_cache_key` and `client_metadata` with keys `thread_id`, `session_id`, `turn_id`, `root_turn_id`, `x-codex-installation-id`, `x-codex-window-id`, `x-codex-turn-metadata`. `store` is `false`.
- The earlier sample reports matching `client_metadata.thread_id` values between server requests, exec streams and an interactive rollout. Preserve that field as correlation evidence; only exact nonempty string thread identifiers group requests in the merged proposal. Other similarly valued fields do not establish a general identity rule.
- `previous_response_id` never appears (0 of 550).
- Within a thread, the `input` array shrank in 11 of 46 threads (context compaction), so message-prefix chaining would split those threads.
- Client prompt content can contain a working-directory block, but it can be truncated or quoted and is not authoritative project metadata. The merged proposal keeps LM Studio projects unknown and does not borrow a matching harness session's project.
- Requests with no client identifier: 46 `/v1/responses` with only `input` (a string), `max_output_tokens`, `model`; 12 `/v1/chat/completions` with `messages`, `tools`, `stream`; a few others. These need a grouping rule that does not rely on IDs.

## Starting point: requirements carried over from the combined draft

These were section 2 of the combined spec before the split. Treat them as input to rewrite, not as approved text.

1. Apsis SHALL read LM Studio server logs from `~/.lmstudio/server-logs/YYYY-MM/YYYY-MM-DD.N.log`, with the directory overridable, and SHALL list them under the source label `lmstudio`, with `-a lmstudio` to force the format.
2. Apsis SHALL translate each logged inference request — `POST` to `/v1/chat/completions`, `/v1/completions`, `/v1/responses`, `/v1/messages` or `/api/v1/chat` — into the shared representation with its timestamp, endpoint, model identifier, request messages and, where logged, the response; model listing, model management, embeddings and other requests SHALL be excluded.
3. Apsis SHALL group requests into sessions by a documented, deterministic rule; IF a request cannot be assigned to a session unambiguously, THEN it SHALL form a session of its own.
4. WHERE LM Studio truncated content (`<Truncated in logs>`), apsis SHALL mark that content as truncated in every output and SHALL NOT present it as complete.
5. Apsis SHALL attach the warnings and errors LM Studio logged for a request (for example unsupported fields, developer-role conversion and server errors) to that request in every output.
6. IF a log contains requests without bodies, THEN apsis SHALL list those requests with the metadata present and SHALL state that the body was not logged.
7. Because the logs record no project, LM Studio sessions SHALL appear when listing across all projects and SHALL NOT appear in a single-project listing.

Carried-over text above is historical input, not the current requirements or a human approval. The merged proposal retains all-projects-only discovery, unknown projects, separate server/harness listings and correlation metadata. Automatic harness links, discovery defaults and other user-owned choices remain in `drafts/requirements/questions.md`. Production renderers currently apply no tool-length truncation; shared JSON must retain all successfully parsed content.

## Review findings that apply to this spec

- No definition of a session; "deterministic rule" and "unambiguously" are circular. Need a stable session identifier, a rule for history that every request repeats (each message shown once), and an outcome for notices and response bodies that cannot be tied to one request when parallel slots interleave (attach to the session, marked unattributed).
- Identifiers must be safe in `apsis <id>` and `/sessions/{source}/{id}` (no `/`, at most 256 characters) and stay stable as the log grows.
- Local time without a zone: say which zone is assumed, emit an explicit offset, and define ordering through the repeated daylight-saving hour (file and line order rather than timestamp).
- Sessions spanning daily or `.N` files, and a body cut off at the end of a file, are undefined.
- Listing must parse large logs; `apsis serve` refreshes its list every 15 seconds. State a measurable target or a caching outcome.
- Raw `-f json` reads JSONL and follow mode (`-F`) parses Claude-shaped lines, so both need a defined outcome for `.log` input. A direct `.log` path and stdin need a defined result.
- The log holds full prompts when `logSensitiveData` is on; all-projects `apsis serve` raises exposure.
- Match endpoints exactly (exclude `/v1/messages/count_tokens`). `/v1/messages` and `/v1/completions` do not appear in captures and need fixtures built from documented request shapes, labelled as such.
- The format is undocumented and tied to the inspected version; say which lines are ignored silently, and what a "line" is for parse-error reporting when bodies span many lines.

## Notes for drafters

- Follow `SPEC-DRIVE.md` for which files to write and how to log the step.
- Read `specs/apsis-shared-json/requirements.md` so this spec builds on it without repeating it.
- You may read apsis code (`internal/transcript/`, `internal/sessions/`, `cmd/apsis/`, `internal/apsisweb/`) to check claims. Use synthetic fixtures and public endpoint documentation; no private log inspection is needed to continue the current work. Never copy private content, identifiers, paths or model names into any file.
- Put questions for the human under `## Open Questions for the User` at the end of your draft.
