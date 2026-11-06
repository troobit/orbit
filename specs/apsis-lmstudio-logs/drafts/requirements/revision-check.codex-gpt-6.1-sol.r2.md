# LM Studio Requirements Revision Check, r2

Fresh-context, non-author verification by `codex-gpt-6.1-sol` of the 18:53Z revision: 47 criteria, Q19–Q28 and Decisions 11–13. This is same-model verification of changes prompted by the [independent Claude review](peer-review.claude-code-opus-5.5.md), not another independent-model review or human approval. Both status logs, handoffs, current requirements/decisions/questions, sibling requirements/source-selector design and the earlier same-model reviews were read.

No residual critical, major or minor contradiction found within the ten requested dispositions.

1. **Listing volume:** [U2](questions.md) now states one row per unthreaded request, potentially thousands, and offers per-file listing or direct-input-only alternatives requiring an amended contract. Decision 1 retains its one-listing-per-request consequence; Q19 records the refinement. [8.5](../../requirements.md#8.5) measures request/row counts and rendered list-fragment bytes using a generated large unthreaded batch. This exposes cost without inventing a portable speed threshold or answering U2.

2. **Inclusive latest:** [7.6](../../requirements.md#7.6), [8.1](../../requirements.md#8.1) and Decision 11 preserve [approved sibling 4.4](../../../apsis-shared-json/requirements.md#4.4). The suggested source-only filter is explicitly rejected; the sibling design's source selector rejects `latest`. Unsupported follow/raw-JSON errors identify the selected source and give disable-discovery/explicit-session remedies, without skipping to older evidence. U1 retains the discovery opt-in choice; disclosure does not settle that policy.

3. **Parameters:** [2.9](../../requirements.md#2.9) preserves remaining top-level fields, complete client metadata and integration headers in shared JSON and non-conversation Markdown/HTML details, including stream/tool-name summaries. Values and truncation markers remain logged evidence, expressly without field-specific redaction. Decision 12, U1 and [7.5](../../requirements.md#7.5) disclose the exposure.

4. **Unsupported endpoints:** Non-Goals names `/v1/decisions`; [4.6](../../requirements.md#4.6) retains endpoint/position metadata and requires body-free per-path count warnings for unsupported POSTs, without inventing inference sessions. Known non-inference endpoints remain excluded; exact rules and warning aggregation scope are deferred explicitly to design.

5. **Client provenance:** The public-reference list pins Codex 0.166.0 metadata source. [3.1](../../requirements.md#3.1)/[3.7](../../requirements.md#3.7), Q23 and parameter retention cover side/compaction membership, parent identifiers and fully recorded request kind. Truncated metadata stays raw/marked; thread membership proves neither dialogue-turn status nor response ownership.

6. **Detection:** [1.1](../../requirements.md#1.1) prevents Kiro-like body markers overriding the valid bounded LM Studio signature and suggests forced parsing when headers lack that signature. [1.2](../../requirements.md#1.2) forces discovered matching files through log interpretation, retaining access to rotated notice-only evidence. [8.3](../../requirements.md#8.3) covers collisions and notice-only excerpts.

7. **Zones:** [5.1](../../requirements.md#5.1)–[5.5](../../requirements.md#5.5) and Decision 13 specify portable IANA availability without system data, explicit override interpretation, optional colon prefixes and Unix/Windows native absolute TZif forms. Invalid/unloadable overrides warn and retain wall time without fabricated instants; U8 remains open.

8. **Attribution notice:** [2.4](../../requirements.md#2.4)/Q28 replaces repetitive absence wording with one human transcript notice and per-request shared-JSON unknown state. Log-level events remain inspectable under [4.2](../../requirements.md#4.2).

9. **Browser commands:** [4.2](../../requirements.md#4.2)/[7.1](../../requirements.md#7.1) distinguish all-projects, source-selected logical-session opening—including oversized errors—from forced direct-file conversion requiring CLI access to server files. No browser file route is promised.

10. **Identity:** [3.4](../../requirements.md#3.4)/Q26 explicitly equates in-root canonical-file request/membership IDs with discovery without reading other bodies. Outer direct-log IDs and out-of-root/stdin membership remain input-scoped; a containing file is not promoted to one listed conversation.

U1–U8 remain explicitly unanswered and all active LM Studio ADRs proposed. Before requirements approval, which discovery, listing, body-visibility, attribution, history, linking, ordering and zone-default alternatives does the human choose? This check supplies no policy answer, design/tasks admission or implementation evidence.

Verification is limited to document consistency, references, numbering, Markdown/whitespace and unchanged protected-input hashes. No renewed public-source verification, private logs/machine data, implementation tests, code/canonical-task edits, commits, pushes or branch changes.
