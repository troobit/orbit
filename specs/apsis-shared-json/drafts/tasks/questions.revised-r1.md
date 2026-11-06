# Questions and Admission State: Shared JSON Tasks

This is the merged planning revision from both task drafts and their non-author cross-reviews. No human design/tasks approval is recorded. The dispatching coordinator's earlier ownership freeze was misattributed to the human; the checked 19:00Z note and SPEC-DRIVE authorise continued reviewed local planning and implementation. Existing task owners and explicit human decisions still apply.

## User-Owned Questions

- **U1, Claude model extraction:** unanswered. Keep current omission under approved 5.1; an expansion needs a later human answer and requirement change.
- **U2, Kiro IDE provider extraction:** unanswered. Keep current omission under approved 3.1/5.1; an expansion needs a later human answer.
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

## Next Review

A non-author critique and a fresh non-author peer review must assess this merged revision. The merge author does not review it. The coordinator may then record a reviewed local implementation admission without claiming human approval. No feature task was completed by this merge; the separately verified baseline repair keeps its original owner until its coordinator records completion.
