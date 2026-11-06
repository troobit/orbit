# Requirements Cross-Review: LM Studio Server Logs

Reviewer: `codex-gpt-6`. Reviews [Claude's draft](draft.claude-code.md), comparing [Codex's draft](draft.codex.md) and both decision proposals. C and X below identify those drafts. Checked CLI conversion, format detection, session resolution, web limits and polling against repository code; endpoint semantics against public documentation. No local logs were read.

## Critical

1. **C4.1–4.5 / X1.4, X4.1–4.3: keep X's explicit-match rule, revise event visibility.** C closes requests at file boundaries or when their session next sends a request, neither of which proves completion. Synthetic counterexample: request A survives rotation; request B starts in the next file; A's response is then assigned to B. Adjacent WARN/ERROR entries can likewise belong to older concurrent work; observed adjacency establishes a pattern, not an attribution guarantee. Remove C's inferred active-request counters. However, X4.2 copies potentially unrelated response bodies into every session touching a file. Keep full unattributed events in direct log views; session views should expose uncertainty and source references without copying unrelated bodies. Fixture both cases.

## Major

2. **C3.1–3.2 / X3.1–3.2: keep X.** Use exact nonempty thread identifiers and singleton sessions otherwise. A unique prefix match among recorded requests cannot establish client identity, especially with truncated strings. C's own Decision 1 admits false merges. Preserve missing/invalid identifier handling from X; prefix grouping needs explicit human approval as a heuristic.

3. **C Terms, C2.1–2.2, C8.5 / X2.1–2.3, X7.4: combine, correcting native payloads.** Keep C's seven exact endpoint paths: both v0 completion endpoints are [documented](https://lmstudio.ai/docs/developer/rest/endpoints). Keep X's explicit system-instruction preservation and fixture-origin distinction. Native chat uses `system_prompt`, typed `input`, `output` and `stats`, rather than Responses' `instructions`/usage layout ([native API](https://lmstudio.ai/docs/developer/rest/chat)). Add endpoint-specific translation cases. [Stateful chat](https://lmstudio.ai/docs/developer/rest/stateful-chats) need not resend history: preserve `previous_response_id` as evidence and label unavailable server-held context without inventing it; response-chain grouping can remain deferred.

4. **C3.3–3.4 / X3.5–3.6: keep X.** Exact leading-prefix references preserve request reconstruction and distinguish supplied assistant context from logged output. C's “each item once” and response-equality suppression can erase intentional repeated messages. Include instructions and recorded identifiers in equality; preserve request boundaries even for identical or shorter input.

5. **C5.1–5.3 / X5.1–5.4: keep X's uncertainty and partial-record outcomes; keep physical ordering from both.** A monotonicity check cannot distinguish the two occurrences of an isolated repeated-hour timestamp. Do not assert a recovered instant for ambiguous/nonexistent local times. Declare the assumed zone in every format. C's existing `TZ` mechanism is sufficient unless a separate option has a concrete need.

6. **C7.1–7.3 / X1.1, X1.3, X7.1: prefer X, pending human choice.** A timestamp/level header alone also recognises unrelated logs; require request-specific evidence with a documented bounded scan and retain forced parsing. Prefer one direct-log transcript with logical-session identifiers over C's input-dependent object/array switch, which conflicts with [shared JSON 1.2](../../../apsis-shared-json/requirements.md#1.2). Keep clear raw-JSON/follow errors directing users to shared JSON.

7. **C6.1–6.4 / X6.1–6.2: keep X's unknown-project baseline and C's richer metadata.** Correlation metadata is useful; automatic Codex links and borrowed projects expand scope. A copied thread identifier does not make a project authoritative, and multiple matching records need a selection rule. Preserve the identifier without merging sources and C6.2's all-projects gate for identifier opening; send automatic linking/project attribution to the human.

8. **Both drafts: define missing outcomes.** Specify malformed-header recovery, unsupported body shapes and oversized-record handling. Define deletion/replacement invalidation and older-file import effects on identity/history. Require warning visibility in web session views; current transcript handling drops parser warnings.

## Minor

9. **C8.1–8.5 / X6.3–6.5, X7.3–7.4: combine.** Keep zero-body-read unchanged refreshes, synthetic fixtures, all renderer/schema checks, help disclosure and C's extended existing network warning. Replace the uncalibrated three-second gate with a reproducible benchmark recording environment, bytes and request count; record any agreed hardware/time target explicitly.

## Remaining Items to Retain

| Items | Choice and reason |
|---|---|
| C1.1–1.4; X1.2, X1.5, X6.4 | Combine directory precedence/disable, containment, missing-root and unreadable-file warnings; preserve other sources. |
| C2.3–2.7, C4.6; X2.4–2.6, X4.3–4.4, X7.2 | Keep notices/errors in every format, generic events, explicit missing responses/bodies, truncation and recovery; X gives physical line ranges and body-free diagnostics. |
| C3.5–3.6; X3.3–3.4 | Combine safe stable IDs, request IDs, correlation metadata and session-owned size; existing web size limits make size semantics observable. |

## Questions for Approval

Should unidentified requests remain singletons? Should unattributed bodies appear only in direct views? Should direct conversion remain one transcript? Should correlation create links/projects? Should default discovery require additional opt-in? Recommendations: yes, yes, yes, defer, and no additional opt-in beyond all-projects discovery.
