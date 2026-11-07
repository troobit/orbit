# Cross-review: draft.codex.md (requirements)

Reviewer: claude-code-opus-5.5. This review covers `draft.codex.md` and compares it with `draft.claude-code.md`. I share a harness and model with the author of `draft.claude-code.md`, so wherever I prefer that draft I give the evidence from code, logs or documentation, and the merge should weigh my preferences with that in mind. `draft.claude-code.md` still needs a cross-review from an agent independent of Claude Code with Opus.

Evidence used (counts and key names only; nothing copied):

- apsis code: `cmd/apsis/main.go` resolves a bare ID by trying sources in a fixed order and warns only when bound to exactly `0.0.0.0`. `internal/apsisweb` refreshes the list every 15 s, refuses sessions over 50 MB and rejects IDs over 256 characters or containing `..`, `/` or `\`.
- LM Studio server logs (79 files over nine months): counts of request paths, entry types and line positions.
- LM Studio REST documentation for `/api/v1/chat`.

## Per item: which version to keep

| Item | Keep | Why |
|---|---|---|
| Endpoints (codex 2.1; claude Terms, Q2) | claude | The logs hold 131 POSTs to `/api/v0/chat/completions`, which codex's allowlist would drop. `/v1/completions` was logged once, so the endpoints that need documented-shape fixtures are `/v1/messages` and `/api/v0/completions` (claude), not `/v1/completions` (codex 7.4). |
| Named entries (codex 2.4, 4.4; claude Terms) | claude, with one fix | The handoff asks for mechanisms by name, and codex never names the response-body entries. Fix in claude: per LM Studio's docs, `/api/v1/chat` sends `system_prompt` (not `instructions`), and `input` can contain image parts. |
| Format detection (codex 1.1; claude 7.1) | claude 7.1, plus codex's rule that a `.log` extension alone does not identify the format | Files often start with startup entries. Codex's "followed by a request" doesn't say how far after. |
| Discovery, override, containment (codex 1.2, 1.5, 6.4; claude 1.1–1.3, 8.4) | both | Claude adds the empty-variable disable that sibling 5.4 uses. Codex adds a warning when an explicitly set directory is missing. |
| Translation (codex 2.2, 2.3; claude 2.1–2.3) | claude, plus the last clause of codex 2.4 | Claude cites the log file and header line, and links each tool result to its call. Codex's "never infer output from a later request's history" is needed too (gap 2). |
| Missing, unreadable, truncated (codex 2.5, 2.6, 7.2; claude 2.4–2.6) | codex 7.2 for diagnostics | It adds "no body content in the diagnostic". |
| Ignored lines (codex 4.4; claude 2.7) | claude 2.7, plus the second clause of codex 4.4 | The handoff asks which lines are ignored silently. Codex adds a warning for unrecognised records that look like requests. |
| Thread grouping (codex 3.1; claude 3.1) | codex 3.1 | It states that a change of endpoint or model, or a shorter history, doesn't split a session. |
| Requests without a thread ID (codex 3.2; claude 3.2) | conflict U1 | |
| Identifiers (codex 3.3; claude 3.5) | codex 3.3, plus claude's "differs from every Codex identifier" | A bare `apsis <id>` tries sources in order, and thread IDs equal Codex session IDs. Codex's character set has no `.`, so an ID can't be mistaken for a file in the working directory (`isFilePath` checks whether the file exists). Per-request identifiers are cheap and help JSON consumers. |
| Repeated history (codex 3.5, 3.6; claude 3.3, 3.4) | codex 3.5 and 3.6, plus claude 3.3's response-body clause and 3.4's count | Codex keeps each request's full input recoverable in JSON. Claude stops a logged response from appearing twice and says how many items were dropped. |
| Attribution (codex 4.1–4.3; claude 4.1–4.6) | claude, with gaps 1 and 7 fixed | Codex 4.1 requires an explicit identifier, and the log has none: response bodies carry only server-generated `id`, `object` and `created`, and notices carry only a model tag. Codex's draft would therefore attribute nothing. Claude's position rule holds in the logs: all 2,643 acceptance warnings, plus all 898 other WARN/ERROR entries found in that position (all request-validation messages), sit in a run directly after a request. The 143 asynchronous failures fall elsewhere. |
| Unattributed lines (codex 4.2; claude 4.4, 4.5) | conflict U2 | |
| Time zone (codex 5.1, 5.2; claude 5.1, 5.3) | claude, plus a marker | Go reads `TZ` into `time.Local`, so no new option is needed. Keep claude's deterministic choice for a repeated hour, and mark those instants as inferred in JSON. That meets codex's "no false precision" without leaving entries without a time. The merge can settle this without the user. |
| Order, partial bodies (codex 5.3, 5.4; claude 5.2, Q5) | codex 5.4 plus claude 5.2 | Codex also covers a body still being written when `apsis serve` reads today's file. |
| Listing fields, gating (codex 6.1, 6.2; claude 3.6, 6.1, 6.2) | claude | Claude names the listing fields and matches sibling 4.2: opening a session needs the all-projects option. Codex 6.1 is ambiguous on that. |
| Correlation (codex 3.4; claude 6.3, 6.4) | conflict U3 | Both keep the thread ID in every output. |
| Refresh, cost (codex 6.3; claude 8.1, 8.2) | both | Codex's "zero log bytes read on an unchanged refresh" is testable. Claude's 3 s cold target needs a named reference machine, or should move to design. |
| Exposure (codex 6.5; claude 8.3) | both | Codex adds the help text and claude the serve warning; they complement each other. |
| Unsupported modes (codex 7.1; claude 7.3) | codex 7.1 | It points users to the shared JSON format. |
| Direct input (codex 1.3, 1.4; claude 7.2) | conflict U4 | Keep codex 1.4 either way: a session view excludes other sessions' bodies. |
| Fixtures (codex 7.3, 7.4; claude 8.5) | both lists combined | Codex adds identity across appends and attribution cases. Claude adds rotation and the repeated hour. |

## Conflicts for the user

- **U1. Requests without a thread ID.** Codex gives each one its own session. Claude joins a request to the one earlier session whose latest conversation it extends, on the same endpoint and model. Recommend claude, provided gap 3 is fixed. `/v1/chat/completions` requests carry no thread ID and make up 5,118 of 6,182 logged inference requests. With one session per request, a multi-turn client would list one session per turn, each repeating its whole history, which is what the handoff's "each message shown once" finding rules out.
- **U2. Lines that can't be attributed.** Codex copies their content into every session that has a request in the same file. Claude shows only a count in the candidate sessions. Recommend claude, adding the file and line of each such line to the session view, and the content only in the direct-file view. In a 10 MiB file of batch calls, codex's rule would copy every unattributed response into each single-request session and expose other clients' prompts.
- **U3. Link to the Codex session.** Codex leaves correlation to consumers. Claude links to the Codex session in `apsis serve` and shows that session's project path. Recommend the link but not the project path: sibling 4.1 shows a project path only "where the source records one", and LM Studio records none.
- **U4. A log file given directly.** Codex renders one transcript in source order, as one JSON object. Claude renders every session separately, as a JSON array. Recommend codex, adding the session identifier to each entry, so the shared JSON keeps one top-level shape.

Both drafts recommend reading the default directory, shown only behind the all-projects option, and both ask whether it should be opt-in instead. One question to the user covers both.

## Gaps in both drafts

1. **Rotation.** 23 of 79 files contain response, end-of-stream or disconnect lines before their first request (7 response or end lines, 45 disconnects), so requests outlive the file they started in. Claude 4.2 ends a request at the end of its file, and codex 4.2 scopes lines by file, so both misplace these lines. In-progress requests should be tracked across consecutive files.
2. **History shown as conversation.** Most streamed responses are not logged, so the assistant turns a reader sees come from the next request's history. Require output to distinguish context the client re-sent from a logged response.
3. **Comparing truncated strings.** Different strings can be identical after truncation to head and tail. Define the comparison: for grouping, a truncated item never matches; prefix de-duplication may treat identical truncated text as equal.
4. **Deleted log files.** Neither draft says what happens to identifiers or continuation chains when old files are deleted.
5. **Stdin identifiers.** Both drafts give sessions read from stdin identifiers, but neither says how they are formed without a file name.
6. **Excerpts.** With `-a lmstudio`, lines before the first header line need a defined outcome.
7. **Concurrent requests in a thread.** Claude 4.2 ends a request when the next request in its session arrives, which assumes a thread never has two requests in flight. That's unverified. If it's wrong, a late line attaches to the wrong request.
8. **Network warning.** `apsis serve` warns only when bound to exactly `0.0.0.0`, and the LM Studio warning text inherits that limitation.
9. **Links.** Codex's `../../../apsis-shared-json/` link only works from `drafts/requirements/`. The merged `requirements.md` needs `../apsis-shared-json/`.
