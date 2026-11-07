# Critique (revision 2): claude-code

Design-critic review of revision 2; decisions in `decision_log.md` taken as settled; code checked in this worktree.

## Critical

**C1 (2.7, 4.2). Exec-stream identity collides with rollouts and depends on the listing scope.**
- De-duplication applies only where the rollout "is also listed". Rollouts list by exact `cwd` match (`lister.go` `listCodex`), streams by containing repository (2.6). A run started in a subdirectory lists as `codex-exec` in one view and `codex` in another.
- Opening ignores scope: `findCodexSession` searches all of `~/.codex/sessions` and `resolveInput` tries `codex` first, so an identifier equal to `thread_id` opens the rollout, not the listed stream. The CLI takes no source, so 4.2's "by its source and identifier" has no CLI form.
- "Stays the same while the file exists" rules out suffixes assigned by order. Whether `codex exec resume` reuses `thread_id` is unchecked.

Fix: de-duplicate whenever a rollout with that `thread_id` exists, whatever the scope. Derive any suffix from the file itself. State how the CLI selects a source, or require that exec identifiers never equal a rollout ID. Verify `exec resume` before design.

## Major

**M1 (1.1, 2.2–2.4, 5.2). The field list, the public contract, does not match the rest of the document or the code.**
- 2.2 needs an exit code and 2.3 an incomplete marker; 1.1 has neither, and `ContentItem` has only `IsError`.
- 2.4 keeps only the event type; Q11's reason (consumers see every event) needs the raw payload.
- Kiro IDE cost (`ParseResultMetadata.TotalCost`, `CostUnit`) is shown in Markdown but missing from 1.1.
- Parse warnings go only to stderr (5.2), yet listing JSON carries warnings (4.3); a consumer cannot detect a partial parse.
- Session-level "model" is undefined when the model changes mid-session (Copilot `session.model_change`; `Entry.Model` is per entry).

Fix: make 1.1 complete: exit code, incomplete state, generic-event payload, cost with unit, parse warnings with line numbers. Define the session model (first, last, or all distinct) or keep it per entry.

**M2 (2.2, 1.4). Token usage semantics are unverified.** Whether `turn.completed.usage` is per turn or a running total is unchecked, yet 2.2 shows it "with its turn" and consumers will sum it. "Every usage field" is an open set; a closed schema fails 1.4 when Codex adds a key, as with `cache_write_input_tokens`. Entries have no turn concept. Fix: verify first; label usage per turn or cumulative; let the schema accept unknown usage keys; name the entry that carries it.

**M3 (1.1, 4.3). Exec-stream creation time has no source.** Events carry no timestamps; modification time marks the end of a run and resets on copy (Q16). Listing order and `apsis latest` depend on it. Fix: state the source and mark it file-derived in JSON, or omit it.

**M4 (2.5, 4.1, 4.3). Listing cost and candidate files are unbounded.** Each listing walks configured directories at any depth, finds a repository root per stream, and across projects reads project path, model and provider from content (Claude folder names do not decode to paths). `apsis serve` repeats this every 15 seconds (`sessions.html`). A directory holding several projects, as Decision 3 proposed, walks `.git` and `node_modules` and lists this feature's own 5.6 fixtures as sessions. Link handling during the walk is unspecified; a stream reached through a link outside the directory lists but fails 5.5. Fix: follow links only within the configured directory (as `walkDirFollowSymlinks` does); add a time budget or cache; state which directories are skipped.

## Minor

- **2.6:** "that project's listing" is undefined; other sources match `normalizePath(cwd) == normalizePath(projectPath)`, so a subdirectory listing shows no streams. Fix: list a stream when its nearest repository or worktree root equals the project path.
- **5.1:** the environment variable alone changes `apsis --list` and `apsis latest`; `apsis latest -F` then fails on a `codex-exec` session (`runLatest`). Say so.
- **2.1:** "first event is `thread.started`" misses runs that fail before a thread starts, and differs from `detectJSONLFormat`, which skips unparseable lines. Define both cases.
- **4.3:** names no surface (`apsis --list`, or serve's `/api/sessions`, which returns HTML) and no schema file for listings. Also define `apsis latest` across projects (4.1).
- **Raw `-f json`:** fails on exec streams today (`DetectFormat`); state it emits events like other JSONL.
- **5.4:** no separator for several directories; "empty means none" adds nothing without a default.
- **File input:** `resolveInput` takes the ID from the file name; use `thread_id`.

## Revision-1 findings still open

- r1 M3 and part of M4 (listing cost, candidate files): M4.
- r1 M6, in part (event and usage shapes): M1, M2.
- r1 minors on creation time (M3) and content reads in listings (M4).
- r1 C3 as narrowed by the peer review (open under the listing's scope), and the peer review's unverified items: C1, M2.

The rest are resolved.

## Questions for the human

1. Should a run with both a rollout and a stream list once, under `codex`, in every scope?
2. How should the CLI choose a source when identifiers match: a prefix, an option, or exec identifiers that never equal a rollout ID?
3. Is file modification time acceptable as an exec stream's creation time, marked as such, or should it be omitted?
4. Should the listing walk skip `.git`, `node_modules` and `testdata`?
5. For `apsis serve` across all projects: a listing time budget, or a cache?
6. Should session JSON carry parse warnings and Kiro IDE cost?
