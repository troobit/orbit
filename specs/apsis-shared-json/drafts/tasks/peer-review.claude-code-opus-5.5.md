# Peer review: tasks revision of 2026-11-04T22:36:41Z (claude-code-opus-5.5)

Independent-model peer review: Claude Code with Opus 5.5 reviewing the `codex-gpt-6.1-sol` revision. The critique and the other reviews were not read. Disclosure: one merged task draft came from an earlier Claude Code/Opus 5.5 session. In this session I also wrote `implementation-review.task2.claude-code-opus-5.5.md`, and M1 and M3 below repeat its findings at plan level.

Checked against requirements, design, Decisions 24/25/28, Q71–Q79, source and pinned Codex rust-v0.166.0 sources; `rune list` parses 46 items. Not human approval.

## Critical

None.

## Major

**M1. The plan's Codex model rests on a record that Codex 0.166.0 does not write.** Several places read the Codex model from `session_meta.payload.model`:

- the design's session and listing rows;
- task 2.1/2.2 (completed);
- task 3.1 ("a header-only Codex rollout keeps both");
- task 4.1 ("a model-only session_meta change");
- task 5.1/5.2 ("Codex rows carry Model/Provider from the same session_meta scan");
- task 14.1 (model-changing regressions);
- Decision 28.

In the pinned [protocol.rs](https://raw.githubusercontent.com/openai/codex/rust-v0.166.0/codex-rs/protocol/src/protocol.rs), `SessionMeta` has `model_provider` but no `model`; the model is `TurnContextItem.model`, written as `turn_context` lines. A key-presence count over this machine's 421 interactive rollouts (counts only) found:

- no `session_meta.model`;
- `turn_context.model` in all 421;
- 5 rollouts with several models;
- no `session_meta` after content.

So for real rollouts the plan delivers no Codex model in documents or listings, loses the recorded model changes (1.1, 1.6, 4.3), and tests a session_meta transition that never occurs.

Fix:

1. Add a red/green pair under task 2, reopening the parent, before task 3's Codex golden:
   - `FirstModel` comes from the first nonempty `turn_context.payload.model`, with `session_meta.model` as a fallback.
   - Under `PreserveModelChanges`, entries take the active `turn_context` model, and a change splits at the next content item.
   - The default `Entry.Model` stays as it is (5.1).
   - Add a synthetic fixture citing `TurnContextItem`, with `model_provider`, as a task 3.1 golden.
2. Task 5.1/5.2: the Codex head scan also reads the first `turn_context.model` within `codexMetaScanLimit`.
3. Tasks 4.1/14.1: drive the model-change regressions with `turn_context`.
4. Revise Decision 28's context.
5. Add user question U4: should Markdown/HTML headers show the Codex model? The options are shared JSON only (recommended under 5.1), headers too (needs a 5.1 amendment), or omit.

**M2. No task gives existing parser warnings a `code`.** The session document always carries a warning `code` (design "Session document" table), and the published list has none for:

- existing JSON decode failures (Claude, Codex, Copilot);
- unrecognised Codex event/item types;
- Codex outputs without a matching call;
- Kiro IDE unknown roles/actions.

Task 2 added `ParseWarning.Code` but sets it nowhere. The updated agent note expects "projections to supply a generic default", but task 3.1/3.2 define none. Any existing-source document with a warning will then emit an empty or invented code in the PR 1 schema.

Fix: extend the design's code list and add a red/green step before 3.2 projects warnings. Map decode failures to `malformed_line`, unrecognised records to a defined code such as `unrecognized_record`, and unmatched outputs to one such as `unmatched_tool_result`. Set the codes at the creation sites (transcript files; coordinate `parser.go` with task 9's owner). Task 3.1 should assert that every projected warning has a nonempty listed code.

**M3. `created_at` takes first-message times over recorded creation records (1.1).** Task 2.1 says "Copilot first entry timestamp". The completed code reads the first converted entry, the first prompt, and its test pins this against an earlier `session.start`. That event records `data.startTime`, already decoded as `CopilotData.StartTime`. The listing uses `workspace.yaml created_at`, so one Copilot session gets two different `created_at` values.

Kiro CLI shows the same pattern: parsed first-message time outranks the database's recorded `created_at` under Decision 24's parsed-first rule.

Fix:

1. In the M1 follow-up pair, Copilot `StartedAt` prefers `session.start` `data.startTime`, else the first parseable event timestamp.
2. Reword the design rows to "first source record".
3. Amend Decision 24 so an explicitly recorded creation field (`session.start`, Kiro CLI database `created_at`, Kiro IDE `startTime`) outranks first-message times, while filesystem fallbacks never qualify.
4. Add listing/session consistency assertions to task 4.1.

## Minor

**m1. Stale statuses.** Tasks 3/3.1 and 9/9.1 are In Progress with Owner `codex-gpt-6.1-sol`, but both claims were released at 22:49Z with no output.

Fix: the next claimant updates the status/owner through rune when claiming and logs it; the history stays in the log.

**m2. The Codex start time is not validated.** Task 2.2 captures the outer timestamp unvalidated, while the listing extractor accepts only RFC3339 values.

Fix: add to the M1 pair "validate with `parseTimestamp` and keep scanning", plus a test with an unparseable first header.

**m3. Ordering can differ between surfaces.** Task 6.2 enables extended ordering for every shared-JSON listing, including project scope, while the no-flag `latest` keeps legacy ordering. With equal or unknown times, the last shared-JSON row can therefore differ from `apsis latest`.

Fix: task 6.1 tests and README document the difference. Optionally, `latest -f shared-json` selects with extended ordering.

**m4. Split boundaries can separate a result from its call.** Decision 28 splits "at the next content item", which includes `function_call_output`, so a result can land in a different entry from its call.

Fix: split only on model-produced items, or assert paired rendering in task 4.1.

## Confirmed

Adjacent red/green pairs; per-slice README/CHANGELOG/Makefile checks; serialised module writes; Windows gating limited to 15.2/16; `ParseResolved` flag forwarding with `-a` precedence; warning-unit mapping; every requirement 1.1–5.9 mapped; no flag collisions.
