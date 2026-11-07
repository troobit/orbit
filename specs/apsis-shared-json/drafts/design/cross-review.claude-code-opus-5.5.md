# Cross-review: `draft.codex-gpt-6.md`

Reviewer `claude-code-opus-5.5`, fresh session. I wrote neither draft, but I share harness and model with the author of `draft.claude-code-opus-5.5.md`, so this review is not independent of that draft. Every preference for it below cites code or pinned Codex 0.166.0 source. Read: both drafts and decision proposals, approved requirements, `decision_log.md`, current code, pinned `exec_events.rs` and JSONL emitter. I read `critique.fresh-context.md` only after forming these findings, and did not read `design.md` or `tasks.md`.

"Codex" means `draft.codex-gpt-6.md`; "Claude" means `draft.claude-code-opus-5.5.md`.

## Which version to keep

| # | Item | Keep | Evidence |
|---|---|---|---|
| 1 | Extend `Entry`, project separately | Either | Same decision (Codex D1, Claude P1) |
| 2 | Envelope | Codex: metadata in a `session` object, reused by listing rows | One `$defs` entry serves both documents; 4.3 repeats 1.1's fields |
| 3 | Time fields | Claude: `created_at` or `file_modified_at` | Every listing fallback is `ModTime()` (`lister.go` 168, 171, 230, 316, 443), so Codex's `sort_time.basis` has one value |
| 4 | Entry fields | Codex's smaller v1 set | Claude publishes Claude-only `uuid`, `parent_uuid`, `meta`, `tool_use_result`; 1.1 needs none, and Q13 allows adding them later |
| 5 | Warnings | Codex `code` plus Claude's position unit | Kiro IDE stores a chat index as `Line` (`kiro_ide_parser.go:81`) |
| 6 | Tool IDs | Codex: qualify by turn, keep `item_id` | The emitter numbers `item_N` from 0 per process (emitter lines 86, 100–101), so appended resumes reuse `item_0`; `resultMap` and `toolMeta` key by tool ID transcript-wide (`grouping.go:161`, `markdown.go:463`) |
| 7 | Tool results | Claude: every exec tool item emits one | Non-Task tools render only with their result (`markdown.go:512`); an incomplete command would vanish |
| 8 | Item translation | Claude's table and `web_search` note | Codex never names the blocks for file changes and searches (2.2). `ThreadItem.id` is flattened with `WebSearchItem.id` (`exec_events.rs` 98–101, 298–299): two `id` keys |
| 9 | Oversized lines | Codex: warn, skip, continue | Scanners stop at 10 MiB (`parser.go:390`) |
| 10 | Provider and grouping | Claude's parity table | Group and user headers pass an empty model (`markdown.go` 197, 232, 311, 363; `html.go` 291, 334, 425, 483); exec tools are never Read/Edit. Codex's new group boundaries change existing output (5.1) |
| 11 | HTML visibility | Claude: both passes | `hasContent` returns early (`html.go:518`) |
| 12 | Rendering new content | Claude's table | Codex specifies none |
| 13 | Opening architecture | Claude: long-lived `Lister` cache, resolver branches | CLI opens without listing (`main.go` 405–424); Codex's catalogue makes `apsis --source … <id>` build a cold snapshot of every source. A whole-pass mutex suffices at the 15 s poll (`sessions.html:21`) |
| 14 | Cold listing reads | Claude: bounded heads | Codex's "one initial body parse" makes each `apsis --list` parse every body; current readers stop early (`lister.go:568`) |
| 15 | Kiro CLI opening | Claude: `GetSessionByID` | `GetSession` re-normalises keys through `Abs`, `Clean`, `EvalSymlinks` (`path.go` 69–80); deleted directories miss |
| 16 | Suppression | Claude: filename-UUID set | `findCodexSession` resolves by filename UUID (`resolver.go` 386–410), which defines "resolvable" (2.7) |
| 17 | Exec identifier | Claude: `sanitise(thread)[:239]-<16 hex>` | Both meet 2.8; Claude's shows the thread (Q16) within the 256-character route limit (`middleware.go:31`). Codex's is opaque |
| 18 | Settings | Codex: empty CLI value clears; reject mixed values; reject `--source` with file, stdin, list, latest | Lets the CLI override a set environment variable; `-s` is free |
| 19 | Schema tests | Claude's closed test copy, apsis-owned objects only | An open schema cannot fail on undeclared emitted fields; P4 as written also closes payloads, inputs and usage counters, forbidding their arbitrary keys. `golang.org/x/text` is already indirect (`go.mod:52`) |
| 20 | Cache dependencies | Codex: include the rollback journal; never serve stale metadata for a failed changed file | Complements Claude's `(size, mtime, ctime, inode)` key |

## Conflicts for the user

- **A. Web parse warnings.** Codex shows them for all sources; the view shows none today (`handlers.go` 39–97), so 5.1 breaks. Recommend exec only (Claude U1).
- **B. Claude-session model.** `Message` keeps only role and content, so the parser drops `message.model`. Claude's listing would show a model its session document lacks; filling `Entry.Model` changes Claude headers (`markdown.go:425`), against 5.1. Options: omit for Claude in v1 (recommended), JSON only, or amend 5.1.
- **C. Kiro IDE `modelProvider`** exists (`kiro_ide_types.go:21`); 3.1 and 5.1 limit provider to Codex. Recommend a follow-up (Claude U3).
- **D. Test-only validator in `go.mod`.** Both drafts want it; upstream carries it (Claude U2).

## Gaps in both drafts

1. Raw `-f json` drops a BOM-prefixed first event: `convertToJSON` scans unstripped bytes (`main.go` 763, 774). Strip the leading BOM first (critique item 8).
2. The raw path also aborts above 10 MiB; apply item 9 there.
3. `convert` returns before output on empty entries (`main.go:701`); dispatch shared JSON before it.
4. Neither reconciles all-projects with Q9. Recommend flag only, matching serve's `--project`, which has no variable (`main.go:72`).
5. Claude's P6 says Kiro IDE ties differ between lister and resolver; they agree in effect (name-sorted `ReadDir`, first tie kept). Fix the rationale; sharing one function remains right.
6. `handlers.go:118` reverses the returned slice in place; caches must hand out copies (critique item 6).
