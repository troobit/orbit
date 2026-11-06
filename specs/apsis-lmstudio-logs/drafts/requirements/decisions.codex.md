# Proposed Decisions: Apsis — LM Studio Server Logs

These proposals accompany `draft.codex.md`. They are not human approvals and do not revise the sibling spec's accepted decisions; merge, review and approval follow `SPEC-DRIVE.md`.

## Quick Decisions

| ID | Date | Decision | Rationale |
|----|------|----------|-----------|
| Q1 | 2026-11-04 | Propose source and forced-format name `lmstudio`, with provider `lmstudio` | Matches the handoff and distinguishes server evidence from the client harness transcript without claiming locality |
| Q2 | 2026-11-04 | Propose additive shared JSON fields, retaining the sibling schema version | The sibling spec establishes versioning and a common representation; endpoint, request, attribution and provenance metadata need no replacement schema |
| Q3 | 2026-11-04 | Propose explicit errors for raw `-f json` and `-F` with this input | Disk logs are neither the JSONL expected by raw output nor the Claude-shaped lines expected by follow mode |
| Q4 | 2026-11-04 | Propose CLI/environment settings for log root and time zone; defer exact names to design | Preserves the sibling configuration approach without introducing a config file |
| Q5 | 2026-11-04 | Propose no cross-file byte concatenation for unfinished bodies | No supplied evidence establishes that a rotated file can begin with the continuation of a previous request body; mark incompleteness rather than combining unrelated input |
| Q6 | 2026-11-04 | Propose unchanged refreshes read zero log-body bytes after the initial listing | An observable caching outcome addresses the existing 15-second web refresh without inventing a hardware-dependent time budget |
| Q7 | 2026-11-04 | Propose wholly synthetic fixtures with log-grammar and documented-payload origins labelled separately | Preserves the verified structure without publishing machine data or claiming documented endpoints were observed in captures |

---

## Decision 1: Group Only Explicit Client Threads

**Date**: 2026-11-04
**Status**: proposed

### Context

The handoff establishes that Codex traffic carries `client_metadata.thread_id` and that its input history can shrink. Other clients omit the identifier; equality of model or prompt does not establish a shared conversation.

### Decision

Group exact nonempty thread identifiers within a configured root across files. Make each remaining request a singleton session. Preserve the client identifier for consumer correlation without merging the corresponding harness transcript.

### Rationale

This uses recorded evidence and remains deterministic through compaction. A source namespace and safe stable identifiers let every session open through existing CLI and web routes.

### Alternatives Considered

- **Time windows or matching message prefixes**: Rejected because parallel clients and compaction make both grouping and separation unreliable.
- **Group by cache keys, session fields or response chains immediately**: Rejected for this draft because their general identity semantics or logged response availability have not been established. Uniquely logged response chains remain a human scope question.

### Consequences

**Positive:**

- Thread sessions survive endpoint/model changes and file rotation without heuristic merges.
- Consumers can correlate independently retained server and client evidence.

**Negative:**

- Clients without the supported identifier produce many singleton sessions.
- Equivalent identifier fields used by another client need separately justified support.

---

## Decision 2: Retain Requests and Reference Repeated Prefixes

**Date**: 2026-11-04
**Status**: proposed

### Context

Inference requests often repeat conversation history. Rendering every snapshot as new messages duplicates conversations, while compaction or branching means equal text cannot always identify one message occurrence.

### Decision

Keep every request boundary. Reference only the identical leading input prefix shared with the preceding request, preserve other items, mark non-extension histories as discontinuities and retain enough JSON information to reconstruct each request's input.

### Rationale

The comparison is explicit and testable. It reduces routine repetition without treating text equality as a global message identity or claiming that context messages were logged model output.

### Alternatives Considered

- **Render every full snapshot as ordinary conversation turns**: Rejected because it misrepresents replayed history as newly occurring messages. Labelled complete snapshots remain an approval option.
- **Globally deduplicate equal messages or splice overlapping tails**: Rejected because deliberate repetition and compaction can remove or misorder real evidence.

### Consequences

**Positive:**

- Repeated prefixes appear once while requests remain individually inspectable.
- Compaction does not change session identity or silently erase context changes.

**Negative:**

- Renderers and the schema need explicit history references or an equivalent recoverable representation.
- A discontinuous history can repeat older material outside the identical prefix.

---

## Decision 3: Preserve Ambiguous Records at File Scope

**Date**: 2026-11-04
**Status**: proposed

### Context

Parallel slots can interleave model-tagged notices and responses with no request identifier. A timestamp, nearby request or matching model tag does not prove ownership.

### Decision

Require an explicit unique recorded match for request attribution. Expose unmatched responses, warnings and errors as labelled log-level events in direct views and every session with requests in the same source file, with a shared event identity.

### Rationale

This preserves errors without inventing a causal link. File scope is reproducible even when the log records no reliable start/end interval for concurrent inference.

### Alternatives Considered

- **Attach to the nearest request or last matching model**: Rejected because concurrent calls can acquire each other's warnings or output.
- **Discard unmatched records or expose them only through file input**: Rejected as the recommendation because opening a listed session could conceal the server errors needed to understand it; direct-only exposure remains a human choice.

### Consequences

**Positive:**

- No guessed response becomes a claimed answer for a request.
- Server diagnostics remain accessible from session views and every output.

**Negative:**

- Many responses may remain unattributed even during apparently serial traffic.
- A session can expose unrelated file-level content; the draft asks the human to choose this exposure explicitly at approval.

---

## Decision 4: Keep Projects Unknown and Time Assumptions Explicit

**Date**: 2026-11-04
**Status**: proposed

### Context

The inspected log grammar has a zone-free wall time and no authoritative project field. Some client prompts contain working directories, but those strings describe request content rather than server metadata.

### Decision

List LM Studio only in all-projects mode with unknown project paths. Default time interpretation to the host zone, permit an explicit IANA zone and label the assumption; preserve source order and omit absolute timestamps when wall times are ambiguous, nonexistent or invalid.

### Rationale

The default permits local inspection without inventing project identity. An explicit assumption and an unknown-time outcome distinguish the source's evidence from a conversion setting.

### Alternatives Considered

- **Extract project paths from environment-context prompt blocks**: Rejected because their semantics are client-specific, can change within a request and can be ordinary quoted text.
- **Treat wall times as UTC or silently choose a daylight-saving offset**: Rejected because either creates a precise instant absent from the evidence.
- **Require an explicit zone and LM Studio opt-in for every use**: Not recommended because local inspection already has a host zone and all-projects is explicit; stricter exposure and time policies remain human questions.

### Consequences

**Positive:**

- Default project listings do not acquire unrelated server traffic.
- Ordering survives daylight-saving ambiguity and is independent of timestamp interpretation.

**Negative:**

- Logs copied between machines need a zone override for equivalent absolute times.
- Ambiguous timestamps remain unknown, and project-specific discovery is unavailable even for identifiable Codex clients.
