# Questions and Admission State: Shared JSON Tasks

This is the merged planning revision from both task drafts and their non-author cross-reviews. No human design/tasks approval is recorded. The dispatching coordinator's earlier ownership freeze was misattributed to the human; the checked 19:00Z note and SPEC-DRIVE authorise continued reviewed local planning and implementation. Existing task owners and explicit human decisions still apply.

## User-Owned Questions

- **U1, Claude model extraction:** unanswered. Keep current omission under approved 5.1; an expansion needs a later human answer and requirement change.
- **U2, Kiro IDE provider extraction:** unanswered. Keep current omission under approved 3.1/5.1; an expansion needs a later human answer.
- **U4, Codex model in Markdown/HTML headers:** unanswered. Decision 29 reads the Codex model from `turn_context` for shared JSON and listings only (option A, recommended); showing it in headers needs a 5.1 amendment. See [design questions](../design/questions.md#u4-should-codex-markdown-and-html-headers-show-the-model).
- **U5, Copilot `producer` as origin:** unanswered. Q87 leaves it unmapped (option A, recommended), so Copilot documents and listings carry no `origin`. See [design questions](../design/questions.md#u5-should-copilots-recorded-producer-count-as-session-origin).
- **U3, test-only schema validator:** unanswered as a human question. The maintained local-only validator is a reversible agent technical admission supported by the design/task reviews (Decision 25); do not label it human-approved.

These choices do not block the plan's existing approved scope. The native Windows check in [prerequisites.md](../../prerequisites.md) is an external capability prerequisite for task 15.2 and cache task 16 only; it is unavailable and unverified here. No CI changes or relaxed cache promises are authorised.

## Cross-Review Dispositions

| Finding | Merged disposition |
|---|---|
| Codex predecessor chain blocks listing/exec on native Windows | Four PR phases; only task 16 depends on the fingerprint gate. |
| Oversized pairs and missing PR boundaries | Claude's sixteen coherent parents with fifteen adjacent red/green child pairs; all source/CLI/web integrations are covered. |
| Concurrent go.mod/go.sum writers | Task 15.2 explicitly waits for task 3.2; module mutations have one owner at a time. |
| Staged closed block enum rejects newer v1 output | Publish the full known v1 schema in task 3.2, plus a mutually exclusive unknown-type extension branch and first-schema compatibility tests (Decision 25). |
| Windows host unavailable; x/sys ABI structs absent | Explicit prerequisites; isolated package, local ABI structs, vet/cross-compile checks, and no native-verification claim (Decision 26). |
| Real digest collisions cannot be fixtures | Inject only the digest helper; assert both omissions and warning under either root order. |
| Missing benchmark acceptance evidence | Record generated dimensions/platform/cold-warm measurements; 1.5 s is an investigation budget, never a portable failure threshold. Zero unchanged body bytes stays mandatory. |
| Missing matched Kiro key and discover injection | Tasks 4 and 16 name the actual matched-key result and both project/all-project discovery injection. |
| Links, metadata/status preservation and freeze attribution | Canonical-relative links; byte-exact snapshot; all original IDs/status/owners retained and scope mapped in [merge map](merge-map.codex-gpt-6.1-sol.md); attribution corrected. |
| CLI/latest/text-column/follow and per-slice conventions | Tasks 4/8/12/14 explicitly cover them; each slice closes with README/CHANGELOG and Makefile checks. |
| Header-only Codex success could change legacy errors | Only shared JSON enables AllowEmptyCodex; default parser/CLI/web acceptance stays unchanged, with paired tests (Q71). |

## Current Review Dispositions

- The current non-author fresh-context critique found no planning defects. It supplies same-model evidence, not independent-model or human approval.
- Peer major 1 is resolved by an option-bearing `ParseResolved` contract and task-14 ID/latest/web regressions. Caller options retain `AllowEmptyCodex` and `PreserveModelChanges`; source-bound Kiro cost data is merged without discarding them. CLI explicit `-a` keeps precedence; web callers use defaults (Q74).
- The checked 22:30:30Z note is applied: Kiro chat indexes map to `chat_message`, execution-detail action indexes to `execution_action`, and physical lines to `line`; existing numeric positions and stderr diagnostics remain unchanged (Q75 supersedes Q39). Both warning paths enter initial-schema tests in tasks 2/3.
- Identical repeated Codex metadata preserves consolidation. Model-only adjacent changes are retained for shared JSON through the explicit `PreserveModelChanges` option, while provider-less default rendering keeps legacy grouping. Provider transitions may create metadata-bearing entries under 3.1. This routine technical reconciliation of approved 1.6/5.1 needs no new human question; Decision 28 and focused parser/helper parity tests make it reviewable.
- Codex creation time uses the outer session_meta envelope timestamp only, consistent with existing listing provenance; no payload timestamp fallback is added (Q77).

Exact reviewed inputs are preserved as `*revised-r1.md`. Task 1 is now legitimately completed under its original `codex-gpt-6` owner by the separate coordinator record; this revision changes no statuses, owners or identities.

## Independent Peer Review Dispositions

| Finding | Disposition |
|---|---|
| M1 Codex model read from `session_meta`, which 0.166.0 never writes | Decision 29 supersedes Decision 28: model from `turn_context` (session_meta fallback), entry models only under `PreserveModelChanges`; new task 2.3/2.4 pair; tasks 3.1, 4.1, 5.1, 5.2 and 14.1 updated; U4 added |
| M2 existing parser warnings have no code | Q80 code mapping set at creation sites in task 2.3/2.4; tasks 3.1/3.2 assert and copy nonempty codes |
| M3 first-message times outrank recorded creation records | Decision 30 supersedes Decision 24: parsed start record, resolver-recorded creation, then first entry timestamp; Copilot uses `session.start`; Kiro CLI parser sets no `StartedAt`; tasks 3.1/3.2/4.1 updated |
| m1 stale In Progress/owner on 3, 3.1, 9, 9.1 | Q84: reset to pending, owners released on 3, 3.1, 3.2, 9, 9.1, 9.2; task 2 reopened with its recorded owner kept for provenance |
| m2 unvalidated Codex start time | Q81 in task 2.3/2.4 |
| m3 shared-JSON vs `latest` tie order | Q83: documented and tested in task 6.1 |
| m4 split at a function call output | Q82 in task 2.3/2.4 |

Exact prior inputs are preserved as `*revised-r2.md`. Tasks 3 and 9 now wait for the reopened task 2.

## Requirements Revision 4 Absorption

| Input | Disposition |
|---|---|
| 1.7 recorded session origin (Decision 31, Q86) | Claude first `entrypoint`, Codex first `session_meta.payload.originator`, others omitted (Q87, U5); shared JSON `origin` in session metadata and listing rows (Q88); listing value from the existing bounded scans (Q90). Tasks 2.5/2.6, 3.1/3.2, 5.1/5.2, 6.1/6.2 and 10.1 |
| 1.8 recorded per-entry identifiers (Decision 31, Q86) | `Entry.RecordID` set only where an entry is built from one record; merged entries carry none; projected as entry `id`, separate from tool-linking fields (Decision 32, Q88). Tasks 2.5/2.6, 3.1/3.2 and 10.1 |
| Requirement links | Tasks 2, 3, 5, 6 and 10 and their children cite 1.7/1.8 where they implement or test them; task 5's title names origin |
| Task identity and state | New pair 2.5/2.6 follows 2.4 because it edits the same parser files; no status or owner changed; rune parses 50 items (Q91) |

Exact prior inputs are preserved as `*revised-r3.md`.

## Next Review

A fresh agent outside the reviewing and revising sessions, preferably another harness or model, must verify two revisions before task 2.3–2.6 is admitted locally:

1. The 23:07Z revision: Decisions 29–30, Q80–Q85 and the task 2.3/2.4 pair against the pinned Codex definitions and current code.
2. The requirements revision 4 absorption: Decision 32, Q87–Q91, U5, task pair 2.5/2.6 and the task 3, 5, 6 and 10 changes against requirements 1.7/1.8 and the current parsers (entry construction per source, the Claude `UnmarshalJSON` alias, raw `-f json` pass-through and the `Entry.UUID` renderer lookups).

U1–U5 and native Windows remain unanswered or unverified; no human approval is implied.
