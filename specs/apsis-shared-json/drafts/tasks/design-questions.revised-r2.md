# Questions: Shared JSON Design

The design has been revised using completed non-author reviews and the task-merge corrections through Q79/Decision 28. Non-superseded design decisions remain agent proposals; no human design/tasks approval is recorded. U1–U3 remain unanswered. The earlier freeze was the dispatching coordinator's ownership instruction, misattributed to the human (checked 19:00Z note), and does not override SPEC-DRIVE's continuous local-work authorisation. Preserve current omission for U1/U2 inside approved scope; admit U3 only as a reversible local technical choice.

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

A matches the approved requirements. B and C can follow without a schema version change, but changing existing rendered output first requires amending approved scope.

### U3. May the module gain a test-only JSON Schema validator?

Decision 25 retains the proposal to add `github.com/santhosh-tekuri/jsonschema/v6` to `go.mod`, imported only from `_test.go` files. It stays out of the binaries; its one non-test dependency is already in the module graph.

- **A — Add the test-only validator (recommended):** fixtures are checked by a maintained Draft 2020-12 implementation.
- **B — No new module:** write and test a validator for the keywords the schema uses.

A is a reversible technical choice admitted for local implementation using non-author review evidence; it is not a human U3 answer. It remains visible because upstream maintainers will carry the dependency.

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
| Current critique 1/5, peer review 1 | Complete-line probing precedes heuristic-only Kiro fallback; split lines are recombined; automatic detection accepts only the exec outcome (Decision 21, Q56). |
| Current critique 2, peer review 2 | Listing/resolution/index share the existing selected rollout winner; an invalid first duplicate cannot be rescued silently by another file for suppression (Decision 22). |
| Current critique 3 | Unix/Windows identity and change fingerprints are required; native Windows replacement verification gates cache admission with no weak fallback (Decision 23). |
| Current critique 4, peer review 4 | Parsed FirstModel/FirstProvider summaries retain header-only and user-only source facts; entries preserve subsequent active values (Decision 24). |
| Current peer review 3 | Existing Claude timestamp and Codex metadata scan/error semantics remain; only new optional Claude cwd extraction is bounded (Q57). |
| Current peer review 5 | A known item without a valid source ID becomes a generic event with a warning, creating no colliding tool links (Q55). |
| Current revision inputs | The exact 18:21Z design/decisions/questions remain in `*merged-r1.md`; both new reviews are same-model fresh-context evidence, not an independent-model pair (Q58/Q59). |
| Explain-like self-validation | New deterministic ordering is scoped to feature-enabled modes, and the options closure targets shared settings rather than Lister runtime state (Q60/Q61). |
| Revision-check major 1 | Global winner selection is restricted to suppression and all-projects row deduplication; legacy project rows keep per-file enumeration/cwd filtering (Q49/Q62). |

## Approval Sequence

1. When a human is present for formal approval, ask U1, U2 and U3 one at a time and record the actual answers. Headless local work keeps U1/U2 omission and the reviewed U3 technical admission without recording a human answer.
2. Ask whether the design is approved, offering approve, request changes, or explain it first.
3. Only after explicit human approval, mark the current non-superseded proposed decisions accepted; keep the superseded entries and their replacement links intact.

## Task-Merge Corrections

The canonical plan now has four PR phases and preserves all original IDs/status/owners. Decision 25 publishes the full known v1 schema initially with an unknown-block extension and exact-initial-schema compatibility tests; Decision 26 isolates native fingerprints; Decision 27 limits the native Windows gate to cache and serializes module writes. Q69–Q72 record reproducible benchmark evidence, matched Kiro keys/all-project discovery injection, shared-json-only empty Codex acceptance, and deterministic digest-conflict testing. The native Windows prerequisite remains unverified in [prerequisites.md](../../prerequisites.md). The merged task critique found no defects; peer major 1 is addressed by option forwarding. A non-author follow-up must check that fix, corrected Kiro warning units and the new shared-json-only PreserveModelChanges option; these remain agent decisions, not approval.

## Review-Driven Revision

Q74 preserves caller options and explicit forced-format precedence through `ParseResolved`; Q75 supersedes the blanket Kiro IDE position assumption in Q39 with physical/chat/action positions. Decision 28 adds a shared-json-only model-change preservation option while identical metadata and provider-less default output keep current consolidation. Task 2 tests both modes and outer Codex timestamp provenance; task 14 repeats ID/latest/web parity after helper replacement. The exact prior reviewed inputs remain in `drafts/tasks/*revised-r1.md`. Task 1's separately verified completion retains the original owner. No new human question, answer or whole-document approval is implied.
