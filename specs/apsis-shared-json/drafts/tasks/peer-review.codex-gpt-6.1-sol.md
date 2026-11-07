# Shared tasks peer review

Reviewed the 2026-11-04T22:27:26Z merged plan, including the authorised status-only completion of task 1 at 22:28:08Z. Fresh-context NON-AUTHOR SAME-MODEL evidence; neither independent-model evidence from the current critique nor human approval. No current critique, cross-review or other current review files were read. Findings use canonical requirements, design, decisions, prerequisites, questions and repository source.

## Critical

None found.

## Major

1. **Tasks 4.2, 14.1 and 14.2: the shared parse helper loses the format-specific empty-Codex option.** Tasks 2/4 and Q71 correctly require `AllowEmptyCodex` only for shared JSON, preserving legacy Markdown/HTML/web errors. The [design](../../design.md#6-resolution-in-all-projects-scope) then prescribes `ParseResolved(rs *ResolvedSession)` with no options parameter, auto-detecting everything except Kiro IDE and exec; task 14 replaces the CLI/web parse branches with that helper. Its only input has source/session metadata, not the requested output format. It cannot distinguish a shared-JSON header-only Codex lookup from the same web/Markdown lookup. Existing [CLI conversion](../../../../cmd/apsis/main.go) chooses parse options while it still knows the output format; [web parsing](../../../../internal/apsisweb/handlers.go) must keep the default. A literal implementation of task 14 therefore undoes task 4's accepted header-only shared-JSON outcome, or broadens legacy acceptance if it enables the option globally.

   **Fix:** make `ParseResolved` accept caller-supplied `transcript.ParseOptions` (or an explicit equivalent parsing context), merge its source-specific Kiro cost option without discarding caller options, and keep CLI output-format selection responsible for enabling `AllowEmptyCodex`. Update the design/helper contract and tasks 14.1/14.2 together. Add task-14 regression cases for the same header-only Codex session through explicit-source ID and `latest`: shared JSON returns metadata plus `entries: []`; Markdown/HTML and the web retain their previous error. Preserve forced-format selection when consolidating the CLI dispatch.

## Minor

None found beyond that integration contract.

## Checks and retained boundaries

- Rune JSON parses 46 items under 16 parents, with 58 explicit dependencies. The graph remains acyclic after adding parent-completion edges; ready leaves are 2.1 and 15.1. Red/green children retain their prerequisite gates.
- Original numbered/stable IDs 1–11 are preserved. Statuses match the pre-merge snapshot except task 1's authorised completion; its owner remains `codex-gpt-6`.
- All 31 criteria are linked. Transcript/projection and sessions/CLI/web writers are ordered across their shared files; module writes are serialized after task 3.2. Native Windows evidence blocks task 15.2 completion and task 16 only, preserving PRs 1–3.
- Task 3 publishes all known v1 branches initially, excludes known discriminators from the open future-type branch, and retains exact-initial-schema checks. This preserves both additive compatibility and validation of malformed known blocks. No schema implementation exists yet, so these are plan checks.
- The five design risks have concrete tests or measurement evidence. U1/U2 remain scope expansions; U3 stays a reversible agent technical admission. LM Studio policy remains unanswered and separate. No source/build tests, native execution, private logs or external actions were used for this review.

Before implementation admission: will task 14 explicitly carry the selected parse options and retain the paired header-only CLI/web regression cases?
