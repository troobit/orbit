# Decision Log: apsis-shared-json tasks draft (claude-code-opus-5.5)

Planning decisions behind [draft.claude-code-opus-5.5.md](draft.claude-code-opus-5.5.md). They are agent proposals, not human approvals. IDs are local to this draft (`TQ` quick, `TD` full) so they cannot be confused with the spec's Q1–Q62 and Decisions 1–24; the tasks merge assigns final numbers.

Inputs: [requirements.md](../../requirements.md) (revision 3, approved), [design.md](../../design.md) and [decision_log.md](../../decision_log.md) through Q62, [questions.md](../design/questions.md), and the code at upstream `7259a6c` plus this worktree's uncommitted test edits. Not read: the other tasks draft, its decisions and cross-review, canonical `tasks.md`, and `drafts/design/tasks.codex-gpt-6-unlogged.md`. The status log and `handoff.md`, which the process requires reading, summarise the other draft's shape (13 tasks, one stream, six red/green pairs, a Windows gate before cache consumers); nothing more of it was seen.

## Quick Decisions

| ID | Date | Decision | Rationale |
|----|------|----------|-----------|
| TQ1 | 2026-11-05 | Include a baseline `make lint` task (task 1) | `make lint` fails on two unchecked receives in `follow_test.go`, so no later task can pass its checks; canonical `tasks.md` already tracks this as task 1 owned by `codex-gpt-6`, and the merge maps this task onto that ID and owner |
| TQ2 | 2026-11-05 | CHANGELOG and README updates ride in the last implementation subtask of each phase | Upstream wants `[Unreleased]` entries per PR; the tasks format excludes documentation-only tasks |
| TQ3 | 2026-11-05 | Requirement links and front-matter references use the canonical location (`requirements.md#N.M`) | The local rune build writes `requirements.md` links whatever `requirements_file` says (checked in a scratch file); the links resolve once merged beside `requirements.md`, not inside `drafts/tasks/` |
| TQ4 | 2026-11-05 | Follow recommendation A for U1–U3; tasks 2 and 3 name what changes under another answer | U1–U3 are unanswered; `questions.md` says recommendations guide planning |
| TQ5 | 2026-11-05 | Resolver metadata fields (`Project`, `Path`, `TimeFromFilesystem`) land in PR 1; listing-row fields in PR 2 | Session documents opened by identifier need the Copilot and Kiro CLI project, and must not present a modification-time fallback as `created_at` (Decision 24) |
| TQ6 | 2026-11-05 | `sessions.ParseResolved` lands in PR 3 | Its only new branch forces `codex-exec`; PR 1 changes no forced format, and the web view emits no shared JSON (non-goal) |
| TQ7 | 2026-11-05 | Verify design risk 2 (Kiro WAL writes change companion fingerprints) in the fingerprint gate, task 15 | The answer decides the Kiro cache key, so it belongs ahead of the cache with the other platform facts |
| TQ8 | 2026-11-05 | Each parent has a red subtask then a green subtask; cross-task blockers and the stream are repeated on the red subtask | Checked with the local rune build: children default to stream 1 and do not inherit their parent's blockers, so per-stream readiness needs both |
| TQ9 | 2026-11-05 | Benchmarks for design risks 1 and 4 are recorded measurements that feed task 16, not pass/fail tests | The design names no threshold (gap G2) |

---

## Decision TD1: Phases Follow Upstream PR Slices, With the Cache as a Fourth Slice

**Date**: 2026-11-05
**Status**: proposed

### Context

`handoff.md` suggests three PRs: shared JSON with the provider, listing across projects, and exec streams. Decision 23 later gated the cache on native Windows fingerprint tests, and CI runs on Ubuntu only. Requirement 5.9 covers every source's collector, not only exec streams.

### Decision

Use four phases, one per PR: shared JSON sessions and the provider; listing across projects; exec streams; the unchanged-refresh cache (fingerprint gate plus cache).

### Rationale

A gate waiting for a Windows host must not hold back the exec PR. The cache changes every collector, so it is its own reviewable change. Each phase keeps existing output intact (5.1) and can merge alone.

### Alternatives Considered

- **Cache inside the exec PR**: matches the handoff's three slices - Rejected: the Windows gate would block exec streams.
- **Cache inside the listing PR**: earlier 5.9 - Rejected: the gate would block cross-project listing, and the cache would be reworked when the exec collector arrives.
- **No phases**: simpler file - Rejected: hides the PR boundaries the handoff asks for.

### Consequences

**Positive:**
- PRs 1–3 can merge without a Windows host.
- The gate's outcome affects one PR.

**Negative:**
- 5.9 is unmet until PR 4; between PRs 3 and 4, serve refreshes re-probe exec stream heads under configured roots (opt-in).
- One more PR than the handoff suggested.

---

## Decision TD2: Introduce Each Schema Branch and Content Kind With Its Producer

**Date**: 2026-11-05
**Status**: proposed

### Context

The design defines complete version 1 documents. Exec block types and fields have no producer before the exec parser, and listing documents none before the listing work.

### Decision

PR 1 publishes the `session` branch for existing content kinds. PR 2 adds the `listing` branch. PR 3 adds `turn`, `item_id`, `status`, `incomplete`, `exit_code`, `thread_id` and the `plan`, `error`, `event` and `usage` blocks, together with their Markdown and HTML rendering. `schema_version` stays 1 (1.3, Q13).

### Rationale

Every schema part arrives with code and tests that exercise it, and new kinds render in all three outputs in the same task, as the user's direction requires.

### Alternatives Considered

- **Whole schema and all content types in PR 1**: one contract change - Rejected: types and projection branches with no producer, which maintainers review without a use.
- **Raise the version per slice**: explicit - Rejected: additive changes do not raise the version (1.3).

### Consequences

**Positive:**
- No unused types or schema branches in any PR.

**Negative:**
- A consumer validating newer documents against an older copy of the schema rejects the new block types; the `$id` points at `main`, which limits this.
- The schema file changes in three PRs, so tasks 6 and 10 are serialised.

---

## Decision TD3: Three Streams Split by File Ownership

**Date**: 2026-11-05
**Status**: proposed

### Context

`SPEC-DRIVE.md` allows agents to implement disjoint files in this worktree concurrently. After PR 1, exec parsing and rendering touch `internal/transcript` and `internal/sharedjson`, while cross-project listing touches `internal/sessions`, `cmd/apsis` and `internal/apsisweb`. The fingerprint gate needs a different host for its native run.

### Decision

Stream 1: tasks 1–4 and 9–11. Stream 2: tasks 5–8, 12–14 and 16. Stream 3: task 15, with no blockers. Cross-stream blockers serialise shared files: 5 after 4 (`sessions` types and resolver), 10 after 6 (schema and `sharedjson`), 12 after 8 (`cmd/apsis/main.go`).

### Rationale

Exec parsing can proceed while listing work runs, and the gate can start at once.

### Alternatives Considered

- **One stream**: no coordination - Rejected: queues the exec parser and the Windows gate behind unrelated listing work.
- **One stream per PR**: mirrors phases - Rejected: PRs share `cmd/apsis/main.go` and `internal/sessions`, so they would edit the same files concurrently.

### Consequences

**Positive:**
- Two agents can work without editing the same file.

**Negative:**
- Stream 2 tests compile `internal/transcript`; an agent mid-edit in stream 1 can break stream 2's `make test` until its package compiles again.
- Cross-stream blockers must be respected when claiming.

---

## Decision TD4: Put the Fingerprint Adapters in Their Own Package

**Date**: 2026-11-05
**Status**: proposed (changes the design's placement table)

### Context

The design places `fingerprint_unix.go` and `fingerprint_windows.go` in `internal/sessions`, which stream 2 edits throughout. `internal/transcript/fileinfo_*.go` already reads inode and size for follow mode, but returns inode 0 on Windows and has no change time or device.

### Decision

Create `internal/fileidentity` with one interface and Unix and Windows adapters; the `sessions` cache consumes it.

### Rationale

Stream 3 never edits a package another stream is changing, the platform calls stay in one small package, and the native Windows run needs only that package.

### Alternatives Considered

- **Design placement in `internal/sessions`**: fewer packages - Rejected: concurrent edits to a package stream 2 is changing.
- **Extend `transcript`'s `getFileInfo`**: reuse - Rejected: it is follow-mode code and its Windows variant returns no identity.

### Consequences

**Positive:**
- The gate can be built, reviewed and run natively on its own.

**Negative:**
- One package the design does not list; the merge must amend the placement table or revert to it.

---

## Gaps and Proposed Changes for the Merge

- **G1**: TD4 changes the design's placement of the fingerprint adapters.
- **G2**: Design risks 1 and 4 give no threshold for their fallbacks. Proposal: apply the fallback when a warm 10,000-file pass (risk 1) or a cold multi-source listing (risk 4) takes more than 1.5 s, a tenth of the 15 s poll interval.
- **G3**: Nothing in the repository can run the native Windows tests; CI runs on Ubuntu and releases cross-compile. Options for the human: (A) run `go test ./internal/fileidentity/...` on a Windows host before PR 4, recorded as a prerequisite that blocks task 16 (recommended for local work); (B) add a `windows-latest` CI job, which is the maintainers' call; (C) narrow 5.9 on Windows, a requirements change.
- **G4**: In project scope the Kiro CLI project is the database key that matched, but `logs.(*DB).GetSession` returns only a reader; the design specifies the key only for `GetSessionByID`. Task 4 makes `GetSession` report it.
- **G5**: `listKiro("")` calls `logs.DiscoverAll` directly, bypassing the injectable `kiroDiscover`; the counting discover in task 16 needs an injection point for it.
- **G6**: `golang.org/x/sys/windows` has `GetFileInformationByHandleEx` and the `FileIdInfo`/`FileBasicInfo` class constants but no `FILE_ID_INFO` or `FILE_BASIC_INFO` structs, so the adapter defines them; `golang.org/x/sys` becomes a direct requirement. `golang.org/x/sys/unix` `Stat_t` names the change time `Ctim` on both Linux and macOS, unlike `syscall`.

## Self-Check

1. **Red/green**: 15 parents each hold an adjacent failing-test subtask then an implementation subtask; task 1 is test-only.
2. **rune**: `rune list` parses the file (local `dev` build): 4 phases, 16 parents, 46 items, 3 streams, every `Blocked-by` resolves to a stable ID, and every edge points to a lower task number, so there is no cycle.
3. **Coverage**: all 31 criteria are referenced; script-checked against the anchors in `requirements.md`.
4. **No orphans**: every task depends on task 1 except task 15, which only task 16 consumes.
5. **Non-goals**: no follow mode for exec streams or shared JSON (both rejected with errors), no JSON HTTP endpoint, no prompt files, LM Studio, local flag, OpenCode, or usage extraction for other sources; raw `-f json` changes only by stripping a BOM from exec input.
6. **Granularity**: phases are PR boundaries; streams are justified in TD3.
7. **Risks**: risk 1 is benchmarked in 13.1; risks 2 and 3 in task 15, which blocks 16; risk 4 in 16.1; assumption 5 (two `id` keys in `web_search`) in the 9.1 fixtures, which every later exec task depends on.
