# Decision Log: Apsis — LM Studio Server Logs

## Quick Decisions

| ID | Date | Decision | Rationale |
|----|------|----------|-----------|
| Q1 | 2026-11-05 | Preserve the user-directed separate LM Studio spec, common representation, renderer parity and upstream independence | The sibling's accepted Decisions 5/7 and both handoffs establish this direction |
| Q2 | 2026-11-05 | Agent proposal: use source, forced format and provider `lmstudio`, without a local/cloud flag | Names the mechanism while preserving the sibling's accepted provider-only policy |
| Q3 | 2026-11-05 | Agent proposal: include the seven exact inference endpoints in requirements 2.1 | Public LM Studio documentation confirms both v0 completion endpoints; exact matching excludes token counting and management |
| Q4 | 2026-11-05 | Agent proposal: use the existing 65,536-byte detection bound, require an included request header, and retain forced parsing | A generic log header does not distinguish LM Studio; startup-heavy excerpts remain explicitly convertible |
| Q5 | 2026-11-05 | Agent proposal: reject raw JSON and follow mode for LM Studio, directing raw JSON users to shared JSON | The text disk log is neither the existing raw JSONL format nor the follow parser's line format |
| Q6 | 2026-11-05 | Agent proposal: retain physical line ranges, partial/oversized states and body-free warnings in CLI, shared JSON and web | Recovery must preserve later requests without presenting unreadable bodies as complete or leaking content through diagnostics |
| Q7 | 2026-11-05 | Agent proposal: never concatenate an incomplete record across files | Supplied rotation reports establish no continuation grammar; unknown bytes must not become another request's body |
| Q8 | 2026-11-05 | Agent proposal: count only session-owned records toward logical-session size | The existing browser limit must not count other sessions merely sharing the log file |
| Q9 | 2026-11-05 | Agent proposal: synthetic fixtures only, with separate provenance for log grammar and public payload documentation | The repository must contain no private log content, identifiers, paths or served model names |
| Q10 | 2026-11-05 | Correct the handoff's monthly-volume and project-inference claims; qualify unverified grammar assertions | Aggregate-only file sizes disprove a monthly rate; prompt content is not authoritative project metadata |
| Q11 | 2026-11-05 | The merge records agent proposals and unanswered user choices; no LM Studio requirements approval is claimed | Continuous-work authorisation permits useful local progress, not invented human answers |

All full decisions below are **proposed**. The merge weighs both drafts and their non-author cross-reviews; it does not promote draft proposals to human-approved decisions. Routine technical choices can be admitted for local work under `SPEC-DRIVE.md` after the coordinator records their review evidence. The repository has no `.agentic.json`, so future decision revisions use superseding entries.

---

## Decision 1: Group Only Explicit Client Threads

**Date**: 2026-11-05
**Status**: proposed

### Context

Reported Codex request bodies contain `client_metadata.thread_id`, while other requests lack identifiers. Context can shrink, unrelated clients can send the same prefixes, and truncated strings can make different original inputs appear equal.

### Decision

Group exact nonempty string thread identifiers within one configured root, including across files, models and endpoints. Keep requests without that identifier as singleton sessions, and preserve other identifiers only as evidence. User question U2 covers broader grouping.

### Rationale

Recorded client identity survives compaction without treating text similarity as proof of a shared conversation. Stable identifiers must derive from recorded identity and stable file locations, not traversal order or the earliest currently retained request.

### Alternatives Considered

- **Prefix, time or model grouping:** rejected for the baseline because it can merge independent clients, particularly after log truncation.
- **Cache keys, session fields or response-chain grouping:** rejected for the baseline because the inspected reports do not establish general conversation-identity semantics or complete response links.

### Consequences

**Positive:**

- Thread sessions survive context changes and rotation.
- Adding older files or deleting other files does not rename surviving sessions.

**Negative:**

- Unidentified multi-turn clients produce one listing per request.
- Additional client identity conventions need separately justified support.

---

## Decision 2: Keep Request Snapshots and History References

**Date**: 2026-11-05
**Status**: proposed

### Context

Repeated history can make a log appear to contain the same conversation many times. Global deduplication or suppressing assistant context equal to a logged response can also erase intentional repetitions or obscure provenance.

### Decision

Retain request boundaries and complete parsed input sequences in shared JSON. In human-readable views, reference only an identical leading sequence of logged items from the previous request in the same session, and mark changed or shortened history as a context discontinuity. User question U5 retains full visible snapshots as an alternative.

### Rationale

The reader can distinguish resent context from recorded generation while reconstructing any request. Truncated text equality means only that logged representations match; it establishes neither original content equality nor identity.

### Alternatives Considered

- **Render every snapshot as ordinary new turns:** rejected for the baseline because replayed context looks like new conversation activity; labelled full snapshots remain U5's alternative.
- **Globally deduplicate messages or suppress equal responses:** rejected because equal text at a different position or with different provenance is separate evidence.

### Consequences

**Positive:**

- Routine history repetition is reduced without losing request reconstruction.
- Compaction and repeated requests remain visible.

**Negative:**

- Renderers need request boundaries and history references.
- Human-readable requests can depend on earlier sections for their repeated prefix.

---

## Decision 3: Require Evidence for Request Attribution

**Date**: 2026-11-05
**Status**: proposed

### Context

Model-tagged notices and responses can interleave, and requests can outlive a rotated file. The supplied observations of adjacent validation warnings establish a pattern, not a guarantee that a later record belongs to the immediately preceding request.

### Decision

Attribute a response or diagnostic only through a unique explicit recorded link. Keep all other included log-level content in direct log transcripts; logical-session views expose body-free counts and source references to contributing files and their successors without suggesting ownership. U3 and U4 ask about visibility and any separately labelled inferred-attribution mode.

### Rationale

No response should become a claimed answer or failure for the wrong request. References preserve access to evidence without copying unrelated clients' bodies into a logical session; file scope is navigation context, not causal evidence.

### Alternatives Considered

- **Attach by adjacency or active-request counters:** rejected because concurrency, missing completion lines and rotation provide counterexamples even when sampled logs look serial.
- **Copy unmatched bodies into every file-sharing session:** rejected because it exposes unrelated content and multiplies output.
- **Drop unmatched records:** rejected because errors and responses would disappear without disclosure.

### Consequences

**Positive:**

- The baseline makes no unsupported request-response claim.
- Direct log inspection retains unattributed evidence across every renderer.

**Negative:**

- Most reported response and warning records may remain unattributed.
- A session reader must inspect referenced files for full diagnostics, including later rotated files.

---

## Decision 4: Separate Wall Time From Assumed Instants

**Date**: 2026-11-05
**Status**: proposed

### Context

The reported disk grammar carries local wall time without a zone. During daylight-saving transitions, ordering or monotonicity cannot establish which offset an isolated timestamp had.

### Decision

Preserve wall time and source order, use the process zone (`TZ`, otherwise host local) as an explicit assumption, and emit an offset-bearing instant only when that interpretation is unique. Leave invalid, ambiguous or nonexistent instants and corresponding creation times absent. U8 asks whether an explicit zone should instead be mandatory.

### Rationale

This supports local inspection using the existing process setting while avoiding fabricated precision. File and line order remains reproducible regardless of timestamp interpretation.

### Alternatives Considered

- **Silently assume UTC or select a daylight-saving offset:** rejected because the source does not establish that instant.
- **Infer offset from monotonicity:** rejected because a log excerpt or missing records can contain only the second occurrence of a repeated hour.
- **Require a new dedicated zone option:** rejected for the baseline because `TZ` already provides a process-scoped override; mandatory explicit selection remains U8's alternative.

### Consequences

**Positive:**

- Wall-time evidence is preserved and the conversion assumption is visible.
- Repeated hours do not reorder requests or create invented timestamps.

**Negative:**

- Copied logs require a correct zone override for useful interpreted times.
- Some sessions lack an absolute creation time and sort under the shared unknown-time policy.

---

## Decision 5: Gate Standard Discovery Behind All Projects

**Date**: 2026-11-05
**Status**: proposed

### Context

LM Studio has a reported standard server-log directory, and logs may include full prompts from every server client. Existing single-project listings should not parse unrelated server traffic.

### Decision

Discover the standard root only in explicit all-projects scope, with CLI/environment override and empty-value disable. Gate identifier opening with the same scope, keep listings metadata-only, and disclose server-wide traffic in help and the existing network warning. U1 retains explicit LM Studio opt-in as the alternative.

### Rationale

This keeps standard-location discovery consistent with existing sources while making the expanded listing scope deliberate. Root confinement applies before candidate reads; explicit file conversion remains an independent user choice.

### Alternatives Considered

- **Require a configured root for all discovery:** rejected for the baseline because it adds setup despite a standard location; retained as U1's user choice.
- **Read logs in ordinary project listings:** rejected because the source records no project and this would add unrelated traffic and parsing work.
- **Refuse non-loopback service:** rejected because it would override an explicitly selected bind policy beyond this parser's scope.

### Consequences

**Positive:**

- Default project listings retain their existing behaviour and avoid log-body reads.
- All-projects users can discover sessions without an extra root setting.

**Negative:**

- All-projects users may discover traffic they did not expect.
- The existing `0.0.0.0` warning does not cover every possible externally reachable bind; general bind-policy changes remain outside this feature.

---

## Decision 6: Preserve Correlation Without Borrowing Projects

**Date**: 2026-11-05
**Status**: proposed

### Context

Reported thread identifiers permit consumers to compare server and client evidence. A request's prompt can quote a working directory, and several saved harness artifacts can share the same thread identifier.

### Decision

Expose the original thread identifier in transcript and listing metadata, keep server and harness sessions separately listed, and leave LM Studio's project unknown. Defer automatic web links and borrowed project attribution; U6 asks whether explicit-match links are wanted.

### Rationale

Correlation metadata does not establish an authoritative server-side project. Leaving linking separate also avoids an implicit selection rule between copied streams, resumed streams and rollouts.

### Alternatives Considered

- **Extract projects from environment-context prompts:** rejected because those are client content and may be truncated or quoted examples.
- **Borrow a matching harness session's project:** rejected for the baseline because correlation and project ownership are different assertions and multiple matches need a policy.
- **Merge or suppress the server transcript:** rejected because server-received context and client events are distinct evidence.

### Consequences

**Positive:**

- Consumers retain exact recorded correlation evidence.
- Project-scoped behaviour remains independent of client-specific prompt parsing.

**Negative:**

- Users cannot navigate to matching harness pages automatically in the baseline.
- Server work remains outside project-specific listings even when a likely client record is available.

---

## Decision 7: Keep One Direct Log Document Shape

**Date**: 2026-11-05
**Status**: proposed

### Context

One disk file can contain several interleaved logical sessions and unattributed events. The approved sibling requires one shared session-document structure for all sources.

### Decision

Convert a supplied file or stdin into one source-ordered log transcript using the common outer document, with logical-session membership on requests and log-level events where applicable. Identifier-based opening selects a logical session. U7 retains grouped sections within that same structure as a presentation alternative.

### Rationale

Direct inspection preserves physical evidence and handles stdin without another lookup. Multiple logical sessions do not require an input-dependent object/array switch or a second LM Studio JSON contract.

### Alternatives Considered

- **Emit an array for files and one object for identifiers:** rejected because it violates the common outer-shape contract.
- **Require a session identifier for direct input:** rejected because excerpts and stdin must remain inspectable before discovery.
- **Treat every request as the same conversation:** rejected because input containment does not establish session identity.

### Consequences

**Positive:**

- Consumers handle one outer session shape for every input.
- Unattributed events have a truthful home in direct views.

**Negative:**

- Direct views can interleave logical sessions.
- Additive schema fields must distinguish log-transcript identity from logical-session membership.

---

## Decision 8: Cache Unchanged Logs Within the Process

**Date**: 2026-11-05
**Status**: proposed

### Context

Grouping needs fields inside request bodies, and the existing web list refreshes every 15 seconds. The drafts' cold-time target relied on one machine and a mistaken monthly-volume claim, so it cannot serve as a portable pass/fail threshold.

### Decision

Require zero log-body reads on unchanged in-process refreshes and reflect added, changed, replaced, truncated or removed inputs on the next successful refresh. Write no persistent index; record generated workload and environment in cold and refresh benchmarks instead of imposing an uncalibrated time limit.

### Rationale

The refresh outcome is measurable and directly addresses repeated work. Cache mechanics, record limits and identifiers belong to design and must be tested together before local implementation admission.

### Alternatives Considered

- **Read every log body on each refresh:** rejected because unchanged inputs provide no new evidence.
- **Persist an index on disk:** rejected for this feature because it introduces stored identifiers and cache-format lifecycle without demonstrated need.
- **Require three seconds for a fixed byte volume everywhere:** rejected because hardware, record count and payload shape affect runtime and no reference environment was agreed.

### Consequences

**Positive:**

- Repeated web refreshes avoid unchanged body reads.
- No additional copy of log metadata is persisted by apsis.

**Negative:**

- Each new CLI process still pays cold parsing cost.
- Replacement and removal invalidation must update affected history and references while preserving surviving identities.
