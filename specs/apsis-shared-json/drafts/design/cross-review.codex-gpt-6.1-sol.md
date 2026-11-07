# Cross-review: Claude design draft

Reviewer: `codex-gpt-6.1-sol`. Reviews [Claude's draft](draft.claude-code-opus-5.5.md) and [decision proposals](decisions.claude-code-opus-5.5.md) against approved requirements revision 3 and repository code. The Codex draft is excluded: its `codex-gpt-6` author is this harness/model. This supplies the non-author review of the Claude draft only; it does not replace Claude's review of Codex.

No critical findings. Resolve the following major findings before their affected implementation slices.

## Major

1. **Scope tool links through rendering, not just parsing (2.2–2.3, 5.7).** P5 correctly keys lifecycle records by turn, but the draft then emits bare item IDs as tool IDs. [BuildToolMeta](../../../../internal/transcript/markdown.go), HTML tool metadata and [preprocessEntries](../../../../internal/transcript/grouping.go) use whole-transcript maps keyed only by ID. Two resumed turns using `item_0` can display the second command beside the first result. Keep P5's lifecycle ordering; derive internal tool IDs from turn ordinal plus item ID, retain the original separately, and test two different commands with reused IDs in JSON and all human renderers.

2. **Complete provider propagation (3.1, 5.1).** Reject the provider parity table's exemption for groups and other metadata sites. Read/Edit groups and standalone HTML tool-result sections bypass the ordinary assistant header; the latter has no originating entry available. Carry recorded provider/model through group and tool metadata and prevent incompatible provider groups from merging. Preserve existing output when provider is absent. Test provider-bearing Read/Edit groups, tool-only entries and deferred results, including provider without model.

3. **Cache every metadata dependency and bound its promises (4.2–4.3, 5.9).** Keep stat-based caching, but P3 omits rollback journals, Kiro execution details and repository-marker changes. Companion-only edits must invalidate summaries and opening locators. Kiro IDE metadata is nested in a JSON document, so a generic bounded JSONL head scan is insufficient. Specify source-specific extraction, dependency fingerprints and omission/warning on changed unreadable inputs; test each companion independently. Reject the proposed directory-mtime optimisation: in-place edits to children do not update the parent's mtime. Any SQLite fallback query must be measured against zero unchanged body reads rather than assumed exempt.

4. **Detection must reach the promised first event (2.1, 5.2).** [DetectFormat](../../../../internal/transcript/parser.go) reads only 64 KiB. Adding a branch in `detectJSONLFormat` cannot find `thread.started` beyond a long malformed prefix or beyond that buffer. Discovery's separate 50-line/1-MiB cap can also reject a directly convertible stream. Use a shared streaming probe with replay of consumed input, retaining physical line numbers and defining oversized-line recovery. Test malformed prefixes exceeding both current caps without changing existing format detection.

5. **Do not close arbitrary payload maps in schema tests (1.3–1.4, 2.4, 2.9).** P4's recursive `additionalProperties:false` transform can forbid unknown usage counters and arbitrary tool/event keys. Keep the maintained test-only validator, but validate the published schema directly; restrict any stricter field audit to apsis-owned structural objects, explicitly leaving maps/payloads open. [JSON Schema's object rules](https://json-schema.org/understanding-json-schema/reference/object) explain the closure behaviour. Include future nested fields, a large integer and invalid known-field types. Decode new arbitrary exec inputs with `UseNumber` or retain raw JSON to avoid float rounding before projection.

6. **Suppress by verified rollout identity (2.7, 5.8).** The filename UUID set proposed for suppression differs from the recorded `session_meta.payload.id` and can include unreadable or malformed rollouts. [findCodexSession](../../../../internal/sessions/resolver.go) matches filenames without validating that metadata. Build the global suppression index from validated, openable rollout identities before project filtering; define handling for mismatched filenames. Test malformed/unreadable matches and filename/recorded-ID disagreement alongside different-project cases.

7. **Canonical identity cannot depend on spelling or traversal (2.5, 2.8).** P2 acknowledges duplicate IDs for case aliases on case-insensitive filesystems, contradicting canonical-file deduplication. Its short path hash also resolves collisions by dropping whichever path sorts later, making availability depend on added copies. Prefer a full digest of unambiguously encoded thread ID and canonical path; define filesystem alias normalisation and report conflicting identities without choosing an arbitrary winner. Test root permutation and case aliases where supported.

## Minor

8. **Separate technical admission from policy questions.** U2's test-only validator is a reversible implementation choice eligible for reviewed local admission. Keep U1/U3's compatibility-preserving defaults; broadening warnings or provider behaviour remains separate user-owned scope. No question should reopen approved global rollout precedence, filesystem fallback, directory skips, provider-only reporting or the no-prompt-file decision.

## Merge prerequisites

Specify scoped tool links, provider propagation, dependency invalidation, detection limits, open payload schemas and validated rollout identity before admitting those slices. Claude's cross-review must cover the Codex draft; the owner must finish or release protected design/tasks outputs before merge. Neither this review nor passing checks constitute human approval.
