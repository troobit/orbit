# Implementation review: task 2 (claude-code-opus-5.5)

Independent-model review (Claude Code/Opus 5.5; author `codex-gpt-6.1-sol`) of the `internal/transcript` diff (excluding task 1's `follow_test.go`), `parser_metadata_test.go` and `provider_test.go`, against task 2, design Components 1, requirements 1.1/1.6/3.1/5.1, Decisions 16/24/28 and Q74–Q79. The earlier same-model review was not read first. Not human approval.

## Critical

None.

## Major

**M1. The Codex model is read from a record that Codex 0.166.0 never writes (1.1, 1.6, Decision 28).** `processSessionMeta` takes the model from `session_meta.payload.model`. In the pinned [rust-v0.166.0 protocol.rs](https://raw.githubusercontent.com/openai/codex/rust-v0.166.0/codex-rs/protocol/src/protocol.rs), `SessionMeta` has `model_provider` but no `model`. The model is `TurnContextItem.model`, written as rollout `turn_context` lines, and `codex_parser.go` skips those lines. A key-presence count over this machine's 421 interactive rollouts (counts only, no content kept) found:

- 0 `session_meta` lines with `model`;
- 421 with `model_provider`;
- 421 rollouts with `turn_context.model`;
- 5 rollouts whose `turn_context` records more than one model;
- 0 rollouts with a `session_meta` after the first content line.

Consequences for real rollouts:

- `FirstModel` is always empty.
- Recorded model changes are dropped, against 1.6.
- Headers render `(openai)` with no model.
- The session_meta-driven split for `PreserveModelChanges` and providers never fires on observed data, while the record that actually changes is ignored.

The synthetic fixture, the design's Codex session and listing rows and the handoff's "apsis keeps the model" share this premise; the code follows the design, so the defect is in the spec.

Fix:

1. Revise the design table and Decision 28 so the Codex model comes from `turn_context.payload.model`, with `session_meta.model` kept only as a fallback for older files.
2. `FirstModel` takes the first nonempty value in transcript order.
3. Under `PreserveModelChanges`, set entry models from the active `turn_context` and split at the next content item when the value changes.
4. Default parsing keeps today's `Entry.Model`, so 5.1 output is unchanged.
5. Whether Markdown/HTML headers should show the Codex model is a user-owned parity question, like U1. Options: shared JSON only, recommended now; also in headers, which needs a 5.1 amendment; or still omitted.
6. Add a synthetic `turn_context` fixture whose shape cites `TurnContextItem`.

Land this before task 3 freezes Codex golden documents.

**M2. The Copilot `created_at` is the first prompt time, not the recorded creation time (1.1).** `ParseCopilot` takes `StartedAt` from the first converted entry and skips `session.start`. That event records `data.startTime` (already decoded as `CopilotData.StartTime`) and an event timestamp. The new test pins the gap: `session.start` is at 00:00:00Z but `StartedAt` is expected at 00:00:01Z. This conflicts with the Claude row, which uses the first parseable record including infrastructure lines. It also makes the shared-JSON `created_at` differ from the listing's `workspace.yaml created_at` for the same session.

Fix: use the first parseable `session.start` `data.startTime`, else the first parseable event timestamp. Update the test, and reword the design row to "first source record". Land this before task 3's Copilot golden.

## Minor

**m1. Existing warnings have no `Code`.** No creation site sets `ParseWarning.Code`. The updated agent note expects "projections to supply a generic default", but neither the design's code list nor task 3 defines one, and the session schema requires `code`.

Fix: assign codes where warnings are created. Use `malformed_line` for Claude/Codex/Copilot JSON decode failures. Use defined codes for unrecognised record types/roles/actions and for Codex outputs with no matching call. Alternatively, task 3 can document one fallback code.

**m2. The Codex `StartedAt` is not validated.** The first nonempty outer timestamp is accepted as is. The listing extractor (`getCodexSessionTimestamp`) accepts only an RFC3339-parseable value and keeps scanning. An unparseable string would therefore become `created_at` and disagree with the listing.

Fix: check with `parseTimestamp`, as the Claude path does, and keep looking when it fails.

**m3. A transition at `function_call_output` separates a result from its call.** `ensureAssistantEntry` runs for outputs too, so a metadata transition moves the result into a new assistant entry under the new label. The test only checks that the output text appears.

Fix: split only on model-produced items (message, reasoning, function_call) so that outputs join their call's entry. Or assert paired rendering. No local rollout has a header after content, so this has low impact today.

## Confirmed

- `Format` is set on every path; the Claude fallback gives `FormatUnknown`.
- Options reach Codex through all three entry points; defaults keep the empty-Codex error and model-only consolidation; identical headers do not split; outer timestamp only.
- `Provider` is `json:"-"`; `modelLabel` is used only at the two assistant header sites, with HTML escaping; other headers are unchanged.
- Kiro IDE units are correct; stderr text is unchanged.
- Every `Metadata != nil` consumer (`cmd/apsis`, `apsisweb`, `internal/web`, `logs/manager.go`) copies the nil-safe `TotalCost`.
- No fixture records `model_provider`, so existing goldens still prove unchanged output.

## Verification run (2026-11-04T22:55Z)

With temporary caches, `go test ./internal/transcript/` and `make lint` passed. `make test` passed except `internal/apsisweb` `TestIntegrationEndToEnd` (`shutdown error: context deadline exceeded`), which failed in two package runs and passed alone with `-run`. It does not exercise the changed metadata; this is the logged shutdown flake.

## Gate

M1 and M2 are fixes to task 2 metadata, and they come before task 3's Codex and Copilot golden documents. M2 is a local parser fix. M1 needs a design/Decision 28 revision, and it raises a header-display question for the human. Task 9 (the exec parser) does not depend on either.
