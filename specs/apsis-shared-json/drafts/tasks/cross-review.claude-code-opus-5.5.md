# Cross-review: `draft.codex-gpt-6.1-sol.md`

Reviewer `claude-code-opus-5.5`, fresh headless session. I wrote neither tasks draft, but I share harness and model with the author of `draft.claude-code-opus-5.5.md`, so this review is not independent of that draft. Each preference for it cites the design, handoff, tasks format or code. Read: both drafts and decision files, the 18:48Z self-review (not a non-author review), requirements, design, decision log, design questions, handoff and code; not the in-progress r2 review.

"Codex" is this draft; "Claude" is the other.

## Mechanical checks

`rune list` (local dev build, scratch copy) parses 13 pending, unowned stream-1 tasks. Its JSON has 17 `blockedBy` edges, all pointing to lower IDs, so there is no cycle. All 31 criteria are linked, and all 38 referenced paths exist relative to `drafts/tasks/`.

## Findings

**Critical**

1. **Tasks 6–13: the Windows gate blocks work the design leaves ungated.** The design gates only the cache (§4 "this cache slice", Decision 23 "cache consumers", risk 3). Codex chains each task to its predecessor, so all-projects listing, exec parsing and discovery (tasks 8–13; 2.1–2.11, 4.1, 4.2, 4.4) wait for native Windows evidence. No one here can produce it: CI tests only on `ubuntu-latest` (`ci.yml:12`), yet `release.yml` ships Windows binaries. Task 6 needs only the lint baseline, but it waits for task 5. Fix:
   - block task 6 on task 1 only;
   - split 8–9 into an all-projects pair blocked by task 5 and a cache pair blocked by task 7 and exec discovery, so the cache covers all six collectors (design §4 table);
   - block task 10 on task 5;
   - move task 11's `ParseResolved` exec branch and web views into 12–13, because task 9 creates `ParseResolved`.

   Claude's order already does this.

**Major**

2. **Tasks are oversized, and the plan has no PR slices.** Task 8 spans tests in four packages: all-projects resolution, Kiro tie order, ordering properties, zero-read caching, WAL, the mutex, two benchmarks, CLI flags and serve. Tasks 12 and 13 each link 23 criteria. A red task failing for a dozen reasons is the "big jump in complexity" the format forbids. The handoff asks for small, ordered PRs, but Codex splits slice 2 between task 5 and task 9. Fix: adopt Claude's phases and parents 5–8, 12–14 and 15–16, keeping Codex's test cases.

**Minor**

3. **Links and references** resolve only from `drafts/tasks/`. Existing tasks files use root-relative references and `requirements.md#N.M`; rewrite on merge.
4. **Process state.** The front-matter `executable`/`freeze`/`mode`, task 1's "after the freeze is lifted" and task 13's "No current execution is authorized" contradict the 19:00Z note. Keep only task 1's mapping to its canonical ID and owner.
5. **No CHANGELOG or README work**, which the handoff requires in each PR (`CHANGELOG.md:8`). Fold it into each slice's last green task.
6. **Kiro CLI project in project scope.** `DB.GetSession` returns only an `io.Reader` after trying two keys (`session.go:28`). The resolver therefore cannot report the matched key, which the design's metadata table needs. Add this to task 9.
7. **Kiro discovery cannot be counted.** `listKiro("")` calls `logs.DiscoverAll` directly (`lister.go:341–342`), bypassing `kiroDiscover`. Task 8's discover count needs an injection point.
8. **Windows adapter.** `x/sys` v0.43.0 has `GetFileInformationByHandleEx` and its class constants, but no `FILE_ID_INFO` or `FILE_BASIC_INFO` structs, and `go.mod:50` marks the module indirect. Task 7 should:
   - define the structs;
   - make the requirement direct;
   - require `GOOS=windows go vet`, because `make lint` on Unix never compiles that file.
9. **Scope slips.**
   - Task 4 omits `latest -f shared-json` (design §7).
   - The all-projects text project column (4.1) arrives only in task 13.
   - The serve `--all-projects`/`--project` usage error (Q31) is untested.
10. **Task 3 adds `Options`/`WithCodexExecRoots` before anything uses them.** Ordering is first used in task 9 and roots in task 13. Introduce each with its first consumer.

## Which version to keep

| Item | Keep | Why |
|---|---|---|
| Dependencies, phases, granularity | Claude | Findings 1–2 |
| Fingerprint package | Codex (design placement) | Claude TD4 changes the design; `go test -run` isolates these tests on Windows |
| Identifier-conflict test | Codex (injected digest) | A real SHA-256 collision cannot be built, so Claude 13.1 needs this hook |
| Schema introduction | Claude TD2, if gap A is fixed | Each branch ships with its producer (finding 10) |
| Streams, WAL test placement, task 1, detection/lifecycle/Q62/suppression cases | Either | Equivalent, or the design names no placement |
| Links, CHANGELOG, Kiro key, discover hook, x/sys | Claude | Findings 3 and 5–8 |

## Conflicts for the user

- **Native Windows verification.** Neither draft can finish the cache without a Windows host. Options:
  - run the fingerprint tests on a Windows machine before the cache PR (recommended);
  - add a `windows-latest` CI job (the maintainers' call);
  - narrow 5.9 on Windows.

## Gaps in both drafts

- **A. New block types.** The design's block `oneOf` on `type` rejects unlisted types, but Q13 treats additions as compatible within version 1. Neither draft tests an unknown block type. Fix: add a fallback branch for unlisted types and a test, or record that a new type raises the version. Send this to the design as a note.
- **B. No `prerequisites.md`.** The tasks format requires one for steps only a human can do. The native Windows run is such a step, and it blocks the cache's green task.
- **C. No benchmark threshold.** Risks 1 and 4 name fallbacks but no trigger. The merge should record one; Claude G2 proposes 1.5 s.
