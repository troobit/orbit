# Questions: Shared JSON Design

The merged design is ready for critique and peer review, then the approval step. Decisions 12–20 and Q29–Q54 are agent decisions with status `proposed`; none is a human approval. The questions below are the only choices that change user-visible scope. Until they are answered, local implementation follows each recommendation. That is the default because it stays inside the approved requirements and can be extended later without breaking anything.

Approved policies are not reopened here: global rollout precedence (Q25), the labelled filesystem time (Q26), directory skips (Q27), provider-only reporting (Decision 6), no prompt-file reads (Decision 4) and the separate LM Studio spec (Decision 7).

## Questions to Ask One at a Time

### U1. Should Claude Code sessions report their model?

Claude Code transcripts record `message.model`, but apsis's parser keeps only role and content, so Claude entries carry no model today and Claude Markdown/HTML headers show none.

- **A — Leave it out in this feature (recommended):** Claude sessions show no model in shared JSON, listings or rendered output. A follow-up amends [5.1](../../requirements.md#5.1) and adds the model everywhere at once.
- **B — Shared JSON only:** extract the model and emit it in shared JSON documents and listings, while Markdown and HTML stay unchanged. This conflicts with the direction that new fields render in Markdown and HTML like existing ones.
- **C — Amend 5.1 now:** extract the model and also show it in Claude Markdown/HTML headers, changing existing Claude output.

A keeps the first PR inside approved scope, and listing and session documents stay consistent. C is the complete answer, but it changes output that 5.1 currently protects.

### U2. Should Kiro IDE sessions report their provider?

Kiro IDE records `metadata.modelProvider`. [3.1](../../requirements.md#3.1) and [5.1](../../requirements.md#5.1) limit the provider to Codex sessions.

- **A — Leave it out in this feature (recommended):** a follow-up amends 3.1/5.1 and adds the Kiro IDE provider to every output.
- **B — Shared JSON only:** emit it in shared JSON while Kiro IDE Markdown/HTML headers stay unchanged.
- **C — Amend now:** show it beside the model in Kiro IDE Markdown/HTML headers as well.

A matches the approved requirements. B and C are additive and can follow without a schema version change.

### U3. May the module gain a test-only JSON Schema validator?

Decision 18 adds `github.com/santhosh-tekuri/jsonschema/v6` to `go.mod`, imported only from `_test.go` files. It stays out of the binaries; its one non-test dependency is already in the module graph.

- **A — Add the test-only validator (recommended):** fixtures are checked by a maintained Draft 2020-12 implementation.
- **B — No new module:** write and test a validator for the keywords the schema uses.

A is a reversible technical choice, admitted for local implementation. It is asked because upstream maintainers will carry the dependency.

## Resolutions Applied

| Input | Resolution |
|---|---|
| Both drafts, both cross-reviews (item 1; review 1) | Extend `Entry`, project in `internal/sharedjson` (Decision 12, Q35). |
| Claude cross-review items 2–5, 18, 20 | Codex envelope, smaller v1 entry fields, warning codes plus position unit, settings rules, rollback-journal and stale-failure cache rules (Q33, Q36, Q38, Q39; design §4). |
| Claude cross-review items 3, 7–17; Codex cross-review 1, 3 | Claude time fields, tool-result emission, item table, two-pass HTML, rendering table, long-lived lister with resolver branches, bounded extraction, `GetSessionByID`; companion fingerprints including absent files and restat after extraction (Decisions 13, 15; Q37, Q42, Q51). |
| Codex cross-review 2; Claude cross-review 10 (disagreement) | Provider shown wherever the model is shown; grouping unchanged (Decision 16). |
| Codex cross-review 4 | Shared streaming probe for detection and discovery (Decision 20). |
| Codex cross-review 5; 17:23Z note | Published schema open; test copy closes apsis-owned structural objects only (Decision 18). |
| Codex cross-review 6; Claude cross-review 16 (disagreement) | Suppression index uses rollouts whose recorded ID matches their filename UUID (Decision 17). |
| Codex cross-review 7; Claude cross-review 17 (disagreement) | Readable thread prefix plus full SHA-256 path digest; on-disk case for roots and link targets; conflicts omit both sides (Decision 14). |
| Codex cross-review 8 | U3 asked for maintainer visibility, but admitted locally; approved policies not reopened. |
| Claude cross-review conflicts A–D | A resolved by 5.1 (Q47); B → U1; C → U2; D → U3. |
| Claude cross-review gaps 1–6 | BOM strip for raw exec (Q48); raw path otherwise unchanged (Q48); shared JSON dispatched before the empty-entries return (design §3); `--all-projects` has no variable (Q31); Kiro IDE tie rules confirmed equivalent and shared (design §6); fresh slices per pass (Decision 13). |
| Fresh-context critique 1–8 | Per-source metadata table and precedence (Decision 19); concrete v1 shapes and dispatch (Data Models); grouping kept unchanged (Decision 16); Kiro exact-row API and shared candidate selection (Q51, design §6); fingerprint rules (design §4); slice ownership (Decision 13); two-pass HTML (parity audit); raw BOM (Q48). |
| Note 17:09Z (checked with grep) | Q6 rationale superseded by Q29. |
| Note 17:19Z requirements (checked against the log and Q25–Q28) | Handoff "Pending approval" corrected. |
| Notes 17:18Z, 17:19Z implementation, 18:13Z | Baseline facts only; no design change. `tasks.md` and code left unchanged. |
| Unlogged `design.md` and `tasks.md` from codex-gpt-6 | Preserved as `design.codex-gpt-6-unlogged.md` (identical to `draft.codex-gpt-6.md` apart from title and link paths) and `tasks.codex-gpt-6-unlogged.md`. |

## Approval Sequence

1. Ask U1, U2 and U3 one at a time; record each answer as a quick decision and apply it to `design.md`.
2. Ask whether the design is approved, offering approve, request changes, or explain it first.
3. On approval, mark Decisions 12–20 accepted.
