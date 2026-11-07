# Questions: Shared JSON Requirements

Revision 3 is ready for the approval step. These choices remain proposals; the user-approved split of LM Studio, provider-only metadata, no prompt-file reads and feature name remain in force.

## Questions to Ask One at a Time

### Q1. Which project owns a run with both a rollout and a stream?

- **A — Prefer the rollout globally (recommended):** list the run only as `codex`, with the rollout's recorded project, even when a stream was copied into another repository. The copied repository's project listing will not show that run. Direct conversion of the stream still works.
- **B — Prefer the rollout only within the selected scope:** allow the copied stream to appear in its containing repository when the rollout belongs elsewhere; source selection still disambiguates opening.

Revision 3 proposes A in [2.7](../../requirements.md#2.7) and Decision 9. It makes rollout preference consistent across views and preserves the richer source. B preserves a view of copied artifacts but makes source selection depend on the listing scope.

### Q2. How should an exec stream with no recorded time participate in listings and `latest`?

- **A — Use a labelled filesystem sort time (recommended):** omit source creation time, use file modification time solely as an explicitly marked listing/sort fallback, and emit no invented message timestamps.
- **B — Leave its time unknown:** omit the fallback too; sort unknown-time sessions before known-time sessions so a stream without a recorded time is not selected over a dated session by `latest`.

Revision 3 proposes A in [2.10](../../requirements.md#2.10), [4.3–4.4](../../requirements.md#4.3) and Decision 10. This preserves useful discovery while making copied/appended files' changing modification times visible as filesystem evidence.

### Q3. Which subdirectories should recursive discovery skip?

- **A — Skip `.git`, `node_modules` and `testdata` (recommended):** avoid repository internals, dependencies and parser fixtures. An explicitly configured directory is always scanned itself, even if it has one of these names, and direct file conversion remains available.
- **B — Skip `.git` only:** also discover logs beneath dependency and fixture directories; large configured trees cost more to scan and fixtures can appear as sessions.

Revision 3 proposes A in [2.5](../../requirements.md#2.5) and Decision 11. No guessed cold-time performance threshold is proposed; unchanged web refreshes must read zero transcript-body bytes.

## Resolutions Applied

| Review item | Resolution |
|---|---|
| Critique C1; peer 1 | Stable bounded per-file IDs, separate thread metadata, explicit CLI source selection, canonical rollout/project proposal; 2.7–2.8, 4.2, Q1. |
| Critique M1; peer 2–3 | Exit code, incomplete state, tool links, turn membership, generic payloads, existing cost/unit, structured warnings and first-known model/provider summaries; 1.1, 1.6, 2.2–2.4, 5.2. |
| Critique M2; peer 2, 4 | Pinned 0.166.0 types and emitter establish shapes and reported total counters; retain every usage field, never sum snapshots; 2.9, 5.6–5.7. |
| Critique M3; peer 6 | Omit unknown source time; distinguish proposed filesystem fallback, source order and stdin behavior; 2.10, 4.3–4.4, Q2. |
| Critique M4; peer 5 | Containment before reads, canonical-file deduplication, no directory-symlink descent, unchanged-refresh caching and explicit skip proposal; 2.5, 5.5, 5.9, Q3. |
| Critique remaining minors; peer 7–8 | Exact project-root matching, pre-thread forced parsing, raw exec JSON, settings defaults, schema coverage, deterministic latest and every-source CLI/web round trips. |
| Notes at 13:39, 14:10 and 14:13 UTC | Rechecked aggregate disk sizes and current Copilot/Kiro conversions. Corrected handoff facts; Decision 7 supersedes Decision 5 with corrected evidence, preserving the human's split decision. |

## Approval Sequence

Record each answer in `decision_log.md` and update the requirements before asking the next question. After Q1–Q3 are settled, offer **approve**, **request changes**, or **explain it first** for the whole requirements document. Do not mark the phase approved before an explicit response.
