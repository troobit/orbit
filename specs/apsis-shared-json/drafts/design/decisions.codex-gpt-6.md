# Proposed Design Decisions: Apsis Shared JSON

These proposals accompany `draft.codex-gpt-6.md`. IDs are local to this draft; the merge step assigns IDs after the feature's existing decisions. Approved requirements and human policy answers remain binding.

## Quick Decisions

| ID | Date | Decision | Rationale |
|---|---|---|---|
| Q1 | 2026-11-05 | Use `-f shared-json`, `--all-projects`, `--source`, and repeated `--codex-exec-dir` | Names distinguish output, scope, storage source and a specific wire mechanism |
| Q2 | 2026-11-05 | Read directories from `APSIS_CODEX_EXEC_DIRS` using `filepath.SplitList`; CLI repetitions override the full list | Handles platform syntax while allowing separator characters inside individual flagged paths |
| Q3 | 2026-11-05 | Publish one version-1 schema with session/listing discriminators under `docs/schemas/` | Both JSON surfaces validate through shared definitions; no source-specific outer shape |
| Q4 | 2026-11-05 | Preserve unknown payloads with `json.RawMessage` and arbitrary numbers with `UseNumber` | Avoids inventing a generic shape or losing numeric precision during exec decoding |
| Q5 | 2026-11-05 | Scope internal tool IDs by turn/item and preserve original item IDs separately | Existing grouping maps key by tool ID and would otherwise collide on resumed item numbering |
| Q6 | 2026-11-05 | Locate repository roots through filesystem git markers, without invoking git | Apsis retains its absence of external CLI dependencies |
| Q7 | 2026-11-05 | Maintain source-specific private locators and stat-dependent summary caches, including Kiro WAL/journal dependencies | Global resolution needs exact storage locations; unchanged refreshes must avoid transcript-body reads |
| Q8 | 2026-11-05 | Resolve unknown/removed locators with one refreshed lookup before returning not found | Covers deletion between listing and opening without an unbounded retry |
| Q9 | 2026-11-05 | Leave stdin artifact ID and unrecorded timestamps absent | No filesystem evidence exists for stdin; the recorded thread ID remains usable correlation metadata |

---

## Decision 1: Project Public JSON From Shared Entries

**Date**: 2026-11-05
**Status**: proposed

### Context

Current entries carry Claude-oriented tags and fields excluded from JSON. Both human renderers already consume those entries, while the public schema must include the new exec states, relationships and warnings.

### Decision

Extend the common entry model for new evidence, then explicitly project it into public document types. Encode those types independently of Markdown/HTML grouping and length limits.

### Rationale

Parsers produce one interpretation that all renderers consume. A separate projection gives the public contract stable names without changing raw source output.

### Alternatives Considered

- **Marshal current entries directly:** rejected because their JSON tags and omitted model fields are unsuitable.
- **Implement a separate JSON parser path:** rejected because it would duplicate translation and could disagree with human output.
- **Replace every parser and renderer with new public types immediately:** rejected because it expands unrelated changes and makes compatibility harder to verify.

### Consequences

**Positive:**
- New evidence reaches all output formats through the same parser.
- Public JSON and existing wire tags can evolve independently.

**Negative:**
- Projection needs explicit coverage when common entries gain fields.
- Internal and external types both need maintenance.

---

## Decision 2: Retain Private Locators in a Shared Catalogue

**Date**: 2026-11-05
**Status**: proposed

### Context

Kiro CLI opening requires a directory key; Kiro IDE opening requires the discovered workspace and execution-detail location. Global discovery currently discards these locators, and web requests rebuild listings repeatedly.

### Decision

Introduce a catalogue that publishes immutable summary/locator snapshots and owns metadata caches. CLI and web operations use the same scoped catalogue for list/latest/source-ID opening; existing constructors remain compatible.

### Rationale

Opening a listed item must use the actual discovered storage location. Snapshot ownership also supplies a concrete lifetime for unchanged-refresh caches.

### Alternatives Considered

- **Pass an empty project into existing resolvers:** rejected because Kiro resolution needs information not recoverable from that value.
- **Encode filesystem paths in public IDs or URLs:** rejected because public routes require safe bounded identifiers and should not expose storage locators.
- **Build separate CLI and web indexes:** rejected because source selection and invalidation could diverge.

### Consequences

**Positive:**
- Listing and opening share source, scope and storage evidence.
- Concurrent web requests can use a consistent snapshot.

**Negative:**
- A catalogue adds lifecycle and invalidation responsibilities.
- Companion-file dependencies must be explicit for each source.

---

## Decision 3: Hash Thread and Canonical Path for Stream IDs

**Date**: 2026-11-05
**Status**: proposed

### Context

A thread can have copied or resumed streams, while appends and overlapping discovery roots must not rename existing files. The approved requirements treat each file as a distinct artifact until a rollout suppresses its listing.

### Decision

Use `exec-` followed by full SHA-256 over length-delimited thread ID and canonical file path. Preserve the thread ID separately and detect conflicting digest identities before listing.

### Rationale

The result is bounded and route-safe, derives from both identity inputs, and does not depend on root order, traversal or file contents.

### Alternatives Considered

- **Thread ID with a traversal suffix:** rejected because newly added files can rename prior artifacts.
- **Content digest:** rejected because appending events changes the identifier.
- **Raw absolute path:** rejected because it is unsafe in routes and exposes storage paths.

### Consequences

**Positive:**
- Copies have separate stable identifiers while appends retain identity.
- Canonical symlink aliases and overlapping roots do not multiply sessions.

**Negative:**
- Moving a file changes its identifier.
- Opaque IDs need a separately visible thread identifier for correlation.

---

## Decision 4: Validate the Schema With a Test-Only Library

**Date**: 2026-11-05
**Status**: proposed

### Context

The contract requires actual schema conformance tests for every source and listing shape. The module has no JSON Schema engine, and custom checks would validate only the subset their author remembered.

### Decision

Use `github.com/santhosh-tekuri/jsonschema/v6` only in tests, with the checked-in Draft 2020-12 schema and network loading disabled. Runtime output uses standard-library JSON encoding. Implementation pins the dependency version before use; this spec run installs nothing.

### Rationale

The [maintainer's API documentation](https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v6) provides local schema resources, compilation and format assertions. A real validator checks the published schema independently of the encoder.

### Alternatives Considered

- **Handwritten shape assertions alone:** rejected because they can disagree with the published schema.
- **External Node/Python validator command:** rejected because tests would gain an additional runtime/toolchain requirement.
- **Runtime validation of every output:** rejected because fixture/schema tests satisfy the requirement without adding runtime cost and dependencies.

### Consequences

**Positive:**
- Tests validate the artifact consumers actually use.
- The shipped binaries gain no schema-validator import.

**Negative:**
- The Go module gains a test dependency and its transitive graph.
- Implementation must pin and review the chosen library version.

---
